
package site

import (
	_"github.com/shopsmart/mgo/bson"
	"github.com/astaxie/beego"

	"fmt"
	"strings"
	"os"
	"path/filepath"
	"log"
)

type menuC struct {
	beego.Controller
}

func (m *menuC) Get() {
	typ := "all"
	filename := filepath.Join("www", "all.json")
	if strings.Contains(m.Ctx.Request.URL.Path, "fetched") {
		typ = "fetched"
		filename = filepath.Join("www", "fetched.json")
	}

	w, err := os.Open(filename)
	if err != nil {
		log.Println("http:", "menu", typ, err)
		m.Abort("404")
		return
	}

	var fi os.FileInfo
	fi, err = w.Stat()
	if err != nil {
		log.Println("http:", "menu", typ, err)
		m.Abort("404")
		return
	}
	m.Data["mtm"] = fi.ModTime()

	tree := &menuM2{}
	err = tree.load(w)
	if err != nil {
		log.Println("http:", filename, "load failed")
		m.Abort("404")
		return
	}

	var node *menuM2
	id := m.Ctx.Params[":id"]
	if id == "" {
		node = tree
	} else {
		var id_ int
		fmt.Sscanf(id, "%d", &id_)
		node = tree.find(id_)
	}
	if node == nil {
		log.Println("http:", "id", id, "not found in", filename)
		m.Abort("404")
		return
	}

	m.Data["type"] = typ
	m.Data["path"] = node.Path()
	m.Data["node"] = node
	m.TplNames = "menu.html"
}

type logFetchC struct {
	beego.Controller
}

func (m *logFetchC) Get() {
}

type logSysC struct {
	beego.Controller
}

func (m *logSysC) Get() {
}

type logParseC struct {
	beego.Controller
}

func (m *logParseC) Get() {
}

func http_loop() {
	beego.SetStaticPath("/www", "www")
	beego.SetStaticPath("/fetch", "fetch")
	beego.Router("/menu/all", &menuC{})
	beego.Router("/menu/fetched", &menuC{})
	beego.Router("/menu/all/:id:int", &menuC{})
	beego.Router("/menu/fetched/:id:int", &menuC{})
	beego.Router("/log/system", &logSysC{})
	beego.Router("/log/fetch", &logFetchC{})
	beego.Router("/log/parse", &logParseC{})
	beego.RunMode = "dev"
	beego.Run()
}

