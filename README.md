# get
Simple download library

### Install
```shell
> go get github.com/gonejack/get
```

### Usage
```golang
func main() {
	err := get.Download(time.Second * 3, "https://www.qq.com", "test.html")
	if err != nil {
		log.Println(err)
	}
}
```
