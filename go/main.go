package main

import (
	"context"
	"fmt"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		// Starter template: Hello from Fastly Compute Go!
		// Follow the tutorial in go/README.md to connect the backend API and configure edge caching.
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Fastly Compute Go Starter</title></head>
<body>
<h1>Fastly Compute Go Starter</h1>
<p>Service is running! Follow the tutorial in <code>go/README.md</code> to connect the backend API.</p>
</body>
</html>`)
		default:
			fsthttp.NotFound(w, r)
		}
	})
}
