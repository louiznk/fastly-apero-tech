// Package fastlyadapter adapte les types du SDK Fastly Compute
// (github.com/fastly/compute-sdk-go/fsthttp) aux ports définis dans
// internal/application. C'est l'implémentation injectée dans le main
// Fastly (cmd/fastly).
package fastlyadapter

import (
	"context"

	"fastly-apero-go/internal/application"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

// Client implémente application.Client avec fsthttp.
type Client struct{}

func (Client) NewRequest(method, url string) (application.BackendRequest, error) {
	req, err := fsthttp.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	return BackendRequest{req: req}, nil
}

// BackendRequest enveloppe *fsthttp.Request : Send porte les
// CacheOptions et le send spécifique Fastly.
type BackendRequest struct{ req *fsthttp.Request }

func (b BackendRequest) Header() application.Header {
	return Header{h: b.req.Header}
}

func (b BackendRequest) Send(ctx context.Context, backend string, opts application.CacheOptions) (*application.Response, error) {
	b.req.CacheOptions = fsthttp.CacheOptions{
		TTL:                  opts.TTL,
		StaleWhileRevalidate: opts.StaleWhileRevalidate,
	}
	resp, err := b.req.Send(ctx, backend)
	if err != nil {
		return nil, err
	}
	return &application.Response{
		StatusCode: resp.StatusCode,
		Header:     Header{h: resp.Header},
		Body:       resp.Body,
	}, nil
}

// ResponseWriter enveloppe fsthttp.ResponseWriter.
type ResponseWriter struct{ w fsthttp.ResponseWriter }

func NewResponseWriter(w fsthttp.ResponseWriter) application.ResponseWriter {
	return ResponseWriter{w: w}
}

func (rw ResponseWriter) Header() application.Header {
	return Header{h: rw.w.Header()}
}

func (rw ResponseWriter) WriteHeader(statusCode int) {
	rw.w.WriteHeader(statusCode)
}

func (rw ResponseWriter) Write(p []byte) (int, error) {
	return rw.w.Write(p)
}

// Request enveloppe *fsthttp.Request entrante.
type Request struct{ req *fsthttp.Request }

func NewRequest(r *fsthttp.Request) application.Request {
	return Request{req: r}
}

func (r Request) Method() string { return r.req.Method }

func (r Request) Path() string { return r.req.URL.Path }

func (r Request) RawQuery() string { return r.req.URL.RawQuery }

// Header enveloppe fsthttp.Header (clés en minuscules côté Fastly).
type Header struct{ h fsthttp.Header }

func (h Header) Set(key, value string) { h.h.Set(key, value) }

func (h Header) Add(key, value string) { h.h.Add(key, value) }

func (h Header) Get(key string) string { return h.h.Get(key) }

func (h Header) Del(key string) { h.h.Del(key) }

func (h Header) Each(fn func(key string, values []string)) {
	for k, v := range h.h {
		fn(k, v)
	}
}
