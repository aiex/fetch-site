
package site

import (
	"github.com/go-av/curl"

	"github.com/shopsmart/mgo/bson"
	"github.com/shopsmart/mgo"

	"fmt"
	"strings"
	"regexp"
	"time"
	"sync"
	"log"
)

type parseS struct {
	name string
	l *sync.Mutex
	w *sync.WaitGroup
	list []bson.M
	n int
	threadN int
	starttm time.Time
}

func (m *parseS) init() {
	m.l = &sync.Mutex{}
	m.w = &sync.WaitGroup{}
	m.list = []bson.M{}
	m.starttm = time.Now()
	m.w.Add(1)
}

func (m *parseS) thread(cb func()) {
	m.w.Add(1)
	m.threadN++
	cb()
	m.w.Done()
}

func (m *parseS) add(list []bson.M) {
	m.l.Lock()
	defer m.l.Unlock()
	m.list = append(m.list, list...)
}

func (m *parseS) end() {
	m.w.Done()
	m.w.Wait()

	got := 0
	n := len(m.list)
	t := time.Now().UnixNano()
	for _, l := range m.list {
		l["createtime"] = t
		t--
		err := C("videos").Insert(l)
		if err != nil && mgo.IsDup(err) {
			n--
		}
		if err == nil {
			got++
		}
	}

	log.Println(
		"parse:", m.name, "got", len(m.list),
		"insert", got, "new", n,
		"time", time.Since(m.starttm),
		)
}

func getmatch(str,reg string) (ret string) {
	re, _ := regexp.Compile(reg)
	arr := re.FindAllStringSubmatch(str, -1)
	if len(arr)>0 {
		ret = arr[0][1]
	}
	return
}

func getattr(str,attr string) (val string) {
	return getmatch(str, attr+`="([^"]+)"`)
}

func gethref(str string) (href string) {
	return getattr(str, "href")
}

func gettags(str string) (tags []string) {
	re, _ := regexp.Compile(`<a[^>]+>([^<]+)</a>`)
	arr := re.FindAllStringSubmatch(str, -1)
	for _, a := range arr {
		tags = append(tags, a[1])
	}
	return
}

func getpubdate(str string) (y,m,d int) {
	ymd := getmatch(str, `<span class="pub"><label>上映:</label>([0-9\-]+)</span>`)
	y = 2012
	m = 1
	d = 1
	fmt.Sscanf(ymd, "%d-%d-%d", &y, &m, &d)
	return
}

func getname(str string) (name string) {
	return getmatch(str, `<span class="name">([^<]+)</span>`)
}

func parse_showpage_movie(url string) (ret bson.M) {
	ret = bson.M{}
	_, str := curl.String(url)
	lines := strings.Split(str, "\n")
	for i, l := range lines {
		if strings.Contains(l, "btnplayposi") {
			//fmt.Println(l)
			ret["_id"] = gethref(l)
		}
		if strings.Contains(l, `<label>类型:</label>`) && i+1 < len(lines) {
			ret["tags"] = gettags(lines[i+1])
			//fmt.Println(l, lines[i+1], tags)
		}
		if strings.Contains(l, `<span class="name">`) {
			ret["title"] = getname(l)
		}
		if strings.Contains(l, `<span class="pub"`) {
			y,m,_ := getpubdate(l)
			ret["year"] = y
			ret["month"] = m
		}
		if strings.Contains(l, `<label>地区:</label>`) && i+1 < len(lines) {
			ret["regions"] = gettags(lines[i+1])
		}
	}
	if _, ok := ret["_id"]; !ok {
		return nil
	}
	return
}

func parse_searchpage_movie(search string) (list []string) {
	_, str := curl.String(search)
	//fmt.Println(str)
	url := ""

	for _, l := range strings.Split(str, "\n") {
		if strings.Contains(l, "p_title") {
			url = gethref(l)
		}
		if strings.Contains(l, "p_status") {
			if strings.Contains(l, "正片") && url != "" {
				list = append(list, url)
			}
		}
	}
	return
}

func parse_searchpage_movie_all(search string, typ string) (ret []bson.M) {
	list := parse_searchpage_movie(search)
	for _, u := range list {
		m := parse_showpage_movie(u)
		if m != nil {
			m["showpage"] = u
			m["type"] = typ
			m["cat"] = "movie"
			ret = append(ret, m)
		}
	}
	return
}

func parse_movie() {

	fmts := map[string]string {
		// 用户好评
		//"用户好评": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_11_p_%d.html",
		// 近期更新
		//"近期更新": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_10_p_%d.html",
		// 今日增加播放
		//"今日最多播放": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_7_p_%d.html",
		// 本周增加播放
		"本周最多播放": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_6_p_%d.html",
		// 历史最多播放
		//"历史最多播放": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_1_p_%d.html",
	}

	urlfmt := "http://www.youku.com/v_olist/"

	p := &parseS{name:"movie"}
	p.init()

	for t, f := range fmts {
		for i := 1; i < 2; i++ {
			p.thread(func () {
				search := fmt.Sprintf(urlfmt+f, i)
				list := parse_searchpage_movie_all(search, t)
				p.add(list)
			})
		}
	}
	p.end()
}

