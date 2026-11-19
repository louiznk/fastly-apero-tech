mod backend;
mod handler;

pub use backend::{
    proxy_to_backend, BackendRequest, CacheOptions, Client, Request, Response, BACKEND_HOST,
    BACKEND_NAME,
};
use std::sync::OnceLock;

static CLIENT: OnceLock<Box<dyn Client>> = OnceLock::new();

/// Inject the runtime-specific client, mirroring the Go `SetClient` pattern.
pub fn set_client(client: Box<dyn Client>) {
    let _ = CLIENT.set(client);
}

fn client() -> Option<&'static dyn Client> {
    CLIENT.get().map(|client| client.as_ref())
}

pub fn handle_with_client(client: &dyn Client, request: &Request) -> Response {
    handler::handle(client, request)
}

pub fn handle(request: &Request) -> Response {
    match client() {
        Some(client) => handle_with_client(client, request),
        None => Response::text(500, "runtime client not injected (SetClient)\n"),
    }
}
