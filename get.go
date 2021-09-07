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
	OnEachSkip  func(t *DownloadTask)
	Header      map[string]string
	Client      http.Client
}

func (g *Get) Download(task *DownloadTask, timeout time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	return g.DownloadWithContext(ctx, task)
}
func (g *Get) DownloadWithContext(ctx context.Context, task *DownloadTask) (err error) {
	if g.shouldSkip(ctx, task) {
		if g.OnEachSkip != nil {
			g.OnEachSkip(task)
		}
		return
	}

	if g.OnEachStart != nil {
		g.OnEachStart(task)
	}

	defer func() {
		task.Err = err
		if g.OnEachStop != nil {
			g.OnEachStop(task)
		}
	}()

	req := resty.NewWithClient(&g.Client).R()
	req.SetContext(ctx)
	req.SetHeaders(g.Header)
	req.SetOutput(task.Path)

	rsp, err := req.Get(task.Link)
	switch {
	case err != nil:
		return
	case !rsp.IsSuccess():
		return fmt.Errorf("response status code %d invalid", rsp.StatusCode())
	case rsp.Size() < rsp.RawResponse.ContentLength:
		return fmt.Errorf("expected %s but downloaded %s", humanize.Bytes(uint64(rsp.RawResponse.ContentLength)), humanize.Bytes(uint64(rsp.Size())))
	default:
		mtime, e := http.ParseTime(rsp.Header().Get("last-modified"))
		if e == nil {
			_ = os.Chtimes(task.Path, mtime, mtime)
		}
		f, e := os.OpenFile(task.Path+".ok", os.O_RDWR|os.O_CREATE, 0666)
		if e == nil {
			_ = f.Close()
		}
		return
	}
}
func (g *Get) Batch(tasks *DownloadTasks, concurrent int, eachTimeout time.Duration) *DownloadTasks {
	var w = semaphore.NewWeighted(int64(concurrent))
	var grp errgroup.Group

	tasks.ForEach(func(t *DownloadTask) {
		_ = w.Acquire(context.TODO(), 1)

		grp.Go(func() (err error) {
			defer w.Release(1)

			t.Err = g.Download(t, eachTimeout)

			return
		})
	})

	_ = grp.Wait()

	return tasks
}
func (g *Get) shouldSkip(ctx context.Context, task *DownloadTask) (skip bool) {
	// check .ok file exist
	fd, err := os.Open(task.Path + ".ok")
	if err == nil {
		_ = fd.Close()
		return true
	}

	// check target file size
	local, err := os.Stat(task.Path)
	if err != nil {
		return false
	}

	switch local.Size() {
	case 0:
		return false
	default:
		req := resty.NewWithClient(&g.Client).R()
		req.SetContext(ctx)
		req.SetHeaders(g.Header)
		rsp, err := req.Head(task.Link)

		// remote and local has equal size
		if err == nil && rsp.RawResponse.ContentLength == local.Size() {
			return true
		}

		return false
	}
}
