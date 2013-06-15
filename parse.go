
package site

import (
	"github.com/go-av/curl"

	"github.com/shopsmart/mgo/bson"

	"fmt"
	"strings"
	"regexp"
	"time"
	"sync"
	"log"
)

type parseN struct {
	my,n int
}

type parseS struct {
	cat string
	w *sync.WaitGroup
	l *sync.Mutex
	list []bson.M
	starttm time.Time
	attrs bson.M
	tm int64

	nrun,ndone,ncurl,ngot,ninsert parseN
	per float64

	stable,done bool
	child []*parseS
}

func (m *parseS) init() {
	m.l = &sync.Mutex{}
	m.list = []bson.M{}
	m.starttm = time.Now()
	m.attrs = bson.M{}
	m.child = []*parseS{}
	m.w = &sync.WaitGroup{}
	m.tm = time.Now().UnixNano()
}

func (m *parseS) task(cb func(*parseS)) {
	m.l.Lock()
	defer m.l.Unlock()

	m2 := &parseS{cat:m.cat}
	m2.init()

	for k,v := range m.attrs {
		m2.attrs[k] = v
	}

	m.child = append(m.child, m2)
	m.w.Add(1)
	m.nrun.my++
	go func () {
		cb(m2)
		m2.wait()
		m.w.Done()
		m.nrun.my--
	}()
}

func (m *parseS) wait() {
	m.stable = true
	m.w.Wait()
	m.done = true
	m.ndone.my++
}

func (m *parseS) sumup() {
	m.l.Lock()
	defer m.l.Unlock()

	if !m.stable {
		return
	}

	m.nrun.n = m.nrun.my
	m.ncurl.n = m.ncurl.my
	m.ndone.n = m.ndone.my
	m.ngot.n = m.ngot.my
	m.ninsert.n = m.ninsert.my
	per := float64(0)

	for _, c := range m.child {
		c.sumup()
		m.nrun.n += c.nrun.n
		m.ncurl.n += c.ncurl.n
		m.ndone.n += c.ndone.n
		m.ngot.n += c.ngot.n
		m.ninsert.n += c.ninsert.n
		per += c.per/float64(len(m.child))
	}

	if m.done { m.per = 1 } else { m.per = per }

	return
}

func (m *parseS) curl(url string) (ret string) {
	_, ret = curl.String(url, "timeout=10")
	m.ncurl.my++
	return
}

func (m *parseS) setattr(a bson.M) (b bson.M) {
	for k,v := range m.attrs {
		a[k] = v
	}
	return a
}

func (m *parseS) add(a bson.M) {
	m.l.Lock()
	//log.Println("parse:", m.cat, "added 1")
	a = m.setattr(a)
	m.list = append(m.list, a)
	m.ngot.my++
	m.l.Unlock()

	m.insert(a)
}

func (m *parseS) insert(a bson.M) {
	dt := time.Now()
	a["cat"] = m.cat
	a["createtime"] = m.tm
	a["createdate"] = dt
	m.tm--
	err := C("videos").Insert(a)
	if err == nil {
		m.ninsert.my++
	}
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

func (m *parseS) episode2(str, cati string) {
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
				m.add(bson.M{cati:tags[0], "_id":href, "title":title})
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

			m.add(bson.M{cati:no, "_id":href, "title":title})
		}
	}
	return
}

func (m *parseS) episode(showurl string, epi []string) {
	for ie := len(epi)-1; ie >= 0; ie-- {
		e := epi[ie]
		eurl := strings.Replace(showurl, "show_page", "show_episode", -1)
		reload_e := "reload_" + e
		eurl += "?dt=json&divid="+reload_e+"&__rt=1&"+"__ro="+reload_e
		estr := m.curl(eurl)
		m.attrs["cat1"] = e
		m.task(func (m2 *parseS) {
			m2.episode2(estr, "cat2")
		})
	}
	return
}

func (m *parseS) showpage(url string) {
	typs := []string{}
	regions := []string{}
	episode := []string{}
	oneurl := ""
	title := ""
	year := ""
	month := ""

	str := m.curl(url)
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

	if len(typs) > 0 { m.attrs["type"] = typs }
	if len(regions) > 0 { m.attrs["regions"] = regions }
	if year != "" { m.attrs["year"] = year }
	if month != "" { m.attrs["month"] = month }
	m.attrs["series"] = title
	m.attrs["showpage"] = url

	if oneurl != "" {
		m.add(bson.M{"_id": oneurl, "title": title})
	}
	m.episode2(str, "cat1")
	if len(episode) > 0 {
		m.episode(url, episode)
	}

	return
}

func (m *parseS) showpages(urls []string) {
	for _, u := range urls {
		url := u
		m.task(func (m2 *parseS) {
			m2.showpage(url)
		})
	}
}

func (m *parseS) searchpage_big(search string) (list []string) {
	str := m.curl(search)
	for _, l := range strings.Split(str, "\n") {
		if strings.Contains(l, "p_title") {
			url := gethref(l)
			list = append(list, url)
		}
	}
	return
}

