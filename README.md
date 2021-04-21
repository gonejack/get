# get
Simple download library

### Install
```shell
> go get github.com/gonejack/get
```

### Usage
```golang
func main() {
	err := get.Download("https://www.qq.com", "test.html", time.Second*3)
	if err != nil {
		log.Fatal(err)
	}

	errors := get.Batch(map[string]string{"https://www.qq.com": "test.html"}, 3, time.Second*3)
	for _, e := range errors {
		log.Print(e)
	}

	refs, errs := get.BatchInOrder([]string{"https://www.qq.com"}, []string{"test.html"}, 3, time.Second*3)
	for i := range refs {
		log.Printf("%s: %s", refs[i], errs[i])
	}
}
```
