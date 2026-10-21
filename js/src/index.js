/// <reference types="@fastly/js-compute" />

addEventListener("fetch", (event) => event.respondWith(handleRequest(event)));

async function handleRequest(event) {
  const req = event.request;
  const url = new URL(req.url);

  // Starter template: Hello from Fastly Compute JavaScript!
  // Follow the tutorial in js/README.md to connect the backend API and configure edge caching.
  if (url.pathname === "/") {
    return new Response(
      `<!DOCTYPE html>
<html>
<head><title>Fastly Compute JS Starter</title></head>
<body>
<h1>Fastly Compute JS Starter</h1>
<p>Service is running! Follow the tutorial in <code>js/README.md</code> to connect the backend API.</p>
</body>
</html>`,
      {
        headers: { "Content-Type": "text/html; charset=utf-8" },
      }
    );
  }

  return new Response("Not Found", { status: 404 });
}
