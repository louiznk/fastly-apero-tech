# Fastly Compute Edge Caching in Go

This guide walks you through building, testing, and deploying a Fastly Compute edge caching application in **Go** that connects to an upstream Fastly Compute backend API.

---

## 📚 Official Documentation & References

- [Fastly Compute Go SDK Documentation (`compute-sdk-go`)](https://pkg.go.dev/github.com/fastly/compute-sdk-go)
- [Fastly Caching Concepts](https://www.fastly.com/documentation/guides/concepts/edge-state/cache/)
- [Fastly Backend Configuration Guide](https://www.fastly.com/documentation/guides/compute/backends/)
- [Fastly Compute Go Documentation](https://www.fastly.com/documentation/guides/compute/go/)

---

## 🛠️ 1. Prerequisites & Toolchain

Fastly Compute compiles Go to WebAssembly using the WASI target (`GOOS=wasip1 GOARCH=wasm`).

1. **Verify Go installation** (Go 1.21 or later is required):
   ```bash
   go version
   ```

2. **Fastly CLI**:
   Verify the Fastly CLI is installed and authenticated:
   ```bash
   fastly whoami
   ```

---

## 📦 2. Project Configuration

### `fastly.toml`
The Fastly configuration file defines the service manifest, build script, and local backend mapping for `fastly compute serve`:

```toml
manifest_version = 3
name = "fastly-apero-go"
description = "Fastly Compute edge caching tutorial application in Go"
authors = ["Fastly Workshop"]
language = "go"

[scripts]
build = "GOOS=wasip1 GOARCH=wasm go build -o bin/main.wasm ."

[local_server.backends.backend_api]
url = "https://nominally-gentle-rodent.edgecompute.app"
```

> [!TIP]
> Setting `[scripts.build]` instructs the Fastly CLI to compile using standard Go (`go build`) targeting WebAssembly WASI instead of defaulting to TinyGo.

### `go.mod`
The Go module depends on `github.com/fastly/compute-sdk-go`:

```go
module fastly-apero-go

go 1.21

require github.com/fastly/compute-sdk-go v1.4.0
```

---

## 💡 3. Fastly Go SDK Caching Mechanics

When sending requests to an upstream backend using `fsthttp.NewRequest`, you can control edge caching behavior through `fsthttp.Request.CacheOptions`:

```go
bereq.CacheOptions = fsthttp.CacheOptions{
    Pass:                 false, // If true, bypasses the cache entirely
    TTL:                  900,   // Cache at the edge for 15 minutes (in seconds)
    StaleWhileRevalidate: 3600,  // Serve stale cached content for up to 1 hour while refreshing in background
}
```

### Backend Forwarding Lifecycle:
1. Client sends `GET /current?country=FR&city=Lyon` to your edge service.
2. Your edge service creates a backend request with `backend_api` and applies `CacheOptions`.
3. Fastly checks its edge cache:
   - **Cache Hit**: Fastly immediately returns the cached response to your application.
   - **Cache Miss**: Fastly forwards the request to `https://nominally-gentle-rodent.edgecompute.app/current?country=FR&city=Lyon`, stores the response in the edge cache with the specified TTL, and returns it.

---

## 👨‍💻 4. Step-by-Step Implementation

### Step 1: Define Backend Constants
In `main.go`, specify the backend configuration:

```go
const (
	backendName = "backend_api"
	backendHost = "nominally-gentle-rodent.edgecompute.app"
)
```

### Step 2: Implement Upstream Proxy Handler
Create a helper function to proxy requests to `backend_api` with custom caching policies:

```go
func proxyToBackend(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request, ttl uint32, swr uint32) {
	// Construct the upstream URL preserving query parameters
	upstreamURL := fmt.Sprintf("https://%s%s", backendHost, r.URL.Path)
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	bereq, err := fsthttp.NewRequest(fsthttp.MethodGet, upstreamURL, nil)
	if err != nil {
		fsthttp.Error(w, err.Error(), fsthttp.StatusInternalServerError)
		return
	}

	// Important: Set the Host header expected by the upstream TLS server
	bereq.Header.Set("Host", backendHost)
	bereq.Header.Set("User-Agent", "fastly-compute-go-edge/1.0")

	// Configure Edge Caching policy
	bereq.CacheOptions = fsthttp.CacheOptions{
		TTL:                  ttl,
		StaleWhileRevalidate: swr,
	}

	// Dispatch request to named backend
	beresp, err := bereq.Send(ctx, backendName)
	if err != nil {
		fsthttp.Error(w, fmt.Sprintf("Error connecting to backend: %v", err), fsthttp.StatusBadGateway)
		return
	}
	defer beresp.Body.Close()

	// Forward response headers and status code to client
	for k, v := range beresp.Header {
		for _, val := range v {
			w.Header().Add(k, val)
		}
	}
	w.WriteHeader(beresp.StatusCode)
	io.Copy(w, beresp.Body)
}
```

### Step 3: Handle Routing
In your `fsthttp.ServeFunc` router:
- `/`: Serve an HTML documentation/landing page or proxy `/` to upstream.
- `/current`: Proxy with `TTL = 900` (15m) and `SWR = 3600` (1h).
- `/next`: Proxy hourly forecast with `TTL = 900` and `SWR = 3600`.
- `/city`: Proxy city lookup with `TTL = 60` (1m) and `SWR = 300` (5m).

---

## 🧪 5. Local Testing

Run the local development server:

```bash
fastly compute serve --watch
```

Test the endpoints using `curl`:

```bash
# 1. Test landing page
curl -i http://localhost:7676/

# 2. Test current weather endpoint
curl -i "http://localhost:7676/current?country=FR&city=Lyon"

# 3. Test next 24h hourly forecast endpoint
curl -i "http://localhost:7676/next?country=FR&city=Paris"

# 4. Test city metadata endpoint
curl -i "http://localhost:7676/city?country=FR&city=Lyon"
```

---

## 🚀 6. Deploying to Fastly

Deploy your compiled WebAssembly package to Fastly:

```bash
fastly compute deploy
```

On first deployment, the CLI prompts:
1. `Create new service? [y/N]`: `y`
2. Service name: `fastly-apero-go`
3. Backend configuration:
   - Backend name: `backend_api`
   - Host/Domain: `nominally-gentle-rodent.edgecompute.app`
   - Port: `443`
   - TLS: `yes`

Once deployed, query your live `.edgecompute.app` domain!

---

## 🔍 Reference Solution

To view the complete working Go solution:

```bash
git checkout solution-backend
cat go/main.go
```
