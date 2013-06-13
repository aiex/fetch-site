
package site

import (
	"github.com/shopsmart/mgo/bson"
	"github.com/shopsmart/mgo"

	"log"
	"io/ioutil"
	"encoding/json"
	"path/filepath"
	_ "fmt"
	"sort"
	"time"
)

func menuqry(opts... interface{}) *mgo.Query{
	m := bson.M{"fetched": bson.M{"$exists": true}}
	for i := 0; i < len(opts); i += 2 {
		m[opts[i].(string)] = opts[i+1]
	}
	return C("videos").Find(m)
}

func menu_cat_dfs(seq []string, at int, title string, qry... interface{}) (ret bson.M) {
	ret = bson.M{}
	child := []bson.M{}

	q := menuqry(qry...)
	n, _ := q.Count()

	if n < 20 || at == len(seq) {
		nodes := []bson.M{}
		q.All(&nodes)
		for _, n := range nodes {
			child = append(child, bson.M{
				"type": "url",
				"url": n["url"],
				"title": n["title"],
			})
		}
	} else {
		cats := []string{}
		q.Distinct(seq[at], &cats)
		for _, c := range cats {
			child = append(child, menu_cat_dfs(
				seq, at+1, c, append(qry, seq[at],c)...,
			))
		}
	}

	ret["type"] = "dir"
	ret["title"] = title
	ret["child"] = child
	return
}

func menu_zongyi_output() (ret bson.M){
	ret = menu_cat_dfs([]string{"series", "cat1", "cat2"}, 0, "综艺", "cat","zongyi")
	return
}

func news_getone(tag string) (ret bson.M) {
	ret = bson.M{}
	ret["type"] = "dir"
	child := []bson.M{}

	arr := []bson.M{}
	menuqry(
		"cat", "news",
		"cat1", tag,
	).Sort("-createtime").Limit(40).All(&arr)

	for _, a := range arr {
		child = append(child, bson.M{
			"title": a["title"],
			"type": "url",
			"url": a["url"],
		})
	}
	ret["child"] = child

	return
}

func menu_news_output() (ret bson.M) {
	list := map[string]string {
		"social": "社会新闻",
		"tech": "科技新闻",
		"life": "生活新闻",
		"time": "时政新闻",
		"army": "军事新闻",
		"money": "财经新闻",
		"law": "法律新闻",
	}
	ret = bson.M{}
	child := []bson.M{}
	for k,v := range list {
		a := news_getone(k)
		a["title"] = v
		child = append(child, a)
	}
	ret["title"] = "新闻"
	ret["type"] = "dir"
	ret["child"] = child

	return
}

func movie_get_node(title string, qry... interface{}) (ret bson.M) {
	//var n int
	//n, _ = q2.Count()
	ret = bson.M{"type":"dir", "title":title}
	child := []bson.M{}
	menuqry(qry).Sort("-createtime").All(&child)
	for _, m := range child {
		child = append(child, bson.M{
			"type": "url",
			"url": m["url"],
			"title": m["title"],
		})
	}
	ret["child"] = child
	return
}

func menu_movie_output() (ret bson.M) {
	ret_child := []bson.M{}

	typs := []string{}
	qry := []interface{}{"cat","movie"}
	menuqry(qry...).Distinct("type", &typs)
	//fmt.Println("by-type", len(typs))
	for _, t := range typs {
		node := movie_get_node(t, append(qry, "type",t)...)
		ret_child = append(ret_child, node)
	}

	tags := []string{}
	menuqry(qry...).Distinct("tags", &tags)
	//fmt.Println("by-tag", len(tags))
	bytag_child := []bson.M{}
	for _, tag := range tags {
		node := movie_get_node(tag, append(qry, "tags",tag)...)
		bytag_child = append(bytag_child, node)
	}
	bytag := bson.M{"type":"dir", "title":"按类型", "child":bytag_child}
	ret_child = append(ret_child, bytag)

	years := []int{}
	menuqry(qry...).Distinct("year", &years)
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	//fmt.Println("by-date", len(years))
	bydate_child := []bson.M{}
	for _, year := range years {
		node := movie_get_node(string(year), append(qry, "year",year)...)
		bydate_child = append(bydate_child, node)
	}
	bydate := bson.M{"type":"dir", "title":"按上映时间", "child":bydate_child}
	ret_child = append(ret_child, bydate)

	regions := []string{}
	menuqry(qry...).Distinct("regions", &regions)
	//fmt.Println("by-regions", len(regions))
	byregion_child := []bson.M{}
	for _, r := range regions {
		node := movie_get_node(r, append(qry, "regions",r)...)
		byregion_child = append(byregion_child, node)
	}
	byregion := bson.M{"type":"dir", "title":"按地区", "child":byregion_child}
	ret_child = append(ret_child, byregion)

	ret = bson.M{"type":"dir", "title":"电影", "child":ret_child}
	return
}

func menu_all_output() (ret bson.M) {
	ret = bson.M{}
	child := []bson.M{}
	ret["type"] = "dir"

	child = append(child, menu_news_output())
	child = append(child, menu_zongyi_output())
	child = append(child, menu_movie_output())

	ret["child"] = child
	return
}

func menu_loop() {
	log.Println("menu: loop starts")
	for {
		ret := menu_all_output()
		b, _ := json.Marshal(ret)
		ioutil.WriteFile(filepath.Join("www", "menu.json"), b, 0777)
		log.Println("menu:", "update", len(b), "bytes")
		time.Sleep(time.Second*60)
	}
}

