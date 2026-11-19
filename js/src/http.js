import http from "node:http";

import { setClient } from "./application/backend.js";
import { handle } from "./application/handler.js";
import { NativeClient } from "./httpadapter.js";

setClient(new NativeClient());

const server = http.createServer(async (request, response) => {
  try {
    const url = new URL(
      request.url ?? "/",
      `http://${request.headers.host ?? "localhost"}`,
    );

    const applicationRequest = {
      method: request.method ?? "GET",
      path: url.pathname,
      rawQuery: url.search.startsWith("?") ? url.search.slice(1) : url.search,
    };

    const applicationResponse = await handle(applicationRequest);
    const headers = new Map();

    for (const [name, value] of applicationResponse.headers) {
      const key = name.toLowerCase();
      if (
        key === "content-length" ||
        key === "transfer-encoding" ||
        key === "connection"
      ) {
        continue;
      }

      const values = headers.get(key);
      if (values) {
        values.push(value);
      } else {
        headers.set(key, [value]);
      }
    }

    for (const [name, values] of headers) {
      response.setHeader(name, values.length === 1 ? values[0] : values);
    }

    response.statusCode = applicationResponse.status;
    if (applicationRequest.method === "HEAD") {
      response.end();
    } else {
      response.end(Buffer.from(applicationResponse.body));
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    response.statusCode = 500;
    response.end(`Internal error: ${message}\n`);
  }
});

server.listen(8080, "127.0.0.1", () => {
  console.log("listening on :8080");
});
