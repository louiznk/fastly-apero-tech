package application

import (
	"context"
)

// Handle applique le dispatch métier sur une requête déjà enveloppée
// dans les ports de application.
func Handle(ctx context.Context, w ResponseWriter, r Request) {
	// Only allow GET and HEAD requests
	if r.Method() != MethodGet && r.Method() != MethodHead {
		MethodNotAllowed(w)
		return
	}

	switch r.Path() {
	case "/":
		// Landing page / help docs: 1 hour cache
		ProxyToBackend(ctx, w, r, 3600, 7200)

	case "/current":
		// Weather observation: 15 minutes cache, 1 hour SWR
		ProxyToBackend(ctx, w, r, 900, 3600)

	case "/next":
		// Hourly forecast: 15 minutes cache, 1 hour SWR
		ProxyToBackend(ctx, w, r, 900, 3600)

	case "/city":
		// City KV store lookup: 1 minute cache, 5 minutes SWR
		ProxyToBackend(ctx, w, r, 60, 300)

	default:
		NotFound(w)
	}
}
