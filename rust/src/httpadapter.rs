use std::io::Read;

use crate::application::{BackendRequest, Client, Response};

/// Native HTTP implementation of the application client port.
///
/// This adapter intentionally ignores edge cache options; the native
/// runtime has no Fastly edge cache.
pub struct NativeClient;

impl Client for NativeClient {
    fn send(&self, request: BackendRequest) -> Result<Response, String> {
        let agent = ureq::AgentBuilder::new().build();
        let mut native_request = agent.request(&request.method, &request.url);

        for (name, value) in &request.headers {
            native_request = native_request.set(name, value);
        }

        match native_request.call() {
            Ok(response) => response_from_ureq(response),
            Err(ureq::Error::Status(_, response)) => response_from_ureq(response),
            Err(error) => Err(error.to_string()),
        }
    }
}

fn response_from_ureq(response: ureq::Response) -> Result<Response, String> {
    let status = response.status();

    let mut headers = Vec::new();
    for name in response.headers_names() {
        for value in response.all(&name) {
            headers.push((name.clone(), value.to_string()));
        }
    }

    let mut body = Vec::new();
    response
        .into_reader()
        .read_to_end(&mut body)
        .map_err(|error| error.to_string())?;

    Ok(Response {
        status,
        headers,
        body,
    })
}
