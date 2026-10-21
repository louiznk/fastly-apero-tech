# Fastly Compute Edge Caching in JavaScript

This guide walks you through building, testing, and deploying a Fastly Compute edge caching application in **JavaScript** that connects to an upstream Fastly Compute backend API.

---

## 📚 Official Documentation & References

- [Fastly Compute JavaScript Reference Docs (`@fastly/js-compute`)](https://js-compute-reference-docs.edgecompute.app/)
- [Fastly Caching Concepts](https://www.fastly.com/documentation/guides/concepts/edge-state/cache/)
- [Fastly Backend Configuration Guide](https://www.fastly.com/documentation/guides/compute/backends/)
- [Fastly Compute JavaScript Guide](https://www.fastly.com/documentation/guides/compute/javascript/)

---

## 🛠️ 1. Prerequisites & Toolchain

Fastly Compute compiles JavaScript to WebAssembly using the `@fastly/js-compute` runtime and toolchain.

1. **Verify Node.js & npm** (Node.js 18 or later is recommended):
   ```bash
   node --version
   npm --version
   ```

2. **Install dependencies**:
   ```bash
   npm install
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
name = "fastly-apero-js"
description = "Fastly Compute edge caching tutorial application in JavaScript"
authors = ["Fastly Workshop"]
language = "javascript"

[local_server.backends.backend_api]
url = "https://nominally-gentle-rodent.edgecompute.app"
```

### `package.json`
Specifies `@fastly/js-compute` and the build script:

```json
{
  "name": "fastly-apero-js",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "build": "js-compute-runtime ./src/index.js ./bin/main.wasm"
  },
  "dependencies": {
    "@fastly/js-compute": "^3.16.0"
  }
}
```

---

## 💡 3. Fastly JS SDK Caching Mechanics: `CacheOverride`

In Fastly Compute JavaScript, you configure custom edge caching behavior using the `CacheOverride` class from `fastly:cache-override`:

```javascript
import { CacheOverride } from "fastly:cache-override";

// Override origin caching headers with custom TTL and Stale-While-Revalidate
const cacheOverride = new CacheOverride("override", {
  ttl: 900,   // Cache at the edge for 15 minutes (900 seconds)
  swr: 3600,  // Serve stale content for up to 1 hour while updating in background
});

const beresp = await fetch(url, {
  backend: "backend_api",
  cacheOverride,
});
```

### Modes of `CacheOverride`:
- `"none"`: Respect standard HTTP response caching headers from the origin.
- `"pass"`: Bypass the edge cache entirely (always fetch from origin).
- `"override"`: Enforce explicit TTL, SWR, and surrogate key overrides regardless of origin headers.

---

## 👨‍💻 4. Step-by-Step Implementation

### Step 1: Define Backend Constants
In `src/index.js`:

```javascript
const BACKEND_NAME = "backend_api";
const BACKEND_HOST = "nominally-gentle-rodent.edgecompute.app";
```

### Step 2: Implement Upstream Proxy Function
Create a helper function to proxy requests to `backend_api`:

```javascript
import { CacheOverride } from "fastly:cache-override";

async function proxyToBackend(url, ttl = 900, swr = 3600) {
  const targetUrl = `https://${BACKEND_HOST}${url.pathname}${url.search}`;

  const cacheOverride = new CacheOverride("override", {
    ttl,
    swr,
  });

  const response = await fetch(targetUrl, {
    backend: BACKEND_NAME,
    headers: {
      Host: BACKEND_HOST,
      "User-Agent": "fastly-compute-js-edge/1.0",
    },
    cacheOverride,
  });

  return response;
}
```

### Step 3: Handle Routing
In your `handleRequest` function:

```javascript
async function handleRequest(event) {
  const req = event.request;
  const url = new URL(req.url);

  switch (url.pathname) {
    case "/":
      // Serve landing page or proxy / to upstream
      return proxyToBackend(url, 3600, 7200);

    case "/current":
      // Weather observation: 15m edge cache, 1h SWR
      return proxyToBackend(url, 900, 3600);

    case "/next":
      // Hourly forecast: 15m edge cache, 1h SWR
      return proxyToBackend(url, 900, 3600);

    case "/city":
      // City KV metadata: 1m edge cache, 5m SWR
      return proxyToBackend(url, 60, 300);

    default:
      return new Response("Not Found", { status: 404 });
  }
}
```

---

## 🧪 5. Local Testing

1. Install dependencies:
   ```bash
   npm install
   ```

2. Run the local development server:
   ```bash
   fastly compute serve --watch
   ```

3. Test the endpoints using `curl`:
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
2. Service name: `fastly-apero-js`
3. Backend configuration:
   - Backend name: `backend_api`
   - Host/Domain: `nominally-gentle-rodent.edgecompute.app`
   - Port: `443`
   - TLS: `yes`

Once deployed, query your live `.edgecompute.app` domain!

---

## 🔍 Reference Solution

To view the complete working JavaScript solution:

```bash
git checkout solution-backend
cat js/src/index.js
```
