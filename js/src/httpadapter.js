import http from "node:http";
import https from "node:https";

/// Native HTTP implementation of the application client port.
///
/// This adapter intentionally ignores edge cache options; the native
/// runtime has no Fastly edge cache.
export class NativeClient {
  send(request) {
    const url = new URL(request.url);
    const transport = url.protocol === "https:" ? https : http;

    return new Promise((resolve, reject) => {
      const nativeRequest = transport.request(
        {
          protocol: url.protocol,
          hostname: url.hostname,
          port: url.port || undefined,
          path: `${url.pathname}${url.search}`,
          method: request.method,
          headers: Object.fromEntries(request.headers),
        },
        (response) => {
          const chunks = [];

          response.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
          response.on("end", () => {
            const headers = [];
            for (let index = 0; index < response.rawHeaders.length; index += 2) {
              headers.push([
                response.rawHeaders[index],
                response.rawHeaders[index + 1],
              ]);
            }

            resolve({
              status: response.statusCode ?? 502,
              headers,
              body: new Uint8Array(Buffer.concat(chunks)),
            });
          });
        },
      );

      nativeRequest.on("error", reject);
      nativeRequest.end();
    });
  }
}
