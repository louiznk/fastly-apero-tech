/// <reference types="@fastly/js-compute" />
import { CacheOverride } from "fastly:cache-override";

const BACKEND_NAME = "backend_api";
const BACKEND_HOST = "nominally-gentle-rodent.edgecompute.app";

addEventListener("fetch", (event) => event.respondWith(handleRequest(event)));

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

async function handleRequest(event) {
  const req = event.request;

  if (req.method !== "GET" && req.method !== "HEAD") {
    return new Response("This method is not allowed\n", {
      status: 405,
      headers: { Allow: "GET, HEAD" },
    });
  }

  const url = new URL(req.url);

  switch (url.pathname) {
    case "/":
      // Landing page / help docs: 1 hour cache
      return proxyToBackend(url, 3600, 7200);

    case "/current":
      // Weather observation: 15 minutes cache, 1 hour SWR
      return proxyToBackend(url, 900, 3600);

    case "/next":
      // Hourly forecast: 15 minutes cache, 1 hour SWR
      return proxyToBackend(url, 900, 3600);

    case "/city":
      // City KV store lookup: 1 minute cache, 5 minutes SWR
      return proxyToBackend(url, 60, 300);

    default:
      return new Response("The page you requested could not be found\n", {
        status: 404,
      });
  }
}
