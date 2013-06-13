
package site

import (

	"github.com/shopsmart/mgo"

	"os"
	"os/exec"
	"log"
	"time"
	"net/http"
	"strings"
	"path/filepath"
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

func http_handle(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path[1:])
	switch {
	case strings.HasPrefix(r.URL.Path, "/fetch"),
			 strings.HasPrefix(r.URL.Path, "/www"):
		log.Println("http:", path)
		http.ServeFile(w, r, path)
	default:
		http.Error(w, "not found", 404)
	}
}

func http_loop() {
	http.HandleFunc("/", http_handle)
	http.ListenAndServe(":1989", nil)
}

func Test() {
	log.Println("starts")
	startdb()
	initdb()
	if len(os.Args) > 1 {
		for _, s := range os.Args[1:] {
			switch s {
			case "parse":
				go parse_loop()
			case "fetch":
				go fetch_loop()
			case "menu":
				go menu_loop()
			case "http":
				go http_loop()
			}
		}
	} else {
		go parse_loop()
		go fetch_loop()
		go menu_loop()
		go http_loop()
	}
	for {
		time.Sleep(time.Hour)
	}
}

