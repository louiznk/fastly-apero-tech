// Package httpadapter adapte l'API native net/http de Go aux ports
// définis dans internal/application. C'est l'implémentation injectée
// dans le main natif (cmd/http).
package httpadapter

import (
	"context"
	"net/http"

	"fastly-apero-go/internal/application"
)

// Client implémente application.Client avec net/http.
type Client struct {
	HTTP *http.Client
}

func (c Client) NewRequest(method, url string) (application.BackendRequest, error) {
	return BackendRequest{
		method: method,
		url:    url,
		header: make(http.Header),
		http:   c.HTTP,
	}, nil
}

// BackendRequest porte la requête sortante : Send joue le rôle de
// fsthttp.Request.Send mais passe par net/http (pas de cache edge).
type BackendRequest struct {
	method string
	url    string
	header http.Header
	http   *http.Client
}

func (b BackendRequest) Header() application.Header {
	return Header{h: b.header}
}

func (b BackendRequest) Send(ctx context.Context, backend string, opts application.CacheOptions) (*application.Response, error) {
	if b.http == nil {
		b.http = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, b.method, b.url, nil)
	if err != nil {
		return nil, err
	}

	// net/http ignore un header "Host" posé via Header() : il doit
	// passer par le champ dédié de la requête.
	if host := b.header.Get("Host"); host != "" {
		req.Host = host
	}
	header := b.header.Clone()
	header.Del("Host")
	req.Header = header

	resp, err := b.http.Do(req)
	if err != nil {
		return nil, err
	}
	return &application.Response{
		StatusCode: resp.StatusCode,
		Header:     Header{h: resp.Header},
		Body:       resp.Body,
	}, nil
}

// ResponseWriter enveloppe http.ResponseWriter.
type ResponseWriter struct{ w http.ResponseWriter }

func NewResponseWriter(w http.ResponseWriter) application.ResponseWriter {
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

// Request enveloppe *http.Request entrante.
type Request struct{ req *http.Request }

func NewRequest(r *http.Request) application.Request {
	return Request{req: r}
}

func (r Request) Method() string { return r.req.Method }

func (r Request) Path() string { return r.req.URL.Path }

func (r Request) RawQuery() string { return r.req.URL.RawQuery }

// Header enveloppe http.Header.
type Header struct{ h http.Header }

func (h Header) Set(key, value string) { h.h.Set(key, value) }

func (h Header) Add(key, value string) { h.h.Add(key, value) }

func (h Header) Get(key string) string { return h.h.Get(key) }

func (h Header) Del(key string) { h.h.Del(key) }

func (h Header) Each(fn func(key string, values []string)) {
	for k, v := range h.h {
		fn(k, v)
	}
}
