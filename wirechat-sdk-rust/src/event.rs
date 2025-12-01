use serde::{Deserialize, Serialize};

use crate::error::WirechatError;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageEvent {
    pub room: String,
    pub user: String,
    pub text: String,
    pub ts: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserEvent {
    pub room: String,
    pub user: String,
}

#[derive(Debug, Clone)]
pub enum Event {
    Message(MessageEvent),
    UserJoined(UserEvent),
    UserLeft(UserEvent),
    Error(WirechatError),
}
