package proxy

import (
	"net/http"
	"net/url"
)

type Transport struct {
	Proxy func(r *http.Request) (*url.URL, error)
}
