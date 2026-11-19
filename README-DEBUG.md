# Debugging Fastly Compute Business Code Natively

Fastly Compute runs your code inside the Fastly WebAssembly runtime. That runtime is required for edge-specific features, but it is not the best place to debug ordinary business logic.

This document explains how to keep the same business code and add a native runtime adapter so you can:

- run the application locally without Fastly/Viceroy;
- use a normal debugger and breakpoints;
- write fast unit/integration tests;
- test routing, status codes, headers, query handling, and backend proxying.

The native runtime is for debugging business logic. Edge caching behavior must still be validated with `fastly compute serve` or on Fastly.

---

## Where to Start

Start from the Fastly entrypoint, not from a new native server.

1. Identify the Fastly entrypoint and list every Fastly SDK type/function it uses.
2. Separate the business rules from the runtime:
   - routes;
   - method rules;
   - TTL/SWR values;
   - backend URL construction;
   - status codes;
   - headers;
   - error responses.
3. Move those rules into an application layer that does not import the Fastly SDK.
4. Keep the Fastly entrypoint working through a Fastly adapter.
5. Add a native adapter and a native entrypoint that call the same application layer.

The native runtime must not change business behavior. It only replaces the host runtime.

---

## The 3 Steps

### Step 1 — DDD: Separate Business Logic

Goal: isolate runtime-independent application code.

Actions:

- Move the Fastly entrypoint into a dedicated file or binary.
- Extract routing, backend constants, cache policy, and response rules into an application layer.
- Keep the Fastly entrypoint as a thin host wrapper.
- Update the Fastly build configuration if the entrypoint path changes.
- Keep the existing Fastly behavior unchanged.

Rules:

- The application layer must not import Fastly SDK types.
- The application layer must not create Fastly requests or responses.
- The application layer must be testable without a Fastly runtime.

Expected result:

```text
application/
  backend.*      # constants, proxy logic, neutral request/response types
  handler.*      # routing, method checks, status codes
```

The Fastly entrypoint still works exactly as before.

---

### Step 2 — Interfaces/Ports: Define the Runtime Boundary

Goal: define the minimum contract the business code needs from any runtime.

Typical ports:

| Port | Responsibility |
|---|---|
| Request | method, path, raw query string |
| Response | status, headers, body |
| Header collection | set/add/get/delete/iterate headers |
| Backend client | create and send an upstream request |
| Cache options | TTL and stale-while-revalidate values |
| Response writer | write status, headers, body to the host |

Rules:

- Ports must describe the business need, not the Fastly SDK.
- The application layer depends only on ports.
- Runtime-specific code implements the ports.
- Cache options may be carried through the port even when a native runtime ignores them.

Then implement the Fastly adapter:

- wrap Fastly requests/responses;
- send to the named backend;
- apply TTL/SWR;
- copy headers, status, and body.

At the end of this step:

- the application layer contains the business logic;
- the Fastly adapter contains Fastly-specific behavior;
- the Fastly entrypoint only wires the adapter and starts the runtime.

---

### Step 3 — Dependency Injection: Add the Native Runtime

Goal: run the same application layer through a native adapter.

Actions:

- implement a native backend client;
- implement a native HTTP server entrypoint;
- inject the native adapter in the native composition root;
- inject the Fastly adapter in the Fastly composition root;
- listen on a local port such as `:8080`;
- ignore Fastly edge cache options in the native adapter unless you add a separate cache layer.

Composition roots are intentionally tiny:

```text
Fastly entrypoint:
  setClient(FastlyAdapter)
  start Fastly handler

Native entrypoint:
  setClient(NativeAdapter)
  listen on :8080
  call application.handle(request)
```

Expected result:

```text
application/          # business logic, no runtime SDK
fastly-adapter/       # Fastly SDK implementation
native-adapter/       # native HTTP implementation
fastly-entrypoint/    # Fastly composition root
native-entrypoint/    # native composition root
```

Debugging workflow:

```bash
# Fastly behavior and cache checks
fastly compute serve --watch

# Native business-logic debugging
run-native-server
curl -i http://localhost:8080/current?country=FR&city=Lyon
```

Use breakpoints and logs in the application layer, not in the adapter, when debugging business rules.

---

## Runtime Mapping by Language

### Go

