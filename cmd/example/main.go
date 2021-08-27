package main

import (
	"log"
	"time"

	"github.com/gonejack/get"
)

func main() {
	err := get.Download(get.NewDownloadTask("https://wx2.sinaimg.cn/large/008h3uCply1gtumw52q7aj31nj27enk8.jpg", "test.jpg"), time.Minute)
	if err != nil {
		log.Fatal(err)
	}
}
