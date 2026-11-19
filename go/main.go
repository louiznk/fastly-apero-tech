package main

import (
	"context"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"fastly-apero-go/pkg/adapter/fastly"
	"fastly-apero-go/pkg/app"
)

func main() {
	adapter := fastly.NewFastlyAdapter()
	application := app.NewApp(adapter.AdaptBackendClient())

	fsthttp.ServeFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		err := application.Handle(ctx, adapter.AdaptResponseWriter(w), adapter.AdaptRequest(r))
		if err != nil {
			if appErr, ok := err.(*app.Error); ok {
				fsthttp.Error(w, appErr.Message, appErr.StatusCode)
			} else {
				fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
			}
		}
	})
}
