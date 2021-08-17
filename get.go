package get

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-resty/resty/v2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type Getter struct {
	BeforeDL func(ref string, path string)
	AfterDL  func(ref string, path string, err error)
	Header   map[string]string
	Client   http.Client
}

func (g *Getter) Download(ref string, path string, timeout time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	return g.DownloadWithContext(ctx, ref, path)
}
func (g *Getter) DownloadWithContext(ctx context.Context, ref string, path string) (err error) {
	if g.BeforeDL != nil {
		g.BeforeDL(ref, path)
	}
	if g.AfterDL != nil {
		defer func() {
			g.AfterDL(ref, path, err)
		}()
	}
	if g.shouldSkip(ctx, ref, path) {
		return
	}

	req := resty.NewWithClient(&g.Client).R()
	req.SetContext(ctx)
	req.SetHeaders(g.Header)
	req.SetOutput(path)

	rsp, err := req.Get(ref)
	switch {
	case err != nil:
		return
	case !rsp.IsSuccess():
		return fmt.Errorf("response status code %d invalid", rsp.StatusCode())
	case rsp.Size() < rsp.RawResponse.ContentLength:
		return fmt.Errorf("expected %s but downloaded %s", humanize.Bytes(uint64(rsp.RawResponse.ContentLength)), humanize.Bytes(uint64(rsp.Size())))
	default:
		f, e := os.OpenFile(path+".ok", os.O_RDWR|os.O_CREATE, 0666)
		if e == nil {
			_ = f.Close()
		}
		return
	}
}
func (g *Getter) Batch(downloads map[string]string, concurrent int, eachTimeout time.Duration) (errors map[string]error) {
	var refs, paths []string
	for r, p := range downloads {
		refs = append(refs, r)
		paths = append(paths, p)
	}

	eRefs, errs := g.BatchInOrder(refs, paths, concurrent, eachTimeout)
	if len(errs) > 0 {
		errors = make(map[string]error)
		for i := range eRefs {
			errors[eRefs[i]] = errs[i]
		}
	}

	return
}
func (g *Getter) BatchInOrder(refs []string, paths []string, concurrent int, eachTimeout time.Duration) (errRefs []string, errors []error) {
	var w = semaphore.NewWeighted(int64(concurrent))
	var eg errgroup.Group
	var mu sync.Mutex

	for i := range refs {
		_ = w.Acquire(context.TODO(), 1)

		ref, path := refs[i], paths[i]
		eg.Go(func() (e error) {
			defer w.Release(1)

			e = g.Download(ref, path, eachTimeout)
			if e != nil {
				mu.Lock()
				errRefs = append(errRefs, ref)
				errors = append(errors, e)
				mu.Unlock()
			}

			return
		})
	}

	_ = eg.Wait()

	return
}

func (g *Getter) shouldSkip(ctx context.Context, ref string, path string) (skip bool) {
	fd, err := os.Open(path + ".ok")
	if err == nil {
		_ = fd.Close()
		return true
	}

	stat, err := os.Stat(path)
	if err != nil {
		return
	}
	if stat.Size() == 0 {
		return
	}

	rq := resty.NewWithClient(&g.Client).R()
	rq.SetContext(ctx)
	rq.SetHeaders(g.Header)
	rp, err := rq.Head(ref)
	if err == nil && stat.Size() == rp.RawResponse.ContentLength {
		return true // skip download
	}

	return
}
