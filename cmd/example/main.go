package main

import (
	"log"
	"time"

	"github.com/gonejack/get"
)

func main() {
	err := get.Download(get.NewDownloadTask("https://www.qq.com", "test.html"), time.Second*3)
	if err != nil {
		log.Fatal(err)
	}

	downloads := get.NewDownloadTasks()
	{
		downloads.Add("https://www.qq.com", "test.html")
	}
	downloads = get.Batch(downloads, 3, time.Second*3)
	for _, d := range downloads.List {
		if d.Err != nil {
			log.Println(d.Err)
		}
	}
}
