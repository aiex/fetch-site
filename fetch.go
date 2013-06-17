
package site

import (
	"github.com/go-av/curl"
	"github.com/go-av/fetcher"

	"github.com/shopsmart/mgo/bson"

	"time"
	"os"
	"path/filepath"
	"errors"
	"math/rand"
	"strings"
	"log"
)

func canfetch() (bson.M) {
	return bson.M {
		"fetched": bson.M{"$exists":false},
		"error": bson.M{"$exists":false},
	}
}

func download_one(id,url,prefix,tag string, maxdur time.Duration) {
	dir := id

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
				st.Io.Speedstr,	curl.PrettyPer(st.Per), curl.PrettySize(st.Size),
				st.Stat, st.Op)

		if maxdur != time.Duration(0) && st.Dur > maxdur {
			log.Println("fetch:", "video too long")
			return errors.New("video too long")
		}
		return nil
	}, "timeout=", 10,
	//"maxspeed=", 1024*300,
	)

	if err != nil {
		return
	}

	C("videos").Update(bson.M{"_id": url}, bson.M{
		"$set": bson.M {
			"fetched": dir,
			"url": "/fetch/"+dir+"/a.ts",
		},
	})

	log.Println("fetch: download end", err)
}

type fetchS struct {
	cat string
	n int
	flag string
}

func (m *fetchS) one() (id,url string) {
	ret := bson.M{}
	qry := canfetch()
	qry["cat"] = m.cat
	//qry["id"] = bson.M{"$ne": ""}
	C("videos").Find(qry).Sort("-createtime").Limit(1).One(&ret)
	id = bson_getid(ret)
	url = bson_geturl(ret)
	return
}

func (m *fetchS) randser() (id,url string) {
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
	id = bson_getid(ret)
	url = bson_geturl(ret)
	return
}

func (m *fetchS) get() {
	url := ""
	id := ""
	if strings.Contains(m.flag, "one") {
		id,url = m.one()
	}
	if strings.Contains(m.flag, "ser") {
		id,url = m.randser()
	}
	if url == "" {
		log.Println("fetch: url empty")
		return
	}
	if id == "" {
		log.Println("fetch: id empty")
		return
	}
	var maxdur time.Duration
	if strings.Contains(m.flag, "short") {
		maxdur = time.Minute*20
	}
	log.Println("fetch: downloading", id, url)
	download_one(id, url, "fetch", m.cat, maxdur)
}

func fetch_oneshot() {
	log.Println("fetch: oneshot")

	f := &fetchS{"news", 1, "one"}
	f.get()
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
