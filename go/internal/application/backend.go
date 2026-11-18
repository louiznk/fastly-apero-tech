package application

import (
	"context"
	"fmt"
	"io"
)

const (
	backendName = "backend_api"
	backendHost = "nominally-gentle-rodent.edgecompute.app"
)

// ---------------------------------------------------------------------------
// Ports (interfaces) : le métier ne dépend que de ceci. Chaque runtime
// (Fastly Compute, net/http natif, ...) fournit sa propre implémentation.
// ---------------------------------------------------------------------------

// Header abstraît les manipulations d'en-têtes, indépendamment du runtime.
type Header interface {
	Set(key, value string)
	Add(key, value string)
	Get(key string) string
	Del(key string)
	Each(fn func(key string, values []string))
}

// ResponseWriter correspond au ResponseWriter du runtime hôte.
type ResponseWriter interface {
	Header() Header
	WriteHeader(statusCode int)
	Write(p []byte) (int, error)
}

// CacheOptions reprend les options de cache edge de fsthttp.
type CacheOptions struct {
	TTL                  uint32
	StaleWhileRevalidate uint32
}

// Response correspond à la réponse renvoyée par le backend.
type Response struct {
	StatusCode int
	Header     Header
	Body       io.ReadCloser
}

// Méthodes HTTP supportées, symétrie avec fsthttp.MethodGet et
// net/http.MethodGet.
const (
	MethodGet  = "GET"
	MethodHead = "HEAD"
)

// Request correspond à la requête entrante du runtime hôte.
type Request interface {
	Method() string
	Path() string
	RawQuery() string
}

// BackendRequest est une requête sortante vers un backend, avec les
// méthodes spécifiques au runtime (ex. fsthttp.Request.Send).
type BackendRequest interface {
	Header() Header
	Send(ctx context.Context, backend string, opts CacheOptions) (*Response, error)
}

// Client crée les requêtes sortantes. Injecté par le composition root (IoC).
type Client interface {
	NewRequest(method, url string) (BackendRequest, error)
}

// upstream est injecté par chaque main (fastly ou natif).
var upstream Client

// SetClient injecte l'implémentation runtime (pattern IoC).
func SetClient(c Client) { upstream = c }

// ---------------------------------------------------------------------------
// Métier
// ---------------------------------------------------------------------------

func ProxyToBackend(ctx context.Context, w ResponseWriter, r Request, ttl uint32, swr uint32) {
	beresp, done := forward(ctx, w, r, ttl, swr)
	if done {
		return
	}
	defer beresp.Body.Close()

	copyHeaders(w.Header(), beresp.Header)
	w.WriteHeader(beresp.StatusCode)
	io.Copy(w, beresp.Body)
}

// NotFound réplique fsthttp.NotFound via le port.
func NotFound(w ResponseWriter) {
	w.WriteHeader(404)
	fmt.Fprint(w, "404 page not found")
}

// MethodNotAllowed réplique fsthttp.Error 405 via le port.
func MethodNotAllowed(w ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD")
	WriteError(w, "This method is not allowed", 405)
}

// WriteError réplique fsthttp.Error via le port.
func WriteError(w ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, msg)
}

func copyHeaders(dst, src Header) {
	src.Each(func(key string, values []string) {
		for _, val := range values {
			dst.Add(key, val)
		}
	})
}

func forward(ctx context.Context, w ResponseWriter, r Request, ttl uint32, swr uint32) (*Response, bool) {
	if upstream == nil {
		WriteError(w, "runtime client not injected (SetClient)", 500)
		return nil, true
	}

	upstreamURL := fmt.Sprintf("https://%s%s", backendHost, r.Path())
	if r.RawQuery() != "" {
		upstreamURL += "?" + r.RawQuery()
	}

	bereq, err := upstream.NewRequest("GET", upstreamURL)
	if err != nil {
		WriteError(w, err.Error(), 500)
		return nil, true
	}

	bereq.Header().Set("Host", backendHost)
	bereq.Header().Set("User-Agent", "fastly-compute-go-edge/1.0")

	// Le Send (et les CacheOptions) est spécifique au runtime :
	// c'est l'implémentation injectée qui le porte.
	beresp, err := bereq.Send(ctx, backendName, CacheOptions{
		TTL:                  ttl,
		StaleWhileRevalidate: swr,
	})
	if err != nil {
		WriteError(w, fmt.Sprintf("Backend error: %v", err), 502)
		return nil, true
	}
	return beresp, false
}
