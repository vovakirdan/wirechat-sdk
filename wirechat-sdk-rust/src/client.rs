use std::sync::Arc;
use std::time::Duration;

use futures_util::SinkExt;
use futures_util::stream::{SplitSink, SplitStream, StreamExt};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use tokio::sync::{broadcast, mpsc, watch};
use tokio::task::JoinHandle;
use tokio::time;
use tokio_tungstenite::tungstenite::Message;
use tokio_tungstenite::{MaybeTlsStream, WebSocketStream, connect_async};

use crate::PROTOCOL_VERSION;
use crate::config::Config;
use crate::error::WirechatError;
use crate::event::{Event, MessageEvent, UserEvent};

type Handler<T> = Arc<dyn Fn(T) + Send + Sync + 'static>;
type WsStream = WebSocketStream<MaybeTlsStream<tokio::net::TcpStream>>;

pub struct Client {
    cfg: Config,
    connected: bool,
    write_tx: Option<mpsc::Sender<Inbound>>,
    shutdown_tx: Option<watch::Sender<bool>>,
    read_handle: Option<JoinHandle<()>>,
    write_handle: Option<JoinHandle<()>>,
    events_tx: broadcast::Sender<Event>,
    on_message: Option<Handler<MessageEvent>>,
    on_user_joined: Option<Handler<UserEvent>>,
    on_user_left: Option<Handler<UserEvent>>,
    on_error: Option<Handler<WirechatError>>,
}

impl Client {
    pub fn new(cfg: Config) -> Self {
        let (events_tx, _) = broadcast::channel(64);
        Self {
            cfg,
            connected: false,
            write_tx: None,
            shutdown_tx: None,
            read_handle: None,
            write_handle: None,
            events_tx,
            on_message: None,
            on_user_joined: None,
            on_user_left: None,
            on_error: None,
        }
    }

    pub fn on_message<F>(&mut self, f: F)
    where
        F: Fn(MessageEvent) + Send + Sync + 'static,
    {
        self.on_message = Some(Arc::new(f));
    }

    pub fn on_user_joined<F>(&mut self, f: F)
    where
        F: Fn(UserEvent) + Send + Sync + 'static,
    {
        self.on_user_joined = Some(Arc::new(f));
    }

    pub fn on_user_left<F>(&mut self, f: F)
    where
        F: Fn(UserEvent) + Send + Sync + 'static,
    {
        self.on_user_left = Some(Arc::new(f));
    }

    pub fn on_error<F>(&mut self, f: F)
    where
        F: Fn(WirechatError) + Send + Sync + 'static,
    {
        self.on_error = Some(Arc::new(f));
    }

    pub fn events(&self) -> broadcast::Receiver<Event> {
        self.events_tx.subscribe()
    }

    pub async fn connect(&mut self) -> Result<(), WirechatError> {
        if self.connected {
            return Err(WirechatError::AlreadyConnected);
        }
        if self.cfg.url.is_empty() {
            return Err(WirechatError::UrlEmpty);
        }

        let connect_fut = connect_async(self.cfg.url.clone());
        let (ws_stream, _) = if self.cfg.handshake_timeout > Duration::ZERO {
            time::timeout(self.cfg.handshake_timeout, connect_fut)
                .await
                .map_err(WirechatError::from)?
                .map_err(WirechatError::from)?
        } else {
            connect_fut.await.map_err(WirechatError::from)?
        };

        let (mut sink, stream) = ws_stream.split();

        let hello = Inbound::hello(&self.cfg)?;
        send_inbound(&mut sink, hello, self.cfg.write_timeout).await?;

        let (write_tx, write_rx) = mpsc::channel(32);
        let (shutdown_tx, shutdown_rx) = watch::channel(false);

        let events_tx = self.events_tx.clone();
        let read_events_tx = self.events_tx.clone();

        let on_error_write = self.on_error.clone();
        let on_error_read = self.on_error.clone();
        let on_message = self.on_message.clone();
        let on_user_joined = self.on_user_joined.clone();
        let on_user_left = self.on_user_left.clone();

        let write_timeout = self.cfg.write_timeout;
        let mut write_shutdown = shutdown_rx.clone();
        self.write_handle = Some(tokio::spawn(async move {
            write_loop(
                sink,
                write_rx,
                write_timeout,
                &events_tx,
                on_error_write,
                &mut write_shutdown,
            )
            .await;
        }));

        let read_timeout = self.cfg.read_timeout;
        let mut read_shutdown = shutdown_rx.clone();
        self.read_handle = Some(tokio::spawn(async move {
            read_loop(
                stream,
                read_timeout,
                &read_events_tx,
                on_message,
                on_user_joined,
                on_user_left,
                on_error_read,
                &mut read_shutdown,
            )
            .await;
        }));

        self.write_tx = Some(write_tx);
        self.shutdown_tx = Some(shutdown_tx);
        self.connected = true;
        Ok(())
    }

