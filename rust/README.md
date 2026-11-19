# Fastly Compute Edge Caching in Rust

This guide walks you through building, testing, and deploying a Fastly Compute edge caching application in **Rust** that connects to an upstream Fastly Compute backend API.

---

## 📚 Official Documentation & References

- [Fastly Compute Rust SDK Documentation (`fastly`)](https://docs.rs/fastly/latest/fastly/)
- [Fastly Caching Concepts](https://www.fastly.com/documentation/guides/concepts/edge-state/cache/)
- [Fastly Backend Configuration Guide](https://www.fastly.com/documentation/guides/compute/backends/)
- [Fastly Compute Rust Guide](https://www.fastly.com/documentation/guides/compute/rust/)

---

## 🛠️ 1. Prerequisites & Toolchain

Fastly Compute compiles Rust to WebAssembly using the WASI target (`wasm32-wasip1`).

1. **Verify Rust toolchain**:
   ```bash
   rustc --version
   cargo --version
   ```

2. **Add WASI target**:
   ```bash
   rustup target add wasm32-wasip1
   ```

3. **Fastly CLI**:
   Verify the Fastly CLI is installed and authenticated:
   ```bash
   fastly whoami
   ```

---

## 📦 2. Project Configuration

### `fastly.toml`
The Fastly configuration file defines the service manifest and local backend mapping for `fastly compute serve`:

```toml
manifest_version = 3
name = "fastly-apero-rust"
description = "Fastly Compute edge caching tutorial application in Rust"
authors = ["Fastly Workshop"]
language = "rust"

[local_server.backends.backend_api]
url = "https://nominally-gentle-rodent.edgecompute.app"
```

### `Cargo.toml`
Specifies dependencies on `fastly`:

```toml
[package]
name = "fastly-apero-rust"
version = "0.1.0"
edition = "2021"

[features]
default = []
fastly = ["dep:fastly"]

[dependencies]
fastly = { version = "0.13", optional = true }
```

### DDD Architecture

The application is split into runtime-agnostic business logic and runtime-specific adapters:

- `src/application/` — ports and business logic (`handle`, `proxy_to_backend`). It does not import the Fastly SDK.
- `src/fastlyadapter.rs` — adapts the Fastly Compute SDK to the application ports.
- `src/httpadapter.rs` — adapts a native blocking HTTP client to the application ports.
- `src/bin/fastly.rs` — composition root for Fastly Compute; injects the Fastly adapter with `application::set_client`.
- `src/bin/http.rs` — composition root for native HTTP; injects the native adapter with `application::set_client`.

This separation allows running the same business logic on Fastly Compute and on a native HTTP server.

### `rust-toolchain.toml`
Pin the WASI target:

```toml
[toolchain]
channel = "stable"
targets = ["wasm32-wasip1"]
```

---

## 💡 3. Fastly Rust SDK Caching Mechanics

In Fastly Compute Rust, you can configure edge caching policies using builder methods on `fastly::Request`:

```rust
let mut bereq = Request::get(upstream_url)
    .with_header(header::HOST, backend_host)
    .with_ttl(900)                          // Cache at the edge for 15 minutes (900 seconds)
    .with_stale_while_revalidate(3600);    // Serve stale content for up to 1 hour while updating

let beresp = bereq.send("backend_api")?;
```

### Request Cache Methods:
- `.with_ttl(secs)`: Overrides response TTL stored in Fastly edge cache.
- `.with_stale_while_revalidate(secs)`: Specifies the window during which Fastly can serve a stale cached response while asynchronously fetching a fresh copy from origin.
- `.with_pass(true)`: Bypasses Fastly's edge cache lookup and insertion entirely.
- `.with_surrogate_key("key")`: Appends surrogate keys for targeted cache purging.

---

## 👨‍💻 4. Step-by-Step Implementation

### Step 1: Define Backend Constants
These constants live in `src/application/backend.rs`:

```rust
const BACKEND_NAME: &str = "backend_api";
const BACKEND_HOST: &str = "nominally-gentle-rodent.edgecompute.app";
```

### Step 2: Implement Upstream Proxy Function
Create a helper function to proxy requests to `backend_api`:

```rust
fn proxy_to_backend(req: &Request, ttl_secs: u32, swr_secs: u32) -> Result<Response, Error> {
    let path = req.get_path();
    let query = req.get_query_str();

    let url = match query {
        Some(q) if !q.is_empty() => format!("https://{BACKEND_HOST}{path}?{q}"),
        _ => format!("https://{BACKEND_HOST}{path}"),
    };

    let bereq = Request::get(url)
        .with_header(header::HOST, BACKEND_HOST)
        .with_header(header::USER_AGENT, "fastly-compute-rust-edge/1.0")
        .with_ttl(ttl_secs)
        .with_stale_while_revalidate(swr_secs);

    let beresp = bereq.send(BACKEND_NAME)?;
    Ok(beresp)
}
```

This logic lives in `src/application/backend.rs`.

### Step 3: Handle Routing
The runtime-agnostic routing logic lives in `src/application/handler.rs`. `src/bin/fastly.rs` is the Fastly composition root:

```rust
#[fastly::main]
fn main(req: Request) -> Result<Response, Error> {
    match req.get_method() {
        &Method::GET | &Method::HEAD => (),
        _ => {
            return Ok(Response::from_status(StatusCode::METHOD_NOT_ALLOWED)
                .with_header(header::ALLOW, "GET, HEAD")
                .with_body_text_plain("This method is not allowed\n"))
        }
    }

    match req.get_path() {
        "/" => proxy_to_backend(&req, 3600, 7200),
        "/current" => proxy_to_backend(&req, 900, 3600),
        "/next" => proxy_to_backend(&req, 900, 3600),
        "/city" => proxy_to_backend(&req, 60, 300),
        _ => Ok(Response::from_status(StatusCode::NOT_FOUND)
            .with_body_text_plain("The page you requested could not be found\n")),
    }
}
```

---

## 🧪 5. Local Testing

1. Run the local development server:
   ```bash
   fastly compute serve --watch
   ```

2. Test the endpoints using `curl`:
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

### Running the native Rust version (optional)

The same business logic can run as a standard HTTP server:

```bash
cargo run --features native-http --bin http
```

The server listens on `:8080`. You can test the same endpoints:

```bash
curl -i http://localhost:8080/current?country=FR&city=Lyon
```

---

## 🚀 6. Deploying to Fastly

Deploy your compiled WebAssembly package to Fastly:

```bash
fastly compute deploy
```

On first deployment, the CLI prompts:
1. `Create new service? [y/N]`: `y`
2. Service name: `fastly-apero-rust`
3. Backend configuration:
   - Backend name: `backend_api`
   - Host/Domain: `nominally-gentle-rodent.edgecompute.app`
   - Port: `443`
   - TLS: `yes`

Once deployed, query your live `.edgecompute.app` domain!

---

## 🔍 Reference Solution

To view the complete working Rust solution:

```bash
git checkout solution-backend
cat rust/src/bin/fastly.rs
```
