import { setClient } from "./application/backend.js";
import { handle } from "./application/handler.js";
import { FastlyClient, toFastlyResponse } from "./fastlyadapter.js";

setClient(new FastlyClient());

addEventListener("fetch", (event) => {
  event.respondWith(handleRequest(event));
});

async function handleRequest(event) {
  const request = event.request;
  const url = new URL(request.url);

  const applicationRequest = {
    method: request.method,
    path: url.pathname,
    rawQuery: url.search.startsWith("?") ? url.search.slice(1) : url.search,
  };

  const applicationResponse = await handle(applicationRequest);
  return toFastlyResponse(applicationResponse);
}
