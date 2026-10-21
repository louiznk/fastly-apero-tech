use fastly::http::{header, Method, StatusCode};
use fastly::{Error, Request, Response};

const BACKEND_NAME: &str = "backend_api";
const BACKEND_HOST: &str = "nominally-gentle-rodent.edgecompute.app";

fn proxy_to_backend(req: &Request, ttl_secs: u32, swr_secs: u32) -> Result<Response, Error> {
    let path = req.get_path();
    let query = req.get_query_str();

    let url = match query {
        Some(q) if !q.is_empty() => format!("https://{BACKEND_HOST}{path}?{q}"),
        _ => format!("https://{BACKEND_HOST}{path}"),
    };

    let bereq = Request::get(url)
        .with_header(header::HOST, BACKEND_HOST)
        .with_header(header::USER_AGENT, "fastly-compute-rust-edge/1.0")
        .with_ttl(ttl_secs)
        .with_stale_while_revalidate(swr_secs);

    let beresp = bereq.send(BACKEND_NAME)?;
    Ok(beresp)
}

#[fastly::main]
fn main(req: Request) -> Result<Response, Error> {
    match req.get_method() {
        &Method::GET | &Method::HEAD => (),
        _ => {
            return Ok(Response::from_status(StatusCode::METHOD_NOT_ALLOWED)
                .with_header(header::ALLOW, "GET, HEAD")
                .with_body_text_plain("This method is not allowed\n"))
        }
    }

    match req.get_path() {
        "/" => proxy_to_backend(&req, 3600, 7200),
        "/current" => proxy_to_backend(&req, 900, 3600),
        "/next" => proxy_to_backend(&req, 900, 3600),
        "/city" => proxy_to_backend(&req, 60, 300),
        _ => Ok(Response::from_status(StatusCode::NOT_FOUND)
            .with_body_text_plain("The page you requested could not be found\n")),
    }
}
