import asyncio
import signal
import sys
from pathlib import Path

EXAMPLES_ROOT = Path(__file__).resolve().parents[2] / "src"
if str(EXAMPLES_ROOT) not in sys.path:
    sys.path.insert(0, str(EXAMPLES_ROOT))

from wirechat import Config, MessageEvent, UserEvent, WirechatClient, WirechatError


async def main() -> None:
    loop = asyncio.get_running_loop()
    stop_event = asyncio.Event()

    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, stop_event.set)
        except NotImplementedError:
            # add_signal_handler is not available on Windows event loop
            pass

    cfg = Config(
        url="ws://localhost:8080/ws",
        user="hello-python",
        token="",  # Leave empty if JWT is not required
    )
    client = WirechatClient(cfg)

    @client.on_message
    async def handle_message(ev: MessageEvent) -> None:
        print(f"[{ev.room}] {ev.user}: {ev.text}")

    @client.on_user_joined
    async def handle_user_joined(ev: UserEvent) -> None:
        print(f">>> {ev.user} joined room {ev.room}")

    @client.on_user_left
    async def handle_user_left(ev: UserEvent) -> None:
        print(f"<<< {ev.user} left room {ev.room}")

    @client.on_error
    async def handle_error(err: WirechatError) -> None:
        print(f"error: {err}")

    await client.connect()

    room = "general"
    await client.join(room)
    await asyncio.sleep(0.5)
    await client.send(room, "Hello from Python SDK!")

    print("Waiting for messages. Press Ctrl+C to stop.")
    try:
        await asyncio.wait_for(stop_event.wait(), timeout=10)
    except asyncio.TimeoutError:
        pass
    finally:
        await client.close()


if __name__ == "__main__":
    asyncio.run(main())
