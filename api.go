package get

import (
	"context"
	"time"
)

func Default() (g Get) {
	g.Header = map[string]string{
		"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.72 Safari/537.36",
	}
	return g
}

var one = Default()

func Download(dl *DownloadTask, timeout time.Duration) (err error) {
	return one.Download(dl, timeout)
}
func DownloadWithContext(ctx context.Context, dl *DownloadTask) (err error) {
	return one.DownloadWithContext(ctx, dl)
}
func Batch(jobs *Downloads, concurrent int, eachTimeout time.Duration) *Downloads {
	return one.Batch(jobs, concurrent, eachTimeout)
}
