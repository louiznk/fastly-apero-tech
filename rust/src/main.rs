use fastly::http::{header, Method, StatusCode};
use fastly::{mime, Error, Request, Response};

#[fastly::main]
fn main(req: Request) -> Result<Response, Error> {
    // Starter template: Hello from Fastly Compute Rust!
    // Follow the tutorial in rust/README.md to connect the backend API and configure edge caching.
    match req.get_method() {
        &Method::GET | &Method::HEAD => (),
        _ => {
            return Ok(Response::from_status(StatusCode::METHOD_NOT_ALLOWED)
                .with_header(header::ALLOW, "GET, HEAD")
                .with_body_text_plain("This method is not allowed\n"))
        }
    }

    match req.get_path() {
        "/" => Ok(Response::from_status(StatusCode::OK)
            .with_header(header::CONTENT_TYPE, mime::TEXT_HTML_UTF_8.as_ref())
            .with_body(
                r#"<!DOCTYPE html>
<html>
<head><title>Fastly Compute Rust Starter</title></head>
<body>
<h1>Fastly Compute Rust Starter</h1>
<p>Service is running! Follow the tutorial in <code>rust/README.md</code> to connect the backend API.</p>
</body>
</html>"#,
            )),
        _ => Ok(Response::from_status(StatusCode::NOT_FOUND)
            .with_body_text_plain("The page you requested could not be found\n")),
    }
}
