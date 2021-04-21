package get

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type Getter struct {
	Header  map[string]string
	Client  http.Client
	Verbose bool
}

func (g *Getter) Download(timeout time.Duration, ref string, path string) (err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	return g.DownloadWithContext(ctx, ref, path)
}
func (g *Getter) DownloadWithContext(ctx context.Context, ref string, path string) (err error) {
	if g.shouldSkipDownload(ctx, ref, path) {
		return
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return
	}
	defer file.Close()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ref, nil)
	if err != nil {
		return
	}
	response, err := g.Client.Do(g.withHeader(request))
	if err != nil {
		return
	}
	defer response.Body.Close()

	var written int64
	if g.Verbose {
		bar := progressbar.NewOptions64(response.ContentLength,
			progressbar.OptionSetTheme(progressbar.Theme{Saucer: "=", SaucerPadding: ".", BarStart: "|", BarEnd: "|"}),
			progressbar.OptionSetWidth(10),
			progressbar.OptionSpinnerType(11),
			progressbar.OptionShowBytes(true),
			progressbar.OptionShowCount(),
			progressbar.OptionSetPredictTime(false),
			progressbar.OptionSetDescription(filepath.Base(ref)),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionClearOnFinish(),
		)
		defer bar.Clear()
		written, err = io.Copy(io.MultiWriter(file, bar), response.Body)
	} else {
		written, err = io.Copy(file, response.Body)
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("response status code %d invalid", response.StatusCode)
	}

	if err == nil && written < response.ContentLength {
		err = fmt.Errorf("expected %s but downloaded %s", humanize.Bytes(uint64(response.ContentLength)), humanize.Bytes(uint64(written)))
	}

	return
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
	var batch = semaphore.NewWeighted(int64(concurrent))
	var group errgroup.Group
	var mutex sync.Mutex

	for i := range refs {
		_ = batch.Acquire(context.TODO(), 1)

		ref, path := refs[i], paths[i]
		group.Go(func() (err error) {
			defer batch.Release(1)

			err = g.Download(eachTimeout, ref, path)
			if err != nil {
				mutex.Lock()
				errRefs = append(errRefs, ref)
				errors = append(errors, err)
				mutex.Unlock()
			}

			return
		})
	}

	_ = group.Wait()

	return
}

func (g *Getter) shouldSkipDownload(ctx context.Context, ref string, path string) (skip bool) {
	stat, err := os.Stat(path)
	if err != nil {
		return
	}

	if stat.Size() > 0 {
		headReq, headErr := http.NewRequestWithContext(ctx, http.MethodHead, ref, nil)
		if headErr != nil {
			return
		}
		resp, headErr := g.Client.Do(g.withHeader(headReq))
		if headErr != nil {
			return
		}
		if stat.Size() == resp.ContentLength {
			return true // skip download
		}
	}

	return
}
func (g *Getter) withHeader(request *http.Request) *http.Request {
	for head, content := range g.Header {
		request.Header.Set(head, content)
	}
	return request
}
