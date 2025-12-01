use tokio::signal;
use tokio::sync::broadcast;
use tokio::time::{self, Duration};
use wirechat_rs::{Client, Config, Event};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let cfg = Config::new("ws://localhost:8080/ws", None).with_user("rust-example");
    let mut client = Client::new(cfg);

    client.on_message(|ev| {
        println!("[{}] {}: {}", ev.room, ev.user, ev.text);
    });

    client.on_error(|err| eprintln!("error: {err}"));

    let mut events = client.events();
    let events_task = tokio::spawn(async move {
        loop {
            match events.recv().await {
                Ok(Event::Message(ev)) => {
                    println!("[stream] [{}] {}: {}", ev.room, ev.user, ev.text)
                }
                Ok(Event::UserJoined(ev)) => {
                    println!("[stream] >>> {} joined {}", ev.user, ev.room)
                }
                Ok(Event::UserLeft(ev)) => println!("[stream] <<< {} left {}", ev.user, ev.room),
                Ok(Event::Error(err)) => eprintln!("[stream] error: {err}"),
                Err(broadcast::error::RecvError::Closed) => break,
                Err(broadcast::error::RecvError::Lagged(_)) => continue,
            }
        }
    });

    client.connect().await?;
    client.join("general").await?;
    client.send("general", "hello from Rust SDK!").await?;

    println!("Press Ctrl+C to exit or wait 10s for auto-shutdown.");
    tokio::select! {
        _ = signal::ctrl_c() => println!("Ctrl+C received, shutting down..."),
        _ = time::sleep(Duration::from_secs(10)) => println!("Timeout reached, shutting down..."),
    }
    client.close().await?;
    events_task.abort();
    Ok(())
}
