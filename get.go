package get

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-resty/resty/v2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type Get struct {
	OnEachStart func(t *DownloadTask)
	OnEachStop  func(t *DownloadTask)
	Header      map[string]string
	Client      http.Client
}

func (g *Get) Download(task *DownloadTask, timeout time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	return g.DownloadWithContext(ctx, task)
}
func (g *Get) DownloadWithContext(ctx context.Context, d *DownloadTask) (err error) {
	if g.OnEachStart != nil {
		g.OnEachStart(d)
	}
	if g.OnEachStop != nil {
		defer func() {
			d.Err = err
			g.OnEachStop(d)
		}()
	}
	if g.shouldSkip(ctx, d) {
		return
	}

	req := resty.NewWithClient(&g.Client).R()
	{
		req.SetContext(ctx)
		req.SetHeaders(g.Header)
		req.SetOutput(d.Path)
	}

	rsp, err := req.Get(d.Link)
	switch {
	case err != nil:
		return
	case !rsp.IsSuccess():
		return fmt.Errorf("response status code %d invalid", rsp.StatusCode())
	case rsp.Size() < rsp.RawResponse.ContentLength:
		return fmt.Errorf("expected %s but downloaded %s", humanize.Bytes(uint64(rsp.RawResponse.ContentLength)), humanize.Bytes(uint64(rsp.Size())))
	default:
		f, e := os.OpenFile(d.Path+".ok", os.O_RDWR|os.O_CREATE, 0666)
		if e == nil {
			_ = f.Close()
		}
		return
	}
}
func (g *Get) Batch(tasks *DownloadTasks, concurrent int, eachTimeout time.Duration) *DownloadTasks {
	var w = semaphore.NewWeighted(int64(concurrent))
	var eg errgroup.Group

	for i := range tasks.List {
		_ = w.Acquire(context.TODO(), 1)

		dl := tasks.List[i]
		eg.Go(func() (err error) {
			defer w.Release(1)
			dl.Err = g.Download(dl, eachTimeout)
			return
		})
	}

	_ = eg.Wait()

	return tasks
}
func (g *Get) shouldSkip(ctx context.Context, d *DownloadTask) (skip bool) {
	fd, err := os.Open(d.Path + ".ok")
	if err == nil {
		_ = fd.Close()
		return true
	}

	stat, err := os.Stat(d.Path)
	if err != nil {
		return
	}
	if stat.Size() == 0 {
		return
	}

	rq := resty.NewWithClient(&g.Client).R()
	{
		rq.SetContext(ctx)
		rq.SetHeaders(g.Header)
	}

	rp, err := rq.Head(d.Link)
	if err == nil && stat.Size() == rp.RawResponse.ContentLength {
		return true // skip download
	}

	return
}
