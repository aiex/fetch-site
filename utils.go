
package site

import (

	"github.com/shopsmart/mgo"

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
			err := c.Run()
			log.Println(err)
		}()
	}
}

func C(c string) *mgo.Collection {
	return gSess.DB("site").C(c)
}

func Test() {
	log.Println("starts")
	startdb()
	initdb()
	menu_zongyi_output()
	if false {
		go parse_loop()
		go download_loop()
	}
	for {
		time.Sleep(time.Hour)
	}
}