| Concern | Fastly SDK | Native Go |
|---|---|---|
| HTTP entrypoint | `fsthttp.ServeFunc` | `net/http.Server` + `http.ServeMux` |
| Incoming request | `*fsthttp.Request` | `*http.Request` |
| Response writer | `fsthttp.ResponseWriter` | `http.ResponseWriter` |
| Headers | `fsthttp.Header` | `http.Header` |
| Backend request | `fsthttp.NewRequest` | `http.NewRequestWithContext` |
| Send to backend | `(*fsthttp.Request).Send(ctx, "backend_api")` | `http.Client.Do` |
| Edge cache options | `fsthttp.CacheOptions{TTL, StaleWhileRevalidate}` | ignored by native adapter |
| Error response | `fsthttp.Error` / `fsthttp.NotFound` | `http.Error` / `WriteHeader` |
| DI | `application.SetClient(fastlyadapter.Client{})` | `application.SetClient(httpadapter.Client{})` |
| Fastly entrypoint | `cmd/fastly/main.go` | — |
| Native entrypoint | — | `cmd/http/main.go` |

### JavaScript

| Concern | Fastly SDK | Native JavaScript |
|---|---|---|
| HTTP entrypoint | `addEventListener("fetch", ...)` + `event.respondWith` | `node:http.createServer` |
| Incoming request | `event.request` (`Request`) | `node:http` `IncomingMessage` |
| Response | `Response` | `node:http` `ServerResponse` |
| Headers | `Headers` | `rawHeaders` / `setHeader` |
| Backend request | `fetch(url, { backend: "backend_api" })` | `node:https.request` |
| Send to backend | `fetch` | `https.request` + `response` events |
| Edge cache options | `CacheOverride("override", { ttl, swr })` | ignored by native adapter |
| Error response | `new Response(status)` | `res.statusCode` + `res.end` |
| DI | `setClient(new FastlyClient())` | `setClient(new NativeClient())` |
| Fastly entrypoint | `src/fastly.js` | — |
| Native entrypoint | — | `src/http.js` |

### Rust

| Concern | Fastly SDK | Native Rust |
|---|---|---|
| HTTP entrypoint | `#[fastly::main]` | `std::net::TcpListener` + HTTP loop |
| Incoming request | `fastly::Request` | `application::Request` from `TcpStream` |
| Response | `fastly::Response` | `application::Response` written to `TcpStream` |
| Headers | Fastly request/response header methods | `ureq` request builder / raw HTTP headers |
| Backend request | `fastly::Request::get(url)` | `ureq::Agent::request` |
| Send to backend | `.send("backend_api")` | `.call()` |
| Edge cache options | `.with_ttl(...)` / `.with_stale_while_revalidate(...)` | ignored by native adapter |
| Error response | `Response::from_status(...)` | neutral `Response { status, ... }` |
| DI | `application::set_client(Box::new(FastlyClient))` | `application::set_client(Box::new(NativeClient))` |
| Fastly entrypoint | `src/bin/fastly.rs` | — |
| Native entrypoint | — | `src/bin/http.rs` |

Important: native adapters do not reproduce Fastly edge caching. Test TTL, SWR, cache hits, and surrogate behavior with Fastly.

---

## Shared Business Contract

All three implementations keep the same rules:

| Path | TTL | SWR |
|---|---:|---:|
| `/` | 3600 | 7200 |
| `/current` | 900 | 3600 |
| `/next` | 900 | 3600 |
| `/city` | 60 | 300 |

Shared behavior:

- only `GET` and `HEAD`;
- other methods return `405` with `Allow: GET, HEAD`;
- unknown paths return `404`;
- query strings are preserved;
- backend requests use `backend_api`;
- upstream host is `nominally-gentle-rodent.edgecompute.app`;
- response headers, status, and body are forwarded.

---

## Branch Solutions

Base branch: `solution-backend`.

### Go

| Step | Branch |
|---|---|
| Move Fastly entrypoint | `go/step1` |
| Ports + Fastly adapter + DI | `go/step2` |
| Native `net/http` adapter | `go/step3` |

Run native Go:

```bash
git checkout go/step3
cd go
go run ./cmd/http
```

### JavaScript

| Step | Branch |
|---|---|
| Move Fastly entrypoint | `js/step1` |
| Ports + Fastly adapter + DI | `js/step2` |
| Native Node HTTP adapter | `js/step3` |

Run native JavaScript:

```bash
git checkout js/step3
cd js
npm run start:http
```

With Bun:

```bash
bun ./src/http.js
```

### Rust

| Step | Branch |
|---|---|
| Move Fastly entrypoint | `rust/step1` |
| Ports + Fastly adapter + DI | `rust/step2` |
| Native HTTP adapter | `rust/step3` |

Run native Rust:

```bash
git checkout rust/step3
cd rust
cargo run --features native-http --bin http
```

All native servers listen on `127.0.0.1:8080`.

---

## Recommended Debugging Loop

1. Start the native server for the language you are working on.
2. Set breakpoints in `application/` or `internal/application/`.
3. Call the endpoint with `curl`.
4. Fix business logic.
5. Re-run native tests.
6. Validate Fastly-specific behavior with `fastly compute serve --watch`.
7. Keep the Fastly adapter thin; it should not contain business rules.
