package main

import (
	"log"
	"net/http"

	"fastly-apero-go/internal/application"
	"fastly-apero-go/internal/httpadapter"
)

func main() {
	// IoC : injection de l'implémentation net/http native.
	application.SetClient(httpadapter.Client{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		application.Handle(r.Context(), httpadapter.NewResponseWriter(w), httpadapter.NewRequest(r))
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
