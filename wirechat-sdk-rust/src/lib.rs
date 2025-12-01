pub mod client;
pub mod config;
pub mod error;
pub mod event;

pub use client::Client;
pub use config::Config;
pub use error::WirechatError;
pub use event::{Event, MessageEvent, UserEvent};

pub const PROTOCOL_VERSION: u8 = 1;
