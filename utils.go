
package site

import (
	"github.com/shopsmart/mgo"

	"strings"
	"os"
	"os/exec"
	"log"
	"time"
)

var (
	gSess *mgo.Session
)

func initdb() {
	log.Println("db: connecting")
	url := "localhost"
	sess, err := mgo.DialWithTimeout(url, time.Second*20)
	if err != nil {
		log.Println("db: connect failed")
		os.Exit(1)
		return
	}
	log.Println("db: connected")
	gSess = sess
}

func startdb() {
	log.Println("db: start mongod")
	exes := []string{
		"./mongod_win64.exe",
		"./mongod_linux64",
	}
	for _, s := range exes {
		go func() {
			c := exec.Command(s, "--dbpath=db")
			c.Run()
			//log.Println(err)
		}()
	}
}

func C(c string) *mgo.Collection {
	return gSess.DB("site").C(c)
}

func Test() {
	log.Println("starts")

	if len(os.Args) < 2 {
		log.Println("no args")
		return
	}

	loops := []string{}
	dos := []string{}
	args := os.Args[1:]
	for i, o := range args {
		if o == "-loop" && i+1 < len(args) {
			loops = strings.Split(args[i+1], ",")
		}
		if o == "-do" && i+1 < len(args) {
			dos = strings.Split(args[i+1], ",")
		}
	}

	if len(loops) == 0 && len(dos) == 0 {
		log.Println("must specify -loop or -do")
		return
	}

	startdb()
	initdb()

	n := 0
	for _, s := range loops {
		switch s {
		case "menu":
			go menu_loop()
			n++
		case "fetch":
			go fetch_loop()
			n++
		case "parse":
			go parse_loop()
			n++
		case "http":
			go http_loop()
			n++
		}
	}
	if len(loops) > 0 && n == 0 {
		log.Println("please specify operations: menu,fetch,parse,http")
		return
	}

	for _, s := range dos {
		switch s {
		case "parse":
			parse_oneshot()
		case "menu":
			menu_oneshot()
		case "http":
			http_loop()
		}
	}

	if len(loops) == 0 {
		return
	}

	for {
		time.Sleep(time.Hour)
	}
}

