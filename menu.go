
package site

import (
	"github.com/shopsmart/mgo/bson"

	"strings"
	"os"
	"log"
	"encoding/json"
	"io"
	"path/filepath"
	"fmt"
	_"sort"
	"time"
)

type menuM2 struct {
	Type string `json:"type"`
	Title string `json:"title"`
	Url string `json:"url"`
	Id int `json:"id"`
	Cnt int `json:"cnt"`
	VidCnt int `json:"vidcnt"`
	Child []*menuM2 `json:"child"`
	parent *menuM2
	idinc int
	tab int
}

func (m *menuM2) Path() (ret []*menuM2) {
	ret2 := []*menuM2{}
	for p := m ; p != nil; p = p.parent {
		ret2 = append(ret2, p)
	}
	for i := len(ret2)-1; i >= 0; i-- {
		ret = append(ret, ret2[i])
	}
	return
}

func (m *menuM2) IsDir() bool {
	return m.Type == "dir"
}

func (m *menuM2) setparent() {
	for _, c := range m.Child {
		c.parent = m
		c.setparent()
	}
}

func (m *menuM2) load(r io.Reader) (err error) {
	dec := json.NewDecoder(r)
	err = dec.Decode(m)
	m.setparent()
	return
}

func (m *menuM2) dump(w io.Writer) {
	enc := json.NewEncoder(w)
	enc.Encode(m)
}

func (m *menuM2) dumptxt(w io.Writer) {
	if !m.IsDir() {
		return
	}
	space := ""
	for i := 0; i < m.tab; i++ { space += " " }
	fmt.Fprintln(w, space, "["+m.Title+"]", m.VidCnt, "videos")
	for _, c := range m.Child {
		c.tab = m.tab+1
		c.dumptxt(w)
	}
}

func (m *menuM2) find(id int) (ret *menuM2) {
	if m.Id == id {
		return m
	}
	for _, c := range m.Child {
		if c.Id <= id && id < c.Id + c.Cnt {
			return c.find(id)
		}
	}
	return
}

func (m *menuM2) insert(dirs []string, title,url string) {
	cur := m
	if cur.Child == nil {
		cur.Child = []*menuM2{}
	}
	for _, dir := range dirs {
		if dir == "" { dir = "更多" }
		var next *menuM2
		for _, c := range cur.Child {
			if c.Title == dir { next = c }
		}
		if next == nil {
			next = &menuM2{
				Title: dir,
				Type: "dir",
				parent: cur,
			}
			cur.Child = append(cur.Child, next)
		}
		cur = next
	}
	cur.Child = append(cur.Child, &menuM2{
		Title: title,
		Url: url,
		Type: "url",
		parent: cur,
	})
}

func (m *menuM2) setid(id int) (ret int) {
	if !m.IsDir() {
		m.VidCnt++
	}
	m.Id = id
	id++
	for _, c := range m.Child {
		id = c.setid(id)
		m.VidCnt += c.VidCnt
	}
	m.Cnt = id - m.Id
	return id
}

func menu2_get(filename string) {
	log.Println("menu:", filename, "starts")

	m := bson.M{}
	if strings.Contains(filename, "fetched") {
		m["fetched"] = bson.M{"$exists": true}
	}
	q := C("videos").Find(m)
	n, _ := q.Count()
	it := q.Iter()

	mu := &menuM2{Title:"根目录", Type:"dir"}

	gstr := func (a bson.M, b string) (c string) {
		var ok bool
		var v interface{}
		v, ok = a[b]
		if !ok { return }
		c, _ = v.(string)
		return
	}

	type catS struct {
		cat,title,dirs,flags string
	}
	cats := []*catS {
		&catS{"yule", "娱乐", "", "limit"},
		&catS{"news", "新闻", "", "limit"},
		&catS{"zongyi", "综艺", "series,cat1,cat2", ""},
		&catS{"movie", "电影", "regions", ""},
		&catS{"jilu", "纪录片", "series,cat1,cat2", ""},
		&catS{"jiaoyu", "教育", "series,cat1,cat2", ""},
		&catS{"tiyu", "体育", "", "limit"},
		&catS{"qiche", "汽车", "", "limit"},
		&catS{"dianshi", "电视剧", "series,cat1,cat2", "limit"},
	}

	log.Println("menu:", filename, "starts processing")

	tm := time.Now()
	i := 0
	for it.Next(&m) {
		cat := gstr(m, "cat")
		var c *catS
		for _, cc := range cats {
			if cc.cat == cat { c = cc }
		}
		if c == nil {
			continue
		}
		dirs := []string{}
		if c.dirs != "" {
			dirs = strings.Split(c.dirs, ",")
		}
		dirs2 := []string{c.title}
		for _, attr := range dirs {
			astr := gstr(m, attr)
			dirs2 = append(dirs2, astr)
		}
		mu.insert(dirs2, gstr(m, "title"), gstr(m, "_id"))
		if time.Since(tm) > time.Second {
			log.Println("menu:", filename, "processing", i,"/",n)
			tm = time.Now()
		}
		i++
	}

	mu.setid(0)
	log.Println("menu:", filename, "processed", i)
	log.Println("menu:", filename, "dumping files")

	var w io.WriteCloser
	var err error

	w, err = os.Create(filepath.Join("www", filename+".json"))
	if w != nil {
		mu.dump(w)
		w.Close()
	} else {
		log.Println("menu:", err)
	}

	w, _ = os.Create(filepath.Join("www", filename+".txt"))
	if w != nil {
		mu.dumptxt(w)
		w.Close()
	}

	log.Println("menu:", filename, "done")
}

func menu_oneshot() {
	log.Println("menu:", "oneshot")

	menu2_get("all")
	menu2_get("fetched")
}

func menu_loop() {
	log.Println("menu: loop starts")
	for {
		menu_oneshot()
		time.Sleep(time.Hour)
	}
}

