package get

import (
	"context"
	"time"
)

var defaultGet = &Getter{
	Header: map[string]string{
		"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.72 Safari/537.36",
	},
}

func Download(timeout time.Duration, ref string, path string) (err error) {
	return defaultGet.Download(timeout, ref, path)
}
func DownloadWithContext(ctx context.Context, ref string, path string) (err error) {
	return defaultGet.DownloadWithContext(ctx, ref, path)
}

func Batch(downloads map[string]string, concurrent int, eachTimeout time.Duration) (errors map[string]error) {
	return defaultGet.Batch(downloads, concurrent, eachTimeout)
}
func BatchInOrder(refs []string, paths []string, concurrent int, eachTimeout time.Duration) (errRefs []string, errors []error) {
	return defaultGet.BatchInOrder(refs, paths, concurrent, eachTimeout)
}
