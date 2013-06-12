
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
	c := exec.Command("mongod_win64.exe", "--dbpath=db")
	go c.Run()
}

func C(c string) *mgo.Collection {
	return gSess.DB("site").C(c)
}

func Test() {
	log.Println("starts")
	startdb()
	initdb()
	go parse_loop()
	go download_loop()
	for {
		time.Sleep(time.Hour)
	}
}
