package crawler

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Page is html single page structure
type Page struct {
	path   string
	body   []byte
	Assets []string
	Pages  []string
}

// NewPage parse http response, find assets links and pages links.
// Return page instance or error.
// The response's body is closed on return.
func NewPage(path string, res *http.Response) (*Page, error) {
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	pgs, asts := parseDoc(doc)

	return &Page{
		path:   path,
		body:   body,
		Assets: asts,
		Pages:  pgs,
	}, nil
}

// GetBody return file body bytes
func (p *Page) GetBody() []byte {
	return p.body
}

// GetPath return file path
func (p *Page) GetPath() string {
	return p.path
}

// Free set body to nil
func (p *Page) Free() {
	p.body = nil
}

// GetType return page type
func (p *Page) GetType() ItemType {
	return PageType
}

func parseDoc(doc *goquery.Document) (pages []string, assets []string) {
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		p, ok := s.Attr("href")
		if !ok {
			// a without href, why not
			return
		}
		pages = append(pages, p)
	})

	tags := map[string]string{
		"link":   "href",
		"script": "src",
	}

	for tag, attr := range tags {
		doc.Find(tag).Each(func(i int, s *goquery.Selection) {
			a, ok := s.Attr(attr)
			if !ok {
				if tag != "script" {
					log.Printf("bad %s tag item: %s", tag, s.Text())
				}
				return
			}
			if strings.Contains(a, ".css") || strings.Contains(a, ".js") {
				assets = append(assets, a)
			}
		})
	}
	return
}
