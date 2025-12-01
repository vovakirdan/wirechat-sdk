use tokio::io::{self, AsyncBufReadExt, BufReader};
use tokio::signal;
use tokio::sync::broadcast;
use wirechat_rs::{Client, Config, Event};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let cfg = Config::new("ws://localhost:8080/ws", None).with_user("rust-join-chat");
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
    let room = "general";
    client.join(room).await?;

    println!("Connected to {room}. Type messages or /quit (Ctrl+C to exit).");

    let stdin = BufReader::new(io::stdin());
    let mut lines = stdin.lines();

    loop {
        tokio::select! {
            _ = signal::ctrl_c() => {
                println!("Ctrl+C received, shutting down...");
                break;
            }
            res = lines.next_line() => {
                match res {
                    Ok(Some(line)) => {
                        let text = line.trim();
                        if text.is_empty() {
                            continue;
                        }
                        if text == "/quit" {
                            println!("Bye!");
                            break;
                        }
                        if let Err(err) = client.send(room, text.to_string()).await {
                            eprintln!("send error: {err}");
                        }
                    }
                    Ok(None) => {
                        println!("stdin closed, exiting...");
                        break;
                    }
                    Err(err) => {
                        eprintln!("stdin read error: {err}");
                        break;
                    }
                }
            }
        }
    }

    client.close().await?;
    events_task.abort();
    Ok(())
}
