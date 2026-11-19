use fastly::http::StatusCode;
use fastly::{Error, Request, Response};
use fastly_apero_rust::application::{
    self, Request as ApplicationRequest, Response as ApplicationResponse,
};
use fastly_apero_rust::fastlyadapter::FastlyClient;

#[fastly::main]
fn main(req: Request) -> Result<Response, Error> {
    application::set_client(Box::new(FastlyClient));

    let request = ApplicationRequest::new(
        req.get_method_str(),
        req.get_path(),
        req.get_query_str().unwrap_or_default(),
    );
    let response = application::handle(&request);

    Ok(to_fastly_response(response))
}

fn to_fastly_response(response: ApplicationResponse) -> Response {
    let status = StatusCode::from_u16(response.status).unwrap_or(StatusCode::INTERNAL_SERVER_ERROR);
    let mut fastly_response = Response::from_status(status);

    for (name, value) in response.headers {
        fastly_response.append_header(name.as_str(), value.as_str());
    }

    fastly_response.with_body(response.body)
}