func (m *parseS) searchpage_small(search string) {
	str := m.curl(search)
	for _, l := range strings.Split(str, "\n") {
		if strings.Contains(l, "v_title") {
			b := bson.M{}
			b["_id"] = getmatch(l, `href="([^"]+)"`)
			b["title"] = getmatch(l, `title="([^"]+)"`)
			m.add(b)
		}
	}
	return
}

func (m *parseS) searchpage_movie(search string) (list []string) {
	str := m.curl(search)
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

func (m *parseS) searchpage_all(
	prefix string,
	fmts map[string]string,
	cb func (*parseS, string),
) {
	for t, f := range fmts {
		for i := 1; i < 2; i++ {
			search := fmt.Sprintf(prefix+f, i)
			play := t
			m.task(func (m2 *parseS) {
				m2.attrs["play"] = play
				cb(m2, search)
			})
		}
	}

	return
}

func (m *parseS) news_template(fmts map[string]string) {
	prefix := "http://www.youku.com/v_showlist/"
	m.searchpage_all(prefix, fmts,
	func (m2 *parseS, url string) {
		m2.searchpage_small(url)
	})
}

func (m *parseS) zongyi_template(fmts map[string]string) {
	prefix := "http://www.youku.com/v_olist/"
	m.searchpage_all(prefix, fmts,
	func (m2 *parseS, url string) {
		urls := m2.searchpage_big(url)
		m2.showpages(urls)
	})
}

func (m *parseS) movie() {
	fmts := map[string]string {
		//"用户好评": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_11_p_%d.html",
		//"近期更新": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_10_p_%d.html",
		//"今日最多播放": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_7_p_%d.html",
		"本周最多播放": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_6_p_%d.html",
		//"历史最多播放": "c_96_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe_1_o_1_p_%d.html",
	}
	prefix := "http://www.youku.com/v_olist/"

	m.searchpage_all(prefix, fmts,
	func (m2 *parseS, url string) {
		urls := m2.searchpage_movie(url)
		m2.showpages(urls)
	})
}

func (m *parseS) news() {
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
	m.news_template(fmts)
}

func (m *parseS) yule() {
	// 娱乐今日最多播放
	fmts := map[string]string {
		"最多播放": "t2c86g0d1p%d.html",
	}
	m.news_template(fmts)
}

func (m *parseS) zongyi() {
	// 综艺今日增加播放
	fmts := map[string]string {
		"最多播放": "c_85_a__s__g__r__lg__im__st__mt__d_1_et_0_fv_0_fl__fc__fe_1_o_7_p_%d.html",
	}
	m.zongyi_template(fmts)
}

func (m *parseS) jilu() {
	// 纪录片今日增加播放
	fmts := map[string]string {
		"最多播放": "c_84_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe__o_7_p_%d.html",
	}
	m.zongyi_template(fmts)
}

func (m *parseS) jiaoyu() {
	// 教育今日增加播放
	fmts := map[string]string {
		"最多播放": "c_87_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe__o_7_p_%d.html",
	}
	m.zongyi_template(fmts)
}

func (m *parseS) tiyu() {
	// 体育今日最多播放
	fmts := map[string]string {
		"最多播放": "t2c98g0d1p%d.html",
	}
	m.news_template(fmts)
}

func (m *parseS) qiche() {
	// 汽车今日
	fmts := map[string]string {
		"最多播放": "t2c98g0d1p%d.html",
	}
	m.news_template(fmts)
}

func (m *parseS) dianshi() {
	fmts := map[string]string {
		"最多播放": "c_97_a__s__g__r__lg__im__st__mt__tg__d_1_et_0_fv_0_fl__fc__fe__o_7_p_%d.html",
	}
	m.zongyi_template(fmts)
}

func parse_loop() {
	log.Println("parse: loop starts")
}

func parse_oneshot() {
	log.Println("parse: starts")
	m := &parseS{}
	m.init()

	done := make(chan int, 0)

	go func() {

		tm := time.Now()
		m.task(func (m2 *parseS) {
			m2.cat = "zongyi"
			m2.zongyi()
		})
		m.task(func (m2 *parseS) {
			m2.cat = "dianshi"
			m2.dianshi()
		})
		m.task(func (m2 *parseS) {
			m2.cat = "yule"
			m2.yule()
		})
		m.task(func (m2 *parseS) {
			m2.cat = "news"
			m2.news()
		})
		m.task(func (m2 *parseS) {
			m2.cat = "movie"
			m2.movie()
		})
		m.task(func (m2 *parseS) {
			m2.cat = "qiche"
			m2.qiche()
		})
		m.task(func (m2 *parseS) {
			m2.cat = "tiyu"
			m2.tiyu()
		})
		m.task(func (m2 *parseS) {
			m2.cat = "jilu"
			m2.jilu()
		})

		m.task(func (m2 *parseS) {
			m2.cat = "jiaoyu"
			m2.jiaoyu()
		})
		m.wait()
		log.Println("parse: done in", time.Since(tm))
		done <- 1
	}()

	for {
		m.sumup()
		select {
		case <-done:
			log.Println("parse:", "got", m.ngot.n, "insert", m.ninsert.n)
			return
		case <-time.After(time.Second):
			log.Println("parse:", curl.PrettyPer(m.per), "got", m.ngot.n, "insert", m.ninsert.n, "thread", m.nrun.n)
		}
	}
}

