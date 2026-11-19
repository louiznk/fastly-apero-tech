use std::fmt;

pub const BACKEND_NAME: &str = "backend_api";
pub const BACKEND_HOST: &str = "nominally-gentle-rodent.edgecompute.app";

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct CacheOptions {
    pub ttl: u32,
    pub stale_while_revalidate: u32,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Request {
    pub method: String,
    pub path: String,
    pub raw_query: String,
}

impl Request {
    pub fn new(
        method: impl Into<String>,
        path: impl Into<String>,
        raw_query: impl Into<String>,
    ) -> Self {
        Self {
            method: method.into(),
            path: path.into(),
            raw_query: raw_query.into(),
        }
    }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct BackendRequest {
    pub method: String,
    pub url: String,
    pub headers: Vec<(String, String)>,
    pub cache_options: CacheOptions,
}

impl BackendRequest {
    pub fn new(
        method: impl Into<String>,
        url: impl Into<String>,
        cache_options: CacheOptions,
    ) -> Self {
        Self {
            method: method.into(),
            url: url.into(),
            headers: Vec::new(),
            cache_options,
        }
    }

    pub fn set_header(&mut self, name: impl Into<String>, value: impl Into<String>) {
        let name = name.into();
        let value = value.into();

        if let Some((_, existing)) = self
            .headers
            .iter_mut()
            .find(|(key, _)| key.eq_ignore_ascii_case(&name))
        {
            *existing = value;
        } else {
            self.headers.push((name, value));
        }
    }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Response {
    pub status: u16,
    pub headers: Vec<(String, String)>,
    pub body: Vec<u8>,
}

impl Response {
    pub fn new(status: u16) -> Self {
        Self {
            status,
            headers: Vec::new(),
            body: Vec::new(),
        }
    }

    pub fn text(status: u16, body: impl Into<String>) -> Self {
        let mut response = Self::new(status);
        response.headers.push((
            "content-type".to_string(),
            "text/plain; charset=utf-8".to_string(),
        ));
        response.body = body.into().into_bytes();
        response
    }

    pub fn with_header(mut self, name: impl Into<String>, value: impl Into<String>) -> Self {
        self.headers.push((name.into(), value.into()));
        self
    }

    pub fn set_header(&mut self, name: impl Into<String>, value: impl Into<String>) {
        let name = name.into();
        let value = value.into();

        if let Some((_, existing)) = self
            .headers
            .iter_mut()
            .find(|(key, _)| key.eq_ignore_ascii_case(&name))
        {
            *existing = value;
        } else {
            self.headers.push((name, value));
        }
    }
}

impl fmt::Display for Response {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{} ({} bytes)", self.status, self.body.len())
    }
}

/// Port implemented by each runtime adapter.
pub trait Client: Send + Sync {
    fn send(&self, request: BackendRequest) -> Result<Response, String>;
}

pub fn proxy_to_backend(
    client: &dyn Client,
    request: &Request,
    ttl: u32,
    stale_while_revalidate: u32,
) -> Response {
    let mut url = format!("https://{BACKEND_HOST}{}", request.path);
    if !request.raw_query.is_empty() {
        url.push('?');
        url.push_str(&request.raw_query);
    }

    let mut backend_request = BackendRequest::new(
        "GET",
        url,
        CacheOptions {
            ttl,
            stale_while_revalidate,
        },
    );
    backend_request.set_header("Host", BACKEND_HOST);
    backend_request.set_header("User-Agent", "fastly-compute-rust-edge/1.0");

    match client.send(backend_request) {
        Ok(response) => response,
        Err(error) => Response::text(502, format!("Backend error: {error}\n")),
    }
}
