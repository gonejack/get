package get

import (
	"context"
	"time"
)

var _default = Getter{
	Header: map[string]string{
		"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.72 Safari/537.36",
	},
}

func DefaultGetter() Getter {
	return _default
}

func Download(ref string, path string, timeout time.Duration) (err error) {
	return _default.Download(ref, path, timeout)
}
func DownloadWithContext(ctx context.Context, ref string, path string) (err error) {
	return _default.DownloadWithContext(ctx, ref, path)
}

func Batch(downloads map[string]string, concurrent int, eachTimeout time.Duration) (errors map[string]error) {
	return _default.Batch(downloads, concurrent, eachTimeout)
}
func BatchInOrder(refs []string, paths []string, concurrent int, eachTimeout time.Duration) (errRefs []string, errors []error) {
	return _default.BatchInOrder(refs, paths, concurrent, eachTimeout)
}
