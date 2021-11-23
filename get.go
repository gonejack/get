package get

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type Get struct {
	OnEachStart func(t *DownloadTask)
	OnEachStop  func(t *DownloadTask)
	OnEachSkip  func(t *DownloadTask)
	Header      http.Header
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

	f, err := os.OpenFile(task.Path, os.O_RDWR|os.O_CREATE, 0766)
	if err != nil {
		return
	}
	defer f.Close()

	req, err := http.NewRequest(http.MethodGet, task.Link, nil)
	if err != nil {
		return
	}
	if s, e := f.Stat(); e == nil {
		if s.Size() > 0 {
			req.Header.Set("range", fmt.Sprintf("bytes=%d-", s.Size()))
		}
	}
	for k := range g.Header {
		req.Header[k] = g.Header[k]
	}

	rsp, err := g.Client.Do(req.WithContext(ctx))
	if err != nil {
		return
	}
	defer func() {
		_, _ = io.Copy(io.Discard, rsp.Body)
		_ = rsp.Body.Close()
	}()

	switch rsp.StatusCode {
	case http.StatusPartialContent:
		_, _ = f.Seek(0, io.SeekEnd)
	case http.StatusOK, http.StatusRequestedRangeNotSatisfiable:
		_ = f.Truncate(0)
	default:
		return fmt.Errorf("invalid status code %d(%s)", rsp.StatusCode, rsp.Status)
	}

	_, err = io.Copy(f, rsp.Body)
	if err != nil {
		return fmt.Errorf("copy error: %s", err)
	}

	mt, e := http.ParseTime(rsp.Header.Get("last-modified"))
	if e == nil {
		_ = os.Chtimes(task.Path, mt, mt)
	}
	ok, e := os.Create(task.Path + ".ok")
	if e == nil {
		_ = ok.Close()
	}

	return
}
func (g *Get) Batch(tasks *DownloadTasks, concurrent int, eachTimeout time.Duration) *DownloadTasks {
	var sema = semaphore.NewWeighted(int64(concurrent))
	var grp errgroup.Group

	tasks.ForEach(func(t *DownloadTask) {
		_ = sema.Acquire(context.TODO(), 1)
		grp.Go(func() (err error) {
			defer sema.Release(1)
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
		req, err := http.NewRequest(http.MethodHead, task.Link, nil)
		if err == nil {
			req.Header = g.Header
			rsp, err := g.Client.Do(req.WithContext(ctx))
			if err == nil {
				_ = rsp.Body.Close()
				return rsp.ContentLength == local.Size()
			}
		}
		return false
	}
}
