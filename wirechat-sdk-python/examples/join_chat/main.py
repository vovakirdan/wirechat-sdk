import asyncio
import signal
import sys
from contextlib import suppress
from pathlib import Path

EXAMPLES_ROOT = Path(__file__).resolve().parents[2] / "src"
if str(EXAMPLES_ROOT) not in sys.path:
    sys.path.insert(0, str(EXAMPLES_ROOT))

from wirechat import Config, MessageEvent, UserEvent, WirechatClient, WirechatError  # noqa: E402


async def main() -> None:
    loop = asyncio.get_running_loop()
    stop_event = asyncio.Event()

    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, stop_event.set)
        except NotImplementedError:
            pass

    cfg = Config(
        url="ws://localhost:8080/ws",
        user="join-and-chat",
        token="",
        read_timeout=0.0,  # Disable read timeout - server handles idle detection with ping/pong
    )
    client = WirechatClient(cfg)

    @client.on_message
    async def handle_message(ev: MessageEvent) -> None:
        print(f"[{ev.room}] {ev.user}: {ev.text}")

    @client.on_user_joined
    async def handle_user_joined(ev: UserEvent) -> None:
        print(f">>> {ev.user} joined {ev.room}")

    @client.on_user_left
    async def handle_user_left(ev: UserEvent) -> None:
        print(f"<<< {ev.user} left {ev.room}")

    @client.on_error
    async def handle_error(err: WirechatError) -> None:
        print(f"error: {err}")

    await client.connect()

    room = "general"
    await client.join(room)

    print("Connected. Type messages to chat, /quit to exit.")

    input_task = asyncio.create_task(input_loop(client, room, stop_event))
    try:
        await stop_event.wait()
    finally:
        input_task.cancel()
        with suppress(asyncio.CancelledError):
            await input_task
        await client.close()


async def input_loop(client: WirechatClient, room: str, stop_event: asyncio.Event) -> None:
    while not stop_event.is_set():
        try:
            line = await asyncio.to_thread(input, "> ")
        except EOFError:
            stop_event.set()
            break
        text = line.strip()
        if not text:
            continue
        if text == "/quit":
            stop_event.set()
            break
        try:
            await client.send(room, text)
        except WirechatError as exc:
            print(f"send error: {exc}")


if __name__ == "__main__":
    asyncio.run(main())
