
package site

import (
	"github.com/go-av/curl"
	"github.com/go-av/fetcher"

	"github.com/shopsmart/mgo/bson"

	"time"
	"fmt"
	"os"
	"path/filepath"
	"errors"
	"math/rand"
	"log"
)

func strhash(in string) (r int) {
	for _, ch := range in {
		r = int(ch) + (r<<6) + (r<<16) - r
	}
	if r < 0 {
		r = -r
	}
	return
}

func canfetch() (bson.M) {
	return bson.M {
		"fetched": bson.M{"$exists":false},
		"error": bson.M{"$exists":false},	
	}
}

func download_one(url,prefix,tag string, maxdur time.Duration) {
	dir := fmt.Sprintf("%.8d", strhash(url)%1e8)

	path := filepath.Join(prefix, dir)
	os.Mkdir(path, 0777)

	filename := filepath.Join(path, "a.ts")

	err := fetcher.DownloadM3u8(url, filename,
	func (st fetcher.Stat) error {
		/*
		if st.Op != "downloading" {
			return nil
		}
		*/
		log.Println("fetch:", tag, dir, curl.PrettyDur(st.Dur),
				st.Io.Speedstr,
				curl.PrettyPer(st.Per), curl.PrettySize(st.Size))
		if maxdur != time.Duration(0) && st.Dur > maxdur {
			log.Println("fetch:", "video too long")
			return errors.New("video too long")
		}
		return nil
	}, "timeout=", 10, "maxspeed=", 1024*300)

	C("videos").Update(bson.M{"_id": url}, bson.M{
		"$set": bson.M {
			"fetched": dir,
			"url": "/fetch/"+dir+"/a.ts",
		},
	})

	log.Println("fetch: download end", err)
}

func bson_getid(b bson.M) (id string) {
	_id, ok := b["_id"]
	if ok {
		id, _ = _id.(string)
	}
	return
}

func fetchone_movie() (url string) {
	ret := bson.M{}
	qry := canfetch()
	qry["cat"] = "movie"
	C("videos").Find(qry).Sort("-createtime").Limit(1).One(&ret)
	url = bson_getid(ret)
	return
}

func fetchone_news() (url string) {
	ret := bson.M{}
	qry := canfetch()
	qry["cat"] = "news"
	C("videos").Find(qry).Sort("-createtime").Limit(1).One(&ret)
	url = bson_getid(ret)
	return
}

func fetchone_zongyi() (url string) {
	series := []string{}
	qry := canfetch()
	qry["cat"] = "zongyi"
	C("videos").Find(qry).Distinct("series", &series)
	if len(series) == 0 {
		return
	}
	ser := series[rand.Int()%len(series)]
	qry["series"] = ser
	ret := bson.M{}
	C("videos").Find(qry).Sort("-cat1", "-cat2").Limit(1).One(&ret)
	url = bson_getid(ret)
	return
}

func fetch_loop() {

	type fetchS struct {
		name string
		f func () (string)
	}

	log.Println("fetch: loop starts")

	for {
		fetchs := []fetchS {
			fetchS{"news", fetchone_news},
			fetchS{"zongyi", fetchone_zongyi},
			fetchS{"movie", fetchone_movie},
		}

		var f fetchS
		var url string
		if false {
			f = fetchs[rand.Int()%len(fetchs)]
		} else {
			for i := 0; i < len(fetchs); i++ {
				f = fetchs[i]
				url = f.f()
				if url != "" {
					break
				}
			}
		}

		log.Println("fetch: selecting", f.name)
		if url == "" {
			log.Println("fetch: url empty")
			time.Sleep(time.Second)
			continue
		}

		var maxdur time.Duration
		if f.name == "news" {
			maxdur = time.Minute*20
		}
		download_one(url, "fetch", f.name, maxdur)
		time.Sleep(time.Second)
	}
}
