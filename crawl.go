package crawler

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	GetType() ItemType
	Free()
}

type Crawler struct {
	endpoint string
	output   string

	UploadWorkers     int
	SaveWorkers       int
	EnableGzip        bool
	IncludeSubDomains bool

	uploadPageCh  chan string
	uploadAssetCh chan string
	saveCh        chan File

	mainURL    *url.URL
	state      *State
	httpClient *http.Client
}

// New return new Crawler instance
func New(h, o string, s *State) (*Crawler, error) {
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
		uploadPageCh:      make(chan string, 1024),
		uploadAssetCh:     make(chan string, 1024),
		saveCh:            make(chan File, 128),
		UploadWorkers:     DefaultWorkersCount,
		SaveWorkers:       DefaultWorkersCount,
		IncludeSubDomains: false,
		EnableGzip:        true,
		state:             s,
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
func (c *Crawler) Run() {
	defer c.close()
	c.runWorkers()

	// if state empty start from main url
	if c.state.IsEmpty() {
		c.enqueUploadPage(c.endpoint)
	} else {
		for url, i := range c.state.GetInflight() {
			switch i.itype {
			case PageType:
				c.enqueUploadPage(url)
			case AssetType:
				c.enqueUploadAsset(url)
			default:
				log.Printf("undefined type: %d from state", i.itype)
			}
		}
	}
	c.state.WaiteAll()
}

func (c *Crawler) enqueUploadPage(url string) {
	if err := c.state.MarkAsInFlight(url, PageType); err != nil {
		if err != errHasMoreOrEqualStatus {
			log.Printf("enqueUploadPage, mark as in flight error: %s", err)
		}
		return
	}
	go func() { c.uploadPageCh <- url }()
}

func (c *Crawler) enqueUploadAsset(url string) {
	if err := c.state.MarkAsInFlight(url, AssetType); err != nil {
		if err != errHasMoreOrEqualStatus {
			log.Printf("enqueUploadAsset, mark as in flight error: %s", err)
		}
		return
	}
	go func() { c.uploadAssetCh <- url }()
}

func (c *Crawler) enqueSave(f File) {
	if !c.state.IsSaved(f.GetPath()) {
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
			c.state.MarkAsIgnored(url, PageType)
			continue
		}

		req, err := craeteRequest(url)
		if err != nil {
			log.Printf("create request error: %s", err)
			c.state.MarkAsIgnored(url, PageType)
			continue
		}

		res, err := c.httpClient.Do(req)
		if err != nil {
			log.Printf("http get: %s, error: %s", url, err)
			c.state.MarkAsIgnored(url, PageType)
			continue
		}

		page, err := NewPage(url, res)
		if err != nil {
			log.Printf("create page error: %s", err)
			c.state.MarkAsIgnored(url, PageType)
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
	}
}

func (c *Crawler) serveUploadAsset() {
	for {
		url := <-c.uploadAssetCh
		if url == "" {
			log.Printf("empty asset url")
			c.state.MarkAsIgnored(url, AssetType)
			continue
		}

		res, err := http.Get(url)
		if err != nil {
			log.Printf("http get: %s, error: %s", url, err)
			c.state.MarkAsIgnored(url, AssetType)
			continue
		}

		asset, err := NewAsset(url, res)
		if err != nil {
			log.Printf("create asset error: %s", err)
			c.state.MarkAsIgnored(url, AssetType)
			continue
		}

		c.enqueSave(asset)
	}
}

func (c *Crawler) serveSave() {
	for {
		f := <-c.saveCh

		name, err := c.getOutputFileNameByURL(f.GetPath())
		if err != nil {
			log.Printf("get file name for url: %s, error: %s", f.GetPath(), err)
			c.state.MarkAsIgnored(f.GetPath(), f.GetType())
			continue
		}
		path := c.output + name

		// create dir if not exists
		dir, _ := filepath.Split(path)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0744)
			if err != nil {
				log.Printf("craete dir: %s error: %s", dir, err)
				c.state.MarkAsIgnored(f.GetPath(), f.GetType())
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

		c.state.MarkAsSaved(f.GetPath(), f.GetType())
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
