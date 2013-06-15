
package site

import (
	"github.com/shopsmart/mgo/bson"
	"github.com/shopsmart/mgo"

	"os"
	"log"
	"io/ioutil"
	"encoding/json"
	"path/filepath"
	"fmt"
	"sort"
	"time"
	"reflect"
)

type menuM struct {
	bson.M
	path []*menuM
	ptr *menuM
}

func (m *menuM) arr(a string) (ret []*menuM) {
	ret = []*menuM{}
	_ret, ok := m.M[a]
	if !ok {
		return
	}
	_ret2, ok2 := _ret.([]interface{})
	if ok2 {
		for _, r := range _ret2 {
			_ret3, ok3 := r.(map[string]interface{})
			if ok3 { ret = append(ret, &menuM{M:bson.M(_ret3)}) }
		}
	}
	if false {
		log.Println(reflect.TypeOf(_ret).String())
		log.Println(len(ret))
	}
	return
}

func (m *menuM) Childs() (ret []*menuM) {
	ret = m.arr("child")
	return
}

func (m *menuM) IsDir() bool {
	return m.str("type") == "dir"
}

func (m *menuM) Url() string {
	return m.str("url")
}

func (m *menuM) str(name string) (ret string) {
	ret, _ = m.M[name].(string)
	return
}

func (m *menuM) Id() string {
	id := m.str("id")
	return id
}

func (m *menuM) Title() string {
	return m.str("title")
}

func (m *menuM) Path() (ret []*menuM) {
	return m.path
}

func (m *menuM) findid(matchid string) {
	m.path = []*menuM{m}
	id := m.str("id")
	if id == matchid {
		m.ptr = m
	}
	for _, c := range m.Childs() {
		c.findid(matchid)
		if c.ptr != nil {
			m.ptr = c.ptr
			m.path = append(m.path, c.path...)
			return
		}
	}
	return
}

func (m *menuM) load(filename string) (err error) {
	var f *os.File
	f, err = os.Open(filename)
	if err != nil {
		log.Println("menu:", "open", filename, "failed")
		return
	}
	dec := json.NewDecoder(f)

	tree := bson.M{}
	err = dec.Decode(&tree)
	if err != nil {
		log.Println("menu:", "decode", filename, "failed")
		return
	}
	tree["title"] = "根目录"
	*m = menuM{M:tree, path:[]*menuM{m}}
	return
}


type menuS struct {
	fetched bool
	id int
}

func (mu *menuS) qry(opts... interface{}) *mgo.Query{
	m := bson.M{}
	if mu.fetched {
		m["fetched"] = bson.M{"$exists": true}
	}
	for i := 0; i < len(opts); i += 2 {
		m[opts[i].(string)] = opts[i+1]
	}
	return C("videos").Find(m)
}

func (mu *menuS) cat_dfs(seq []string, mustcat bool, at int, title string, qry... interface{}) (ret bson.M) {
	ret = bson.M{}
	child := []bson.M{}

	q := mu.qry(qry...)
	n, _ := q.Count()

	if (mustcat && n < 20) || at == len(seq) {
		nodes := []bson.M{}
		q.Sort("-createtime").All(&nodes)
		for _, n := range nodes {
			child = append(child, bson.M{
				"type": "url",
				"url": n["url"],
				"title": n["title"],
				"id": fmt.Sprint(mu.id),
			})
			mu.id++
		}
	} else {
		cats := []string{}
		q.Distinct(seq[at], &cats)
		sort.Sort(sort.Reverse(sort.StringSlice(cats)))
		for _, c := range cats {
			child = append(child, mu.cat_dfs(
				seq, mustcat, at+1, c, append(qry, seq[at],c)...,
			))
		}
		q = mu.qry(append(qry, seq[at], bson.M{"$exists": false})...)
		nodes := []bson.M{}
		q.All(&nodes)
		for _, n := range nodes {
			child = append(child, bson.M{
				"type": "url",
				"url": n["url"],
				"title": n["title"],
				"id": fmt.Sprint(mu.id),
			})
			mu.id++
		}
	}

	ret["type"] = "dir"
	ret["title"] = title
	ret["child"] = child
	ret["id"] = fmt.Sprint(mu.id)
	mu.id++
	return
}

func (mu *menuS) jilu() (ret bson.M) {
	ret = mu.cat_dfs([]string{"cat1"}, false, 0, "纪录片", "cat","jilu")
	return
}

func (mu *menuS) zongyi() (ret bson.M) {
	ret = mu.cat_dfs([]string{"series", "cat1", "cat2"}, false, 0, "综艺", "cat","zongyi")
	return
}

func (mu *menuS) news() (ret bson.M) {
	ret = mu.cat_dfs([]string{"play"}, true, 0, "新闻", "cat","news")
	return
}

func (mu *menuS) jiaoyu() (ret bson.M) {
	ret = mu.cat_dfs([]string{"cat1"}, false, 0, "教育", "cat","jiaoyu")
	return
}

func (mu *menuS) yule() (ret bson.M) {
	ret = mu.cat_dfs([]string{}, false, 0, "娱乐", "cat","yule")
	return
}

func (mu *menuS) tiyu() (ret bson.M) {
	ret = mu.cat_dfs([]string{}, false, 0, "体育", "cat","tiyu")
	return
}

func (mu *menuS) qiche() (ret bson.M) {
	ret = mu.cat_dfs([]string{}, false, 0, "汽车", "cat","qiche")
	return
}

func (mu *menuS) dianshi() (ret bson.M) {
	ret = mu.cat_dfs([]string{"series", "cat1", "cat2"}, true, 0, "电视剧", "cat","dianshi")
	return
}

func (mu *menuS) movie() (ret bson.M) {
	child := []bson.M{}
	qry := []interface{}{"cat","movie"}
	child = append(child, mu.cat_dfs([]string{"type"}, true, 0, "按类型", qry...))
	child = append(child, mu.cat_dfs([]string{"year"}, true, 0, "按年份", qry...))
	child = append(child, mu.cat_dfs([]string{"regions"}, true, 0, "按地区", qry...))
	ret = bson.M{"type":"dir", "title":"电影", "child":child, "id":fmt.Sprint(mu.id)}
	mu.id++
	return
}

func (mu *menuS) all() (ret bson.M) {
	child := []bson.M{}
	child = append(child, mu.yule())
	child = append(child, mu.news())
	child = append(child, mu.zongyi())
	child = append(child, mu.movie())
	child = append(child, mu.jilu())
	child = append(child, mu.jiaoyu())
	child = append(child, mu.tiyu())
	child = append(child, mu.qiche())
	child = append(child, mu.dianshi())

	ret = bson.M{}
	ret["type"] = "dir"
	ret["child"] = child
	return
}

func menu_oneshot() {
	log.Println("menu:", "oneshot")
	mu := &menuS{}
	ret := mu.all()
	b, _ := json.Marshal(ret)
	ioutil.WriteFile(filepath.Join("www", "all.json"), b, 0777)
	log.Println("menu:", "all", len(b), "bytes")

	mu = &menuS{fetched:true}
	ret = mu.all()
	b, _ = json.Marshal(ret)
	ioutil.WriteFile(filepath.Join("www", "fetched.json"), b, 0777)
	log.Println("menu:", "menu", len(b), "bytes")
}

func menu_loop() {
	log.Println("menu: loop starts")
	for {
		menu_oneshot()
		time.Sleep(time.Hour*4)
	}
}

