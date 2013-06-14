
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
	threadN,running int
	starttm time.Time
	isend bool
}

func (m *parseS) init() {
	m.l = &sync.Mutex{}
	m.w = &sync.WaitGroup{}
	m.list = []bson.M{}
	m.starttm = time.Now()
	m.w.Add(1)

	go func() {
		for {
			if m.isend {
				return
			}
			log.Println("parse:", m.name, "thread created", m.threadN, "running", m.running)
			time.Sleep(time.Second)
		}
	}()
}

func (m *parseS) thread(cb func()) {
	m.w.Add(1)
	m.threadN++
	m.running++
	cb()
	m.running--
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
	dt := time.Now()
	for _, l := range m.list {
		l["cat"] = m.name
		l["createtime"] = t
		l["createdate"] = dt
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
	m.isend = true
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
	fmt.Sscanf(ymd, "%d-%d-%d", &y, &m, &d)
	return
}

func getname(str string) (name string) {
	return getmatch(str, `<span class="name">([^<]+)</span>`)
}

func parse_episode2(str, cati string) (ret []bson.M) {
	larr := strings.Split(str, "\n")
	coll10 := false
	for i, l := range larr {
		if i == 0 && strings.Contains(l, `class="coll_10"`) {
			coll10 = true
		}
		if coll10 {
			href := gethref(l)
			tags := gettags(l)
			title := getattr(l, "title")
			if href != "" && len(tags) > 0 {
				ret = append(ret, bson.M{cati: tags[0], "_id": href, "title": title})
			}
			continue
		}
		if !strings.Contains(l, `<li class="ititle_w`) {
			continue
		}
		if i+1 < len(larr) {
			no := getmatch(l, `<label>([\d\-]+)`)
			title := getattr(larr[i+1], "title")
			href := getattr(larr[i+1], "href")
			if href == "" {
				continue
			}
			ret = append(ret, bson.M{cati: no, "_id": href, "title": title})
		}
	}
	return
}

func parse_episode(showurl string, epi []string) (ret []bson.M) {
	for ie := len(epi)-1; ie >= 0; ie-- {
		e := epi[ie]
		eurl := strings.Replace(showurl, "show_page", "show_episode", -1)
		reload_e := "reload_" + e
		eurl += "?dt=json&divid="+reload_e+"&__rt=1&"+"__ro="+reload_e
		_, estr := curl.String(eurl)
		eret := parse_episode2(estr, "cat2")
		for il := 0; il < len(eret); il++ {
			rl := eret[il]
			rl["cat1"] = e
			ret = append(ret, rl)
		}
	}
	return
}

func parse_showpage(url string) (ret []bson.M) {
	ret = []bson.M{}
	typs := []string{}
	regions := []string{}
	episode := []string{}
	oneurl := ""
	title := ""
	year := ""
	month := ""

	_, str := curl.String(url)
	lines := strings.Split(str, "\n")
	for i, l := range lines {
		// 播放正片
		if strings.Contains(l, "btnplayposi") {
			oneurl = gethref(l)
		}
		if strings.Contains(l, `<label>类型:</label>`) && i+1 < len(lines) {
			typs = gettags(lines[i+1])
		}
		if strings.Contains(l, `<span class="name">`) {
			title = getname(l)
		}
		if strings.Contains(l, `<span class="pub"`) {
			y,m,_ := getpubdate(l)
			if y != 0 { year = fmt.Sprintf("%d", y) }
			if m != 0 { month = fmt.Sprintf("%d", m) }
		}
		if strings.Contains(l, `<label>地区:</label>`) && i+1 < len(lines) {
			regions = gettags(lines[i+1])
		}
		if strings.Contains(l, `<li data=`) {
			re, _ := regexp.Compile(`<li data="reload_(\d+)"`)
			arr := re.FindAllStringSubmatch(l, -1)
			if len(arr) > 0 {
				episode = append(episode, arr[0][1])
			}
		}
	}

	if oneurl != "" {
		ret = append(ret, bson.M{"_id": oneurl, "title": title})
	}
	ret = append(ret, parse_episode2(str, "cat1")...)
	if len(episode) > 0 {
		ret = append(ret, parse_episode(url, episode)...)
	}

	cat := bson.M{}
	if len(typs) > 0 { cat["type"] = typs }
	if len(regions) > 0 { cat["regions"] = regions }
	if year != "" { cat["year"] = year }
	if month != "" { cat["month"] = month }
	cat["series"] = title
	cat["showpage"] = url

	for _, l := range ret {
		for k, v := range cat { l[k] = v }
	}

	return
}

func parse_searchpage_big(search string) (list []string) {
	_, str := curl.String(search)
	for _, l := range strings.Split(str, "\n") {
		if strings.Contains(l, "p_title") {
			url := gethref(l)
			list = append(list, url)
		}
	}
	return
}

func parse_searchpage_small(search string) (list []bson.M) {
	_, str := curl.String(search)
	for _, l := range strings.Split(str, "\n") {
		if strings.Contains(l, "v_title") {
			b := bson.M{}
			b["_id"] = getmatch(l, `href="([^"]+)"`)
			b["title"] = getmatch(l, `title="([^"]+)"`)
			list = append(list, b)
		}
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

func parse_searchpage_all(
	prefix string,
	fmts map[string]string,
	cat string,
	cb func (string)([]bson.M),
) (p *parseS) {

	p = &parseS{name:cat}
	p.init()

	for t, f := range fmts {
		for i := 1; i < 2; i++ {
			p.thread(func () {
				search := fmt.Sprintf(prefix+f, i)
				ret := cb(search)
				for _, l := range ret { l["play"] = t }
				p.add(ret)
			})
		}
	}

	return
}

func parse_movie() (p *parseS) {
	fmts := map[string]string {
		//"用户好评": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_11_p_%d.html",
		//"近期更新": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_10_p_%d.html",
		//"今日最多播放": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_7_p_%d.html",
		"本周最多播放": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_6_p_%d.html",
		//"历史最多播放": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_1_p_%d.html",
	}
	prefix := "http://www.youku.com/v_olist/"

	return parse_searchpage_all(prefix, fmts, "movie",
	func (url string) (ret []bson.M) {
		urls := parse_searchpage_movie(url)
		for _, u := range urls {
			ret = append(ret, parse_showpage(u)...)
		}
		return
	})
}

func parse_news_template(cat string, fmts map[string]string) (p *parseS) {
	prefix := "http://www.youku.com/v_showlist/"
	return parse_searchpage_all(prefix, fmts, cat, parse_searchpage_small)
}

func parse_zongyi_template(cat string, fmts map[string]string) (p *parseS) {
	prefix := "http://www.youku.com/v_olist/"
	return parse_searchpage_all(prefix, fmts, cat,
	func (url string) (ret []bson.M) {
		urls := parse_searchpage_big(url)
		for _, u := range urls {
			ret = append(ret, parse_showpage(u)...)
		}
		return
	})
}

func parse_news() (p *parseS) {
	// 资讯今日最新
	fmts := map[string]string {
		"社会新闻": "t1c91g2143d1p%d.html",
		"科技新闻":	"t1c91g2147d1p%d.html",
		"生活新闻": "t1c91g2148d1p%d.html",
		"时政新闻": "t1c91g2144d1p%d.html",
		"军事新闻": "t1c91g258d1p%d.html",
		"财经新闻":	"t1c91g308d1p%d.html",
		"法律新闻": "t1c91g2351d1p%d.html",
	}
	return parse_news_template("news", fmts)
}

func parse_yule() (p *parseS) {
	// 娱乐今日最多播放
	fmts := map[string]string {
		"最多播放": "t2c86g0d1p%d.html",
	}
	return parse_news_template("yule", fmts)
}

func parse_zongyi() (p *parseS) {
	// 综艺今日增加播放
	fmts := map[string]string {
		"最多播放": "c_85_a__s__g__r__lg__im__st__mt__d_1_et_0_fv_0_fl__fc__fe_1_o_7_p_%d.html",
	}
	return parse_zongyi_template("zongyi", fmts)
}

func parse_jilu() (p *parseS) {
	// 纪录片今日增加播放
	fmts := map[string]string {
		"最多播放": "c_84_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe__o_7_p_%d.html",
	}
	return parse_zongyi_template("jilu", fmts)
}

func parse_jiaoyu() (p *parseS) {
	// 教育今日增加播放
	fmts := map[string]string {
		"最多播放": "c_87_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe__o_7_p_%d.html",
	}
	return parse_zongyi_template("jiaoyu", fmts)
}

func parse_tiyu() (p *parseS) {
	// 体育今日最多播放
	fmts := map[string]string {
		"最多播放": "t2c98g0d1p%d.html",
	}
	return parse_news_template("tiyu", fmts)
}

func parse_qiche() (p *parseS) {
	// 汽车今日
	fmts := map[string]string {
		"最多播放": "t2c98g0d1p%d.html",
	}
	return parse_news_template("qiche", fmts)
}

func parse_dianshi() (p *parseS) {
	fmts := map[string]string {
		"最多播放": "c_97_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe__o_7_p_%d.html",
	}
	return parse_zongyi_template("dianshi", fmts)
}

func _parse_loop(oneshot bool) {

	type parseF func () (*parseS)

	loop := func (cb parseF) {
		for {
			p := cb()
			p.end()
			if oneshot {
				break
			}
			time.Sleep(time.Second*120)
		}
	}

	go loop(parse_news)
	go loop(parse_zongyi)
	go loop(parse_movie)
	go loop(parse_jilu)
	go loop(parse_jiaoyu)
	go loop(parse_yule)
	go loop(parse_tiyu)
	go loop(parse_qiche)
	go loop(parse_dianshi)

	for {
		time.Sleep(time.Second*120)
	}
}

func parse_loop() {
	log.Println("parse: loop starts")
	_parse_loop(false)
}

func parse_oneshot() {
	log.Println("parse: one shot")
	_parse_loop(true)
}

