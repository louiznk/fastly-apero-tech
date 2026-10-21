package main

import (
	"context"
	"fmt"
	"io"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

const (
	backendName = "backend_api"
	backendHost = "nominally-gentle-rodent.edgecompute.app"
)

func proxyToBackend(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request, ttl uint32, swr uint32) {
	upstreamURL := fmt.Sprintf("https://%s%s", backendHost, r.URL.Path)
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	bereq, err := fsthttp.NewRequest(fsthttp.MethodGet, upstreamURL, nil)
	if err != nil {
		fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
		return
	}

	bereq.Header.Set("Host", backendHost)
	bereq.Header.Set("User-Agent", "fastly-compute-go-edge/1.0")

	// Apply Fastly Edge Cache Options (TTL & Stale-While-Revalidate)
	bereq.CacheOptions = &fsthttp.CacheOptions{
		TTL:                  ttl,
		StaleWhileRevalidate: swr,
	}

	beresp, err := bereq.Send(ctx, backendName)
	if err != nil {
		fsthttp.Error(w, fmt.Sprintf("Backend error: %v", err), fsthttp.StatusBadGateway)
		return
	}
	defer beresp.Body.Close()

	for k, v := range beresp.Header {
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}
	w.WriteHeader(beresp.StatusCode)
	io.Copy(w, beresp.Body)
}

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		// Only allow GET and HEAD requests
		if r.Method != fsthttp.MethodGet && r.Method != fsthttp.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			fsthttp.Error(w, "This method is not allowed", fsthttp.StatusMethodNotAllowed)
			return
		}

		switch r.URL.Path {
		case "/":
			// Landing page / help docs: 1 hour cache
			proxyToBackend(ctx, w, r, 3600, 7200)

		case "/current":
			// Weather observation: 15 minutes cache, 1 hour SWR
			proxyToBackend(ctx, w, r, 900, 3600)

		case "/next":
			// Hourly forecast: 15 minutes cache, 1 hour SWR
			proxyToBackend(ctx, w, r, 900, 3600)

		case "/city":
			// City KV store lookup: 1 minute cache, 5 minutes SWR
			proxyToBackend(ctx, w, r, 60, 300)

		default:
			fsthttp.NotFound(w, r)
		}
	})
}
