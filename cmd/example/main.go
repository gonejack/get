package main

import (
	"log"
	"time"

	"github.com/gonejack/get"
)

func main() {
	err := get.Download(time.Second*3, "https://www.qq.com", "test.html")
	if err != nil {
		log.Println(err)
	}
}
