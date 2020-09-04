package httppool

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type HttpConPool struct {
	Conn *http.Client
}

var once sync.Once
var hpool *HttpConPool

// NewHttpPool
func NewHttpPool(max_conn_per, max_idle_conn_per int, duration int64) *HttpConPool {
	once.Do(func() {
		hpool = new(HttpConPool)
		hpool.Conn = &http.Client{
			Transport: &http.Transport{
				MaxConnsPerHost:     max_conn_per,
				MaxIdleConnsPerHost: max_idle_conn_per,
			},
			Timeout: time.Duration(duration) * time.Second,
		}
	})
	return hpool
}

// send a http request of post or get
func (h *HttpConPool) Request(url string, method string, data string, header map[string]string) (string, error, int) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return "", err, 0
	}

	for h, v := range header {
		req.Header.Set(h, v)
	}

	response, err := h.Conn.Do(req)

	if err != nil {
		return "", err, 0
	} else if response != nil {
		defer response.Body.Close()

		r_body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return "", err, response.StatusCode
		} else {
			return string(r_body), nil, response.StatusCode
		}
	} else {
		return "", nil, 0
	}
}
