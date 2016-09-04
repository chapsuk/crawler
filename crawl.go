package crawler

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	DefaultWorkersCount = 150
)

var (
	errAnotherDomain = errors.New("another domain")
	errIsMailTo      = errors.New("is mailto")
	errIsAnchor      = errors.New("is anchor")
)

type File interface {
	GetPath() string
	GetBody() []byte
	Free()
}

type Crawler struct {
	endpoint string
	mainURL  *url.URL
	output   string

	UploadWorkers     int
	SaveWorkers       int
	EnableGzip        bool
	IncludeSubDomains bool

	saved   map[string]bool
	visited map[string]bool

	uploadPageCh  chan string
	uploadAssetCh chan string
	saveCh        chan File

	vmu sync.Mutex
	smu sync.Mutex
	wg  sync.WaitGroup

	httpClient *http.Client
}

// New return new Crawler instance
func New(h, o string) (*Crawler, error) {
	m, err := url.Parse(h)
	if err != nil {
		return nil, err
	}
	if m.Host == "" {
		return nil, errors.New("empty main host")
	}
	return &Crawler{
		endpoint:          h,
		mainURL:           m,
		output:            o,
		saved:             make(map[string]bool),
		visited:           make(map[string]bool),
		uploadPageCh:      make(chan string, 1024),
		uploadAssetCh:     make(chan string, 1024),
		saveCh:            make(chan File, 128),
		UploadWorkers:     DefaultWorkersCount,
		SaveWorkers:       DefaultWorkersCount,
		IncludeSubDomains: false,
		EnableGzip:        true,
		httpClient: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   15 * time.Second,
					KeepAlive: 180 * time.Second,
				}).Dial,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}, nil
}

// Run crawler proccess
func (c *Crawler) Run(resume bool) {
	defer c.close()
	c.runWorkers()
	c.enqueUploadPage(c.endpoint)
	c.wg.Wait()
}

func (c *Crawler) enqueUploadPage(url string) {
	if !c.isVisited(url) {
		c.wg.Add(1)
		go func() { c.uploadPageCh <- url }()
	}
}

func (c *Crawler) enqueUploadAsset(url string) {
	if !c.isVisited(url) {
		c.wg.Add(1)
		go func() { c.uploadAssetCh <- url }()
	}
}

func (c *Crawler) enqueSave(f File) {
	if !c.isSaved(f.GetPath()) {
		c.wg.Add(1)
		go func() { c.saveCh <- f }()
	}
}

func (c *Crawler) runWorkers() {
	for i := 0; i < c.UploadWorkers; i++ {
		go c.serveUploadPage()
		go c.serveUploadAsset()
	}
	for i := 0; i < c.SaveWorkers; i++ {
		go c.serveSave()
	}
}

func (c *Crawler) close() {
	close(c.uploadAssetCh)
	close(c.uploadPageCh)
	close(c.saveCh)
}

func (c *Crawler) serveUploadPage() {
	for {
		url := <-c.uploadPageCh
		if url == "" {
			log.Printf("gotten empty url")
			c.wg.Done()
			continue
		}

		err := c.addVisited(url)
		if err != nil {
			// error only if visited before
			c.wg.Done()
			continue
		}

		req, err := craeteRequest(url)
		if err != nil {
			log.Printf("create request error: %s", err)
			c.wg.Done()
			continue
		}

		res, err := c.httpClient.Do(req)
		if err != nil {
			log.Printf("http get: %s, error: %s", url, err)
			c.wg.Done()
			continue
		}

		page, err := NewPage(url, res)
		if err != nil {
			log.Printf("create page error: %s", err)
			c.wg.Done()
			continue
		}

		c.enqueSave(page)

		for _, purl := range page.Pages {
			u, err := c.normalizeURL(purl)
			if err != nil {
				if err != errAnotherDomain && err != errIsMailTo && err != errIsAnchor {
					log.Printf("fail normilize url: %s error: %s", purl, err)
				}
				continue
			}
			c.enqueUploadPage(u)
		}

		for _, aurl := range page.Assets {
			u, err := c.normalizeURL(aurl)
			if err != nil {
				if err != errAnotherDomain {
					log.Printf("fail normilize url: %s error: %s", aurl, err)
				}
				continue
			}
			c.enqueUploadAsset(u)
		}
		c.wg.Done()
	}
}

