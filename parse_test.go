
package site

import (
	"testing"
	"os"
	_"log"
	"fmt"
)

func TestParseShowpage(t *testing.T) {
	f, _ := os.Create("dump.1")
	defer f.Close()
	fmt.Println("test")
	m := &parseS{debug:true}
	m.init()

	// style1
	//m.showpage("http://www.youku.com/show_page/id_z779fb5c8a25211e296da.html")
	// style2
	//m.showpage("http://www.youku.com/show_page/id_zc3dd544e3d0911e2b356.html")
	// style3
	//m.showpage("http://www.youku.com/show_page/id_z53c8401cc84411e2b356.html")
	// style4
	//m.showpage("http://www.youku.com/show_page/id_z6e78f9a0dd4511e196ac.html")

	m.showpage("http://www.youku.com/show_page/id_z1b477bfe2fba11e2b16f.html")

	//m.yule()

	m.wait()
}

