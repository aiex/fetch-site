
package site

import (
	"github.com/shopsmart/mgo/bson"

	"testing"
	"fmt"
	"os"
)


func TestParseShowpage(t *testing.T) {
	dump := func (name string, list []bson.M) {
		fname := "dump."+name
		f, _ := os.Create(fname)
		defer f.Close()
		fmt.Println("testing", name)
		fmt.Println("total", len(list), "dumpto", fname)
		for _, l := range list {
			fmt.Fprintln(f, "  ", l)
		}
	}

	ret := []bson.M{}

	ret = parse_showpage("http://www.youku.com/show_page/id_zf3b63266595211e29498.html")
	dump("1", ret)

	ret = parse_showpage("http://www.youku.com/show_page/id_z53c8401cc84411e2b356.html")
	dump("2", ret)

	//ret = parse_showpage("http://www.youku.com/show_page/id_zd69d747acaf711e2b356.html")
	//dump(ret)

	//ret = parse_showpage("http://www.youku.com/show_page/id_z47c8257c48bd11e29013.html")
	//dump(ret)
}

