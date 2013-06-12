
package site

import (
	"github.com/shopsmart/mgo/bson"

	"encoding/json"
	"io/ioutil"
	"fmt"
	"sort"
	"time"
)

func news_getone(tag string) (ret bson.M) {
	ret = bson.M{}
	ret["type"] = "dir"
	child := []bson.M{}

	arr := []bson.M{}
	err := gSess.DB("site").C("news").Find(bson.M{
		"fetched": bson.M{"$exists": true},
		"tag": tag,
	}).Sort("-createtime").Limit(40).All(&arr)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(tag, len(arr))

	for _, a := range arr {
		child = append(child, bson.M{
			"desc": a["title"],
			"type": "url",
			"url": a["_id"],
		})
	}
	ret["child"] = child

	return
}

func news_output() {
	list := map[string]string {
		"social": "社会新闻", 
		"tech": "科技新闻",
		"life": "生活新闻",
		"time": "时政新闻",
		"army": "军事新闻",
		"money": "财经新闻",
		"law": "法律新闻",
	}
	ret := bson.M{}
	ret["type"] = "dir"
	child := []bson.M{}
	for k,v := range list {
		a := news_getone(k)
		a["desc"] = v
		child = append(child, a)
	}
	ret["child"] = child
	b, _ := json.Marshal(ret)
	ioutil.WriteFile("news.json", b, 0777)
}

func movie_get_node(query bson.M, title string) (node jsonM) {
	q2 := gSess.DB("site").C("movie").Find(query)
	var n int
	n, _ = q2.Count()
	fmt.Println("  ", title, n)
	node = jsonM{"type":"dir", "title":title}
	child := []jsonM{}
	q2.Sort("-createtime").All(&child)
	for _, m := range child {
		child = append(child, jsonM{
			"type": "url",
			"url": m["_id"],
			"title": m["title"],
		})
	}
	node["child"] = child
	return
}

type jsonM map[string]interface{}

func movie_output() {
	ret_child := []jsonM{}

	q := gSess.DB("site").C("movie").Find(bson.M{})
	
	typs := []string{}
	q.Distinct("type", &typs)
	fmt.Println("by-type", len(typs))
	for _, t := range typs {
		node := movie_get_node(bson.M{
			"type": t,
		}, t)
		ret_child = append(ret_child, node)
	}

	tags := []string{}
	q.Distinct("tags", &tags)
	fmt.Println("by-tag", len(tags))
	bytag_child := []jsonM{}
	for _, tag := range tags {
		node := movie_get_node(bson.M{
			"tags": bson.M{"$in": []string{tag}},
		}, tag)
		bytag_child = append(bytag_child, node)
	}
	bytag := jsonM{"type":"dir", "title":"按类型", "child":bytag_child}
	ret_child = append(ret_child, bytag)

	years := []int{}
	q.Distinct("year", &years)
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	fmt.Println("by-date", len(years))
	bydate_child := []jsonM{}
	for _, year := range years {
		node := movie_get_node(bson.M{
			"year": year,
		}, fmt.Sprintf("%d", year))
		bydate_child = append(bydate_child, node)
	}
	bydate := jsonM{"type":"dir", "title":"按上映时间", "child":bydate_child}
	ret_child = append(ret_child, bydate)

	regions := []string{}
	q.Distinct("regions", &regions)
	fmt.Println("by-regions", len(regions))
	byregion_child := []jsonM{}
	for _, r := range regions {
		node := movie_get_node(bson.M{
			"regions": r,
		}, r)
		byregion_child = append(byregion_child, node)
	}
	byregion := jsonM{"type":"dir", "title":"按地区", "child":byregion_child}
	ret_child = append(ret_child, byregion)

	ret := jsonM{"type":"dir", "title":"电影", "child":ret_child}
	b, _ := json.Marshal(ret)
	ioutil.WriteFile("movies.json", b, 0777)
}

func menu_loop() {
	for {
		movie_output()
		time.Sleep(time.Second*60)
	}
}
