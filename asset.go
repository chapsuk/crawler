package crawler

import (
	"io/ioutil"
	"net/http"
)

type Asset struct {
	body []byte
	path string
}

func NewAsset(path string, res *http.Response) (*Asset, error) {
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return &Asset{
		path: path,
		body: body,
	}, nil
}

func (a *Asset) GetBody() []byte {
	return a.body
}

func (a *Asset) GetPath() string {
	return a.path
}

func (a *Asset) Free() {
	a.body = nil
}