    pub async fn join(&self, room: impl Into<String>) -> Result<(), WirechatError> {
        let inbound = Inbound::join(room.into())?;
        self.queue(inbound).await
    }

    pub async fn leave(&self, room: impl Into<String>) -> Result<(), WirechatError> {
        let inbound = Inbound::leave(room.into())?;
        self.queue(inbound).await
    }

    pub async fn send(
        &self,
        room: impl Into<String>,
        text: impl Into<String>,
    ) -> Result<(), WirechatError> {
        let inbound = Inbound::msg(room.into(), text.into())?;
        self.queue(inbound).await
    }

    pub async fn close(&mut self) -> Result<(), WirechatError> {
        if !self.connected {
            return Ok(());
        }
        self.connected = false;

        if let Some(shutdown) = self.shutdown_tx.take() {
            let _ = shutdown.send(true);
        }

        self.write_tx = None;

        if let Some(handle) = self.read_handle.take() {
            let _ = handle.await?;
        }
        if let Some(handle) = self.write_handle.take() {
            let _ = handle.await?;
        }
        Ok(())
    }

    async fn queue(&self, inbound: Inbound) -> Result<(), WirechatError> {
        let Some(tx) = &self.write_tx else {
            return Err(WirechatError::NotConnected);
        };
        tx.send(inbound)
            .await
            .map_err(|_| WirechatError::NotConnected)
    }
}

#[derive(Debug, Serialize, Clone)]
struct Inbound {
    #[serde(rename = "type")]
    r#type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    data: Option<Value>,
}

impl Inbound {
    fn hello(cfg: &Config) -> Result<Self, WirechatError> {
        let payload = HelloPayload {
            protocol: PROTOCOL_VERSION,
            token: cfg.token.clone(),
            user: cfg.user.clone(),
        };
        Ok(Self {
            r#type: "hello".to_string(),
            data: Some(serde_json::to_value(payload)?),
        })
    }

    fn join(room: String) -> Result<Self, WirechatError> {
        Ok(Self {
            r#type: "join".to_string(),
            data: Some(serde_json::to_value(JoinPayload { room })?),
        })
    }

    fn leave(room: String) -> Result<Self, WirechatError> {
        Ok(Self {
            r#type: "leave".to_string(),
            data: Some(serde_json::to_value(JoinPayload { room })?),
        })
    }

    fn msg(room: String, text: String) -> Result<Self, WirechatError> {
        Ok(Self {
            r#type: "msg".to_string(),
            data: Some(serde_json::to_value(MsgPayload { room, text })?),
        })
    }
}

#[derive(Debug, Serialize, Deserialize)]
struct HelloPayload {
    protocol: u8,
    token: Option<String>,
    user: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
struct JoinPayload {
    room: String,
}

#[derive(Debug, Serialize, Deserialize)]
struct MsgPayload {
    room: String,
    text: String,
}

#[derive(Debug, Deserialize)]
struct Outbound {
    #[serde(rename = "type")]
    r#type: String,
    #[serde(default)]
    event: Option<String>,
    #[serde(default)]
    data: Option<Value>,
    #[serde(default)]
    error: Option<ErrorPayload>,
}

#[derive(Debug, Deserialize)]
struct ErrorPayload {
    code: String,
    msg: String,
}

async fn write_loop(
    mut sink: SplitSink<WsStream, Message>,
    mut rx: mpsc::Receiver<Inbound>,
    write_timeout: Duration,
    events_tx: &broadcast::Sender<Event>,
    on_error: Option<Handler<WirechatError>>,
    shutdown: &mut watch::Receiver<bool>,
) {
    let mut should_close = false;
    while !*shutdown.borrow() {
        tokio::select! {
            _ = shutdown.changed() => { should_close = true; break; }
            inbound = rx.recv() => {
                let Some(inbound) = inbound else { should_close = true; break; };
                if let Err(err) = send_inbound(&mut sink, inbound, write_timeout).await {
                    emit_error(err, &on_error, events_tx);
                    should_close = true;
                    break;
                }
            }
        }
    }

    if should_close {
        // try to send close frame; ignore errors
        let _ = time::timeout(write_timeout, sink.send(Message::Close(None))).await;
        let _ = sink.close().await;
    }
}

async fn read_loop(
    mut stream: SplitStream<WsStream>,
    read_timeout: Duration,
    events_tx: &broadcast::Sender<Event>,
    on_message: Option<Handler<MessageEvent>>,
    on_user_joined: Option<Handler<UserEvent>>,
    on_user_left: Option<Handler<UserEvent>>,
    on_error: Option<Handler<WirechatError>>,
    shutdown: &mut watch::Receiver<bool>,
) {
    while !*shutdown.borrow() {
        let msg = if read_timeout > Duration::ZERO {
            tokio::select! {
                _ = shutdown.changed() => break,
                res = time::timeout(read_timeout, stream.next()) => {
                    match res {
                        Ok(m) => m,
                        Err(_) => continue, // idle timeout; keep waiting
                    }
                }
            }
        } else {
            tokio::select! {
                _ = shutdown.changed() => break,
                m = stream.next() => m,
            }
        };

        let Some(msg) = msg else {
            break;
        };
        match msg {
            Ok(Message::Text(text)) => {
                if let Err(err) = handle_outbound(
                    &text,
                    events_tx,
                    &on_message,
                    &on_user_joined,
                    &on_user_left,
                    &on_error,
                ) {
                    emit_error(err, &on_error, events_tx);
                    break;
                }
            }
            Ok(Message::Close(_)) => break,
            Ok(Message::Ping(_)) | Ok(Message::Pong(_)) => {}
            Ok(Message::Binary(_)) => {}
            Ok(Message::Frame(_)) => {}
            Err(err) => {
                emit_error(WirechatError::from(err), &on_error, events_tx);
                break;
            }
        }
    }
}

fn handle_outbound(
    text: &str,
    events_tx: &broadcast::Sender<Event>,
    on_message: &Option<Handler<MessageEvent>>,
    on_user_joined: &Option<Handler<UserEvent>>,
    on_user_left: &Option<Handler<UserEvent>>,
    on_error: &Option<Handler<WirechatError>>,
) -> Result<(), WirechatError> {
    let outbound: Outbound = serde_json::from_str(text)?;

    match outbound.r#type.as_str() {
        "event" => handle_event(
            outbound,
            events_tx,
            on_message,
            on_user_joined,
            on_user_left,
        ),
        "error" => {
            if let Some(err) = outbound.error {
                let server_err = WirechatError::Server {
                    code: err.code,
                    message: err.msg,
                };
                emit_error(server_err, on_error, events_tx);
            } else {
                emit_error(
                    WirechatError::Protocol("missing error payload".into()),
                    on_error,
                    events_tx,
                );
            }
            Ok(())
        }
        other => {
            let err = WirechatError::Protocol(format!("unexpected outbound type: {other}"));
            emit_error(err, on_error, events_tx);
            Ok(())
        }
    }
}

