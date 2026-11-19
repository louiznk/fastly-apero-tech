use fastly::Request as FastlyRequest;

use crate::application::{BackendRequest, Client, Response, BACKEND_NAME};

/// Fastly Compute implementation of the application client port.
pub struct FastlyClient;

impl Client for FastlyClient {
    fn send(&self, request: BackendRequest) -> Result<Response, String> {
        let mut backend_request = FastlyRequest::get(&request.url)
            .with_ttl(request.cache_options.ttl)
            .with_stale_while_revalidate(request.cache_options.stale_while_revalidate);

        for (name, value) in &request.headers {
            backend_request.set_header(name.as_str(), value.as_str());
        }

        let backend_response = backend_request
            .send(BACKEND_NAME)
            .map_err(|error| error.to_string())?;
        let status = backend_response.get_status().as_u16();
        let headers = backend_response
            .get_headers()
            .map(|(name, value)| {
                (
                    name.to_string(),
                    value.to_str().unwrap_or_default().to_string(),
                )
            })
            .collect();
        let body = backend_response.into_body_bytes();

        Ok(Response {
            status,
            headers,
            body,
        })
    }
}
