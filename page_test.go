package crawler_test

import (
	"net/http"
	"testing"

	"github.com/chapsuk/crawler"
)

func TestNewPage(t *testing.T) {
	url := "https://github.com/chapsuk"
	res, err := http.Get(url)
	if err != nil {
		t.Errorf("http, get: %s error: %s", url, err)
	}

	_, err = crawler.NewPage(url, res)
	if err != nil {
		t.Errorf("create page error: %s", err)
	}
}
