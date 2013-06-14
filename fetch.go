
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
	"strings"
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
	}, "timeout=", 10, "maxspeed=", 1024*100)

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

type fetchS struct {
	cat string
	n int
	flag string
}

func (m *fetchS) one() (url string) {
	ret := bson.M{}
	qry := canfetch()
	qry["cat"] = m.cat
	C("videos").Find(qry).Sort("-createtime").Limit(1).One(&ret)
	url = bson_getid(ret)
	return
}

func (m *fetchS) randser() (url string) {
	series := []string{}
	qry := canfetch()
	qry["cat"] = m.cat
	C("videos").Find(qry).Distinct("series", &series)
	if len(series) == 0 {
		return
	}
	ser := series[rand.Int()%len(series)]
	qry["series"] = ser
	ret := bson.M{}
	C("videos").Find(qry).Sort("-createtime").Limit(1).One(&ret)
	url = bson_getid(ret)
	return
}

func (m *fetchS) get() {
	url := ""
	if strings.Contains(m.flag, "one") {
		url = m.one()
	}
	if strings.Contains(m.flag, "ser") {
		url = m.randser()
	}
	if url == "" {
		log.Println("fetch: url empty")
		return
	}
	var maxdur time.Duration
	if strings.Contains(m.flag, "short") {
		maxdur = time.Minute*20
	}
	log.Println("fetch: downloading", url)
	download_one(url, "fetch", m.cat, maxdur)
}

func fetch_loop() {

	log.Println("fetch: loop starts")

	for {
		fetchs := []*fetchS {
			&fetchS{"news", 8, "one|short"},
			&fetchS{"zongyi", 1, "ser"},
			&fetchS{"yule", 5, "one|short"},
			&fetchS{"movie", 1, "ser"},
			&fetchS{"tiyu", 5, "one|short"},
			&fetchS{"jilu", 1, "ser"},
			&fetchS{"qiche", 5, "one|short"},
			&fetchS{"jiaoyu", 1, "ser"},
			&fetchS{"dianshi", 1, "ser"},
		}

		for _, f := range fetchs {
			for i := 0; i < f.n; i++ {
				log.Println("fetch:", f.cat, i+1, "/", f.n)
				f.get()
				time.Sleep(time.Second)
			}
		}
	}
}
