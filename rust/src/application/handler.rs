use super::backend::{proxy_to_backend, Client, Request, Response};

const METHOD_GET: &str = "GET";
const METHOD_HEAD: &str = "HEAD";

pub fn handle(client: &dyn Client, request: &Request) -> Response {
    if request.method != METHOD_GET && request.method != METHOD_HEAD {
        return Response::text(405, "This method is not allowed\n")
            .with_header("Allow", "GET, HEAD");
    }

    match request.path.as_str() {
        "/" => proxy_to_backend(client, request, 3600, 7200),
        "/current" => proxy_to_backend(client, request, 900, 3600),
        "/next" => proxy_to_backend(client, request, 900, 3600),
        "/city" => proxy_to_backend(client, request, 60, 300),
        _ => Response::text(404, "The page you requested could not be found\n"),
    }
}
