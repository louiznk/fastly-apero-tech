import { CacheOverride } from "fastly:cache-override";

import { BACKEND_NAME } from "./application/backend.js";

/// Fastly Compute implementation of the application client port.
export class FastlyClient {
  async send(request) {
    const cacheOverride = new CacheOverride("override", {
      ttl: request.cacheOptions.ttl,
      swr: request.cacheOptions.swr,
    });

    const response = await fetch(request.url, {
      backend: BACKEND_NAME,
      headers: Object.fromEntries(request.headers),
      cacheOverride,
    });

    const headers = [];
    for (const [name, value] of response.headers) {
      headers.push([name, value]);
    }

    return {
      status: response.status,
      headers,
      body: new Uint8Array(await response.arrayBuffer()),
    };
  }
}

export function toFastlyResponse(response) {
  return new Response(response.body, {
    status: response.status,
    headers: response.headers,
  });
}