func parse_searchpage_news(search,tag string) (list []bson.M) {
	_, str := curl.String(search)
	for _, l := range strings.Split(str, "\n") {
		if strings.Contains(l, "v_title") {
			b := bson.M{}
			b["_id"] = getmatch(l, `href="([^"]+)"`)
			b["title"] = getmatch(l, `title="([^"]+)"`)
			b["cat1"] = tag
			b["cat"] = "news"
			list = append(list, b)
		}
	}
	return
}

func parse_news() {
	fmts := map[string]string {
		// 今日最新
		"social": "t1c91g2143d1p%d.html",
		"tech": 	"t1c91g2147d1p%d.html",
		"life":		"t1c91g2148d1p%d.html",
		"time": 	"t1c91g2144d1p%d.html",
		"army":		"t1c91g258d1p%d.html",
		"money":	"t1c91g308d1p%d.html",
		"law": 		"t1c91g2351d1p%d.html",
	}
	urlfmt := "http://www.youku.com/v_showlist/"

	p := &parseS{name:"news"}
	p.init()

	for tag, f := range fmts {
		for i := 1; i < 2; i++ {
			search := fmt.Sprintf(urlfmt+f, i)
			p.thread(func () {
				list := parse_searchpage_news(search, tag)
				p.add(list)
			})
		}
	}
	p.end()
}

func parse_showpage_zongyi(showurl string) (title string, episode []string) {
	_, str := curl.String(showurl)	
	for _, l := range strings.Split(str, "\n") {
		if strings.Contains(l, `<li data=`) {
			re, _ := regexp.Compile(`<li data="reload_(\d+)"`)
			arr := re.FindAllStringSubmatch(l, -1)
			if len(arr) > 0 {
				episode = append(episode, arr[0][1])
			}
		}
		if strings.Contains(l, `<span class="name">`) {
			title = getname(l)
		}
	}
	return
}

func parse_showpage_zongyi_all(showurl string) (ret []bson.M) {
	title, epi := parse_showpage_zongyi(showurl)
	for ie := len(epi)-1; ie >= 0; ie-- {
		e := epi[ie]
		eurl := strings.Replace(showurl, "show_page", "show_episode", -1)
		reload_e := "reload_" + e
		eurl += "?dt=json&divid="+reload_e+"&__rt=1&"+"__ro="+reload_e
		eret := parse_episode_data(eurl)
		for il := 0; il < len(eret); il++ {
			rl := eret[il]
			rl["cat1"] = e
			rl["cat2"] = rl["no"]
			delete(rl, "no")
			rl["series"] = title
			rl["cat"] = "zongyi"
			ret = append(ret, rl)
		}
	}
	epi1 := parse_episode_data(showurl)
	for _, rl := range epi1 {
		rl["cat1"] = rl["no"]
		delete(rl, "no")
		rl["series"] = title
		rl["cat"] = "zongyi"
		ret = append(ret, rl)
	}
	return
}

func parse_episode_data(eurl string) (ret []bson.M) {
	_, str := curl.String(eurl)
	larr := strings.Split(str, "\n")
	for i, l := range larr {
		if !strings.Contains(l, `<li class="ititle`) {
			continue
		}
		if i+1 < len(larr) {
			no := getmatch(l, `<label>([\d\-]+)`)
			title := getattr(larr[i+1], "title")
			href := getattr(larr[i+1], "href")
			if href == "" {
				continue
			}
			ret = append(ret, bson.M{
				"no": no, "title": title, "_id": href,
			})
		}
	}
	return
}

func parse_searchpage_zongyi(search string) (list []string) {
	_, str := curl.String(search)	
	for _, l := range strings.Split(str, "\n") {
		if strings.Contains(l, "p_title") {
			url := gethref(l)
			list = append(list, url)
		}
	}
	return
}

func parse_zongyi() {
	urlfmt := "http://www.youku.com/v_olist/"
	// 今日增加播放
	urlfmt += "c_85_a__s__g__r__lg__im__st__mt__d_1_et_0_fv_0_fl__fc__fe_1_o_7_p_%d.html"

	p := &parseS{name:"zongyi"}
	p.init()

	for i := 1; i < 2; i++ {
		list := parse_searchpage_zongyi(fmt.Sprintf(urlfmt, i))
		for _, u := range list {
			p.thread(func () {
				list := parse_showpage_zongyi_all(u)
				p.add(list)
			})
		}
	}
	p.end()
}

func dump_showpage_zongyi(url string) {
	ret := parse_showpage_zongyi_all(url)
	for _, r := range ret {
		if _, ok := r["cat2"]; ok {
			fmt.Println(r["cat1"], r["cat2"], r["title"])
		} else {
			fmt.Println(r["cat1"], r["title"])
		}
	}
}

func parse_loop() {
	for {
		log.Println("fetch:", "starts")
		go parse_zongyi()
		go parse_news()
		go parse_movie()
		time.Sleep(time.Second*120)
	}
}