func (c *Crawler) serveUploadAsset() {
	for {
		url := <-c.uploadAssetCh
		if url == "" {
			log.Printf("empty asset url")
			c.wg.Done()
			continue
		}

		err := c.addVisited(url)
		if err != nil {
			// error only if visited before
			c.wg.Done()
			continue
		}

		res, err := http.Get(url)
		if err != nil {
			// todo handle http errors and:
			//   - remove visited flags and push to upload channel
			// or
			//   - wg.Done() and continue
			log.Printf("http get: %s, error: %s", url, err)
			c.removeVisited(url)
			c.wg.Done()
			continue
		}

		asset, err := NewAsset(url, res)
		if err != nil {
			log.Printf("create asset error: %s", err)
			c.wg.Done()
			continue
		}

		c.enqueSave(asset)
		c.wg.Done()
	}
}

func (c *Crawler) serveSave() {
	for {
		f := <-c.saveCh

		err := c.addSaved(f.GetPath())
		if err != nil {
			c.wg.Done()
			f.Free()
			continue
		}

		name, err := c.getOutputFileNameByURL(f.GetPath())
		if err != nil {
			log.Printf("get file name for url: %s, error: %s", f.GetPath(), err)
			c.wg.Done()
			f.Free()
			continue
		}
		path := c.output + name

		// create dir if not exists
		dir, _ := filepath.Split(path)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0744)
			if err != nil {
				log.Printf("craete dir: %s error: %s", dir, err)
				c.wg.Done()
				f.Free()
				continue
			}
		}

		if c.EnableGzip {
			path += ".gz"
			var b bytes.Buffer
			w := gzip.NewWriter(&b)
			w.Write(f.GetBody())
			w.Close()
			err = ioutil.WriteFile(path, b.Bytes(), 0666)
		} else {
			err = ioutil.WriteFile(path, f.GetBody(), 0666)
		}

		if err != nil {
			log.Printf("wrte file error: %s", err)
		}

		c.wg.Done()
		f.Free()
	}
}

// todo add cancelation context
func craeteRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return req, err
}

func (c *Crawler) addVisited(url string) error {
	c.vmu.Lock()
	defer c.vmu.Unlock()
	if _, ok := c.visited[url]; ok {
		return fmt.Errorf("fail add visited url %s, visited before", url)
	}
	c.visited[url] = true
	return nil
}

func (c *Crawler) isVisited(url string) bool {
	c.vmu.Lock()
	defer c.vmu.Unlock()
	if v, ok := c.visited[url]; ok {
		return v
	}
	return false
}

func (c *Crawler) removeVisited(url string) error {
	c.vmu.Lock()
	defer c.vmu.Unlock()
	if _, ok := c.visited[url]; ok {
		c.visited[url] = false
		return nil
	}
	return fmt.Errorf("not visited url: %s", url)
}

func (c *Crawler) addSaved(url string) error {
	c.smu.Lock()
	defer c.smu.Unlock()
	if _, ok := c.saved[url]; ok {
		return fmt.Errorf("fail add saved url %s, saved before", url)
	}
	c.saved[url] = true
	return nil
}

func (c *Crawler) isSaved(url string) bool {
	c.smu.Lock()
	defer c.smu.Unlock()
	_, ok := c.saved[url]
	return ok
}

func (c *Crawler) normalizeURL(u string) (string, error) {
	u = strings.TrimSpace(strings.Trim(u, "\n"))
	if strings.Contains(u, "mailto") {
		return "", errIsMailTo
	}

	if strings.Contains(u, "#") {
		return "", errIsAnchor
	}

	t, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	if t.Host == "" {
		t.Host = c.mainURL.Host
		t.Scheme = c.mainURL.Scheme
	}
	if t.Scheme == "" {
		t.Scheme = c.mainURL.Scheme
	}

	// check doamin
	if c.IncludeSubDomains {
		if !strings.Contains(t.Host, c.mainURL.Host) {
			return "", errAnotherDomain
		}
	} else {
		if t.Host != c.mainURL.Host {
			return "", errAnotherDomain
		}
	}

	t.RawQuery = ""
	return t.String(), nil
}

func (c *Crawler) getOutputFileNameByURL(u string) (string, error) {
	t, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	if t.Host == "" {
		t.Host = c.mainURL.Host
	}

	if t.Path == "" {
		return t.Host + "/index.html", nil
	}

	name := t.Host + t.Path
	_, file := filepath.Split(t.Path)
	if file == "" || !strings.Contains(file, ".") || file == t.Host {
		if t.Path[len(t.Path)-1] == '/' {
			return name + "index.html", nil
		}
		return name + "/index.html", nil
	}

	return name, nil
}