fn handle_event(
    outbound: Outbound,
    events_tx: &broadcast::Sender<Event>,
    on_message: &Option<Handler<MessageEvent>>,
    on_user_joined: &Option<Handler<UserEvent>>,
    on_user_left: &Option<Handler<UserEvent>>,
) -> Result<(), WirechatError> {
    let event_name = outbound
        .event
        .ok_or_else(|| WirechatError::Protocol("missing event name".into()))?;
    let data = outbound
        .data
        .ok_or_else(|| WirechatError::Protocol("missing event payload".into()))?;

    match event_name.as_str() {
        "message" => {
            let ev: MessageEvent = serde_json::from_value(data)?;
            emit_message(ev, events_tx, on_message);
        }
        "user_joined" => {
            let ev: UserEvent = serde_json::from_value(data)?;
            emit_user_joined(ev, events_tx, on_user_joined);
        }
        "user_left" => {
            let ev: UserEvent = serde_json::from_value(data)?;
            emit_user_left(ev, events_tx, on_user_left);
        }
        other => {
            return Err(WirechatError::Protocol(format!("unknown event: {other}")));
        }
    }
    Ok(())
}

fn emit_message(
    ev: MessageEvent,
    events_tx: &broadcast::Sender<Event>,
    on_message: &Option<Handler<MessageEvent>>,
) {
    let _ = events_tx.send(Event::Message(ev.clone()));
    if let Some(cb) = on_message {
        cb(ev);
    }
}

fn emit_user_joined(
    ev: UserEvent,
    events_tx: &broadcast::Sender<Event>,
    on_user_joined: &Option<Handler<UserEvent>>,
) {
    let _ = events_tx.send(Event::UserJoined(ev.clone()));
    if let Some(cb) = on_user_joined {
        cb(ev);
    }
}

fn emit_user_left(
    ev: UserEvent,
    events_tx: &broadcast::Sender<Event>,
    on_user_left: &Option<Handler<UserEvent>>,
) {
    let _ = events_tx.send(Event::UserLeft(ev.clone()));
    if let Some(cb) = on_user_left {
        cb(ev);
    }
}

async fn send_inbound(
    sink: &mut SplitSink<WsStream, Message>,
    inbound: Inbound,
    write_timeout: Duration,
) -> Result<(), WirechatError> {
    let payload = serde_json::to_string(&inbound)?;
    let send_fut = sink.send(Message::Text(payload.into()));
    if write_timeout > Duration::ZERO {
        time::timeout(write_timeout, send_fut)
            .await
            .map_err(WirechatError::from)?
            .map_err(WirechatError::from)?;
    } else {
        send_fut.await.map_err(WirechatError::from)?;
    }
    Ok(())
}

fn emit_error(
    err: WirechatError,
    on_error: &Option<Handler<WirechatError>>,
    events_tx: &broadcast::Sender<Event>,
) {
    let _ = events_tx.send(Event::Error(err.clone()));
    if let Some(cb) = on_error {
        cb(err);
    }
}
