export const BACKEND_NAME = "backend_api";
export const BACKEND_HOST = "nominally-gentle-rodent.edgecompute.app";

let upstreamClient = null;

/// Inject the runtime-specific client, mirroring the Go SetClient pattern.
export function setClient(client) {
  upstreamClient = client;
}

function client() {
  if (!upstreamClient) {
    throw new Error("runtime client not injected (setClient)");
  }
  return upstreamClient;
}

export function buildUpstreamUrl(path, rawQuery) {
  const query = rawQuery ? `?${rawQuery}` : "";
  return `https://${BACKEND_HOST}${path}${query}`;
}

export function applicationResponse(status, body = new Uint8Array(), headers = []) {
  return { status, headers, body };
}

export function textResponse(status, text, headers = []) {
  return applicationResponse(status, new TextEncoder().encode(text), [
    ["content-type", "text/plain; charset=utf-8"],
    ...headers,
  ]);
}

export async function proxyToBackend(request, ttl, swr) {
  try {
    return await client().send({
      method: "GET",
      url: buildUpstreamUrl(request.path, request.rawQuery),
      headers: [
        ["Host", BACKEND_HOST],
        ["User-Agent", "fastly-compute-js-edge/1.0"],
      ],
      cacheOptions: { ttl, swr },
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return textResponse(502, `Backend error: ${message}\n`);
  }
}
