use std::error::Error as StdError;
use std::fmt;

use tokio::task::JoinError;
use tokio::time::error::Elapsed;
use tokio_tungstenite::tungstenite::Error as WsError;

#[derive(Debug, Clone)]
pub enum WirechatError {
    NotConnected,
    AlreadyConnected,
    UrlEmpty,
    Connect(String),
    Timeout(String),
    Protocol(String),
    WebSocket(String),
    Serde(String),
    Server { code: String, message: String },
    Task(String),
}

impl fmt::Display for WirechatError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            WirechatError::NotConnected => write!(f, "not connected"),
            WirechatError::AlreadyConnected => write!(f, "already connected"),
            WirechatError::UrlEmpty => write!(f, "url is empty"),
            WirechatError::Connect(msg) => write!(f, "connect error: {msg}"),
            WirechatError::Timeout(msg) => write!(f, "timeout: {msg}"),
            WirechatError::Protocol(msg) => write!(f, "protocol error: {msg}"),
            WirechatError::WebSocket(msg) => write!(f, "websocket error: {msg}"),
            WirechatError::Serde(msg) => write!(f, "serde error: {msg}"),
            WirechatError::Server { code, message } => write!(f, "server error {code}: {message}"),
            WirechatError::Task(msg) => write!(f, "task error: {msg}"),
        }
    }
}

impl StdError for WirechatError {}

impl From<WsError> for WirechatError {
    fn from(err: WsError) -> Self {
        WirechatError::WebSocket(err.to_string())
    }
}

impl From<serde_json::Error> for WirechatError {
    fn from(err: serde_json::Error) -> Self {
        WirechatError::Serde(err.to_string())
    }
}

impl From<Elapsed> for WirechatError {
    fn from(err: Elapsed) -> Self {
        WirechatError::Timeout(err.to_string())
    }
}

impl From<JoinError> for WirechatError {
    fn from(err: JoinError) -> Self {
        WirechatError::Task(err.to_string())
    }
}
