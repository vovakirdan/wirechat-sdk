use std::time::Duration;

/// Controls how the client connects to Wirechat.
#[derive(Debug, Clone)]
pub struct Config {
    pub url: String,
    pub token: Option<String>,
    pub user: Option<String>,
    pub handshake_timeout: Duration,
    pub read_timeout: Duration,
    pub write_timeout: Duration,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            url: String::new(),
            token: None,
            user: None,
            handshake_timeout: Duration::from_secs(10),
            read_timeout: Duration::from_secs(30),
            write_timeout: Duration::from_secs(10),
        }
    }
}

impl Config {
    pub fn new(url: impl Into<String>, token: impl Into<Option<String>>) -> Self {
        let mut cfg = Self::default();
        cfg.url = url.into();
        cfg.token = token.into();
        cfg
    }

    pub fn with_user(mut self, user: impl Into<String>) -> Self {
        self.user = Some(user.into());
        self
    }

    pub fn with_timeouts(mut self, handshake: Duration, read: Duration, write: Duration) -> Self {
        self.handshake_timeout = handshake;
        self.read_timeout = read;
        self.write_timeout = write;
        self
    }
}
