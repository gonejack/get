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

func (g *Getter) Download(ref string, path string, timeout time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	return g.DownloadWithContext(ctx, ref, path)
}
func (g *Getter) DownloadWithContext(ctx context.Context, ref string, path string) (err error) {
	if g.shouldSkip(ctx, ref, path) {
		return
	}

	file, err := os.Create(path)
	if err != nil {
		return
	}
	defer file.Close()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ref, nil)
	if err != nil {
		return
	}
	response, err := g.Client.Do(g.patchHeader(request))
	if err != nil {
		return
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("response status code %d invalid", response.StatusCode)
	}

	var writer io.Writer = file
	if g.Verbose {
		bar := g.progressBar(ref, response.ContentLength)
		defer func() {
			_ = bar.Clear()
			_ = bar.Close()
		}()
		writer = io.MultiWriter(file, bar)
	}

	wrote, err := io.Copy(writer, response.Body)
	if err == nil && wrote < response.ContentLength {
		err = fmt.Errorf("expected %s but downloaded %s", humanize.Bytes(uint64(response.ContentLength)), humanize.Bytes(uint64(wrote)))
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

			err = g.Download(ref, path, eachTimeout)
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

func (g *Getter) shouldSkip(ctx context.Context, ref string, path string) (skip bool) {
	stat, err := os.Stat(path)
	if err != nil {
		return
	}
	if stat.Size() == 0 {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, ref, nil)
	if err != nil {
		return
	}
	resp, err := g.Client.Do(g.patchHeader(req))
	if err != nil {
		return
	}
	if stat.Size() == resp.ContentLength {
		return true // skip download
	}

	return
}
func (g *Getter) patchHeader(request *http.Request) *http.Request {
	for head, content := range g.Header {
		request.Header.Set(head, content)
	}
	return request
}
func (g *Getter) progressBar(ref string, size int64) *progressbar.ProgressBar {
	return progressbar.NewOptions64(size,
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
}
