
package site

import (
	_"github.com/shopsmart/mgo/bson"
	"github.com/astaxie/beego"

	"strings"
	"log"
)

type menuC struct {
	beego.Controller
}

func (m *menuC) Get() {
	filename := "www/all.json"
	if strings.Contains(m.Ctx.Request.URL.Path, "fetched") {
		filename = "www/fetched.json"
	}

	tree := &menuM{}
	err := tree.load(filename)
	if err != nil {
		log.Println("menu:", filename, "load failed")
		m.Abort("404")
		return
	}

	var node *menuM
	id := m.Ctx.Params[":id"]
	if id == "" {
		node = tree
	} else {
		tree.findid(id)
		node = tree.ptr
	}
	if node == nil {
		log.Println("menu:", "id", id, "not found in", filename)
		m.Abort("404")
		return
	}

	m.Data["path"] = tree.path
	m.Data["node"] = node
	m.TplNames = "menu.html"
}

func http_loop() {
	beego.SetStaticPath("/www", "www")
	beego.SetStaticPath("/fetch", "fetch")
	beego.Router("/menu/all", &menuC{})
	beego.Router("/menu/fetched", &menuC{})
	beego.Router("/menu/all/:id:int", &menuC{})
	beego.Router("/menu/fetched/:id:int", &menuC{})
	beego.RunMode = "dev"
	beego.Run()
}

