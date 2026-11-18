package main

import (
	"context"

	"fastly-apero-go/internal/application"
	"fastly-apero-go/internal/fastlyadapter"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

func main() {
	// IoC : injection de l'implémentation Fastly.
	application.SetClient(fastlyadapter.Client{})

	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		application.Handle(ctx, fastlyadapter.NewResponseWriter(w), fastlyadapter.NewRequest(r))
	})
}
