use std::io::{BufRead, BufReader, Write};
use std::net::{TcpListener, TcpStream};
use std::thread;

use fastly_apero_rust::application::{self, Request, Response};
use fastly_apero_rust::httpadapter::NativeClient;

fn main() {
    application::set_client(Box::new(NativeClient));

    let listener = TcpListener::bind("127.0.0.1:8080").expect("failed to bind 127.0.0.1:8080");
    println!("listening on :8080");

    for stream in listener.incoming() {
        match stream {
            Ok(stream) => {
                thread::spawn(move || handle_connection(stream));
            }
            Err(error) => eprintln!("connection error: {error}"),
        }
    }
}

fn handle_connection(mut stream: TcpStream) {
    let mut reader = match stream.try_clone() {
        Ok(stream) => BufReader::new(stream),
        Err(error) => {
            eprintln!("failed to clone stream: {error}");
            return;
        }
    };

    let mut request_line = String::new();
    if reader.read_line(&mut request_line).unwrap_or(0) == 0 {
        return;
    }

    let mut parts = request_line.split_whitespace();
    let method = parts.next().unwrap_or("GET").to_string();
    let target = parts.next().unwrap_or("/").to_string();

    let (path, raw_query) = match target.split_once('?') {
        Some((path, query)) => (path, query),
        None => (target.as_str(), ""),
    };

    loop {
        let mut line = String::new();
        if reader.read_line(&mut line).unwrap_or(0) == 0 || line == "\r\n" || line == "\n" {
            break;
        }
    }

    let request = Request::new(method.clone(), path, raw_query);
    let response = application::handle(&request);

    if let Err(error) = write_response(&mut stream, &method, response) {
        eprintln!("failed to write response: {error}");
    }
}

fn write_response(stream: &mut TcpStream, method: &str, response: Response) -> std::io::Result<()> {
    let mut head = format!(
        "HTTP/1.1 {} {}\r\n",
        response.status,
        status_reason(response.status)
    );

    for (name, value) in &response.headers {
        if name.eq_ignore_ascii_case("content-length")
            || name.eq_ignore_ascii_case("transfer-encoding")
            || name.eq_ignore_ascii_case("connection")
        {
            continue;
        }
        head.push_str(&format!("{name}: {value}\r\n"));
    }

    head.push_str(&format!("Content-Length: {}\r\n", response.body.len()));
    head.push_str("Connection: close\r\n\r\n");

    stream.write_all(head.as_bytes())?;
    if method != "HEAD" {
        stream.write_all(&response.body)?;
    }
    stream.flush()
}

fn status_reason(status: u16) -> &'static str {
    match status {
        200 => "OK",
        201 => "Created",
        202 => "Accepted",
        204 => "No Content",
        301 => "Moved Permanently",
        302 => "Found",
        304 => "Not Modified",
        400 => "Bad Request",
        401 => "Unauthorized",
        403 => "Forbidden",
        404 => "Not Found",
        405 => "Method Not Allowed",
        500 => "Internal Server Error",
        502 => "Bad Gateway",
        503 => "Service Unavailable",
        _ => "Unknown",
    }
}
