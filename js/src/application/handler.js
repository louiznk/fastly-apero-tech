import { proxyToBackend, textResponse } from "./backend.js";

const METHOD_GET = "GET";
const METHOD_HEAD = "HEAD";

export async function handle(request) {
  if (request.method !== METHOD_GET && request.method !== METHOD_HEAD) {
    return textResponse(405, "This method is not allowed\n", [
      ["Allow", "GET, HEAD"],
    ]);
  }

  switch (request.path) {
    case "/":
      // Landing page / help docs: 1 hour cache
      return proxyToBackend(request, 3600, 7200);

    case "/current":
      // Weather observation: 15 minutes cache, 1 hour SWR
      return proxyToBackend(request, 900, 3600);

    case "/next":
      // Hourly forecast: 15 minutes cache, 1 hour SWR
      return proxyToBackend(request, 900, 3600);

    case "/city":
      // City KV store lookup: 1 minute cache, 5 minutes SWR
      return proxyToBackend(request, 60, 300);

    default:
      return textResponse(404, "The page you requested could not be found\n");
  }
}
