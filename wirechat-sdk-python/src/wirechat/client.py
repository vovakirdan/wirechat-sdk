from __future__ import annotations

import asyncio
import inspect
import json
import logging
from typing import Any, Awaitable, Callable, Optional

import websockets
from websockets.client import WebSocketClientProtocol
from websockets.exceptions import ConnectionClosed, ConnectionClosedError, ConnectionClosedOK

from .errors import (
    NotConnectedError,
    WirechatConnectionError,
    WirechatError,
    WirechatProtocolError,
    WirechatServerError,
    wrap_error,
)
from .types import (
    CONFIG_DEFAULT_HANDSHAKE_TIMEOUT,
    CONFIG_DEFAULT_READ_TIMEOUT,
    CONFIG_DEFAULT_WRITE_TIMEOUT,
    EVENT_MESSAGE,
    EVENT_USER_JOINED,
    EVENT_USER_LEFT,
    INBOUND_HELLO,
    INBOUND_JOIN,
    INBOUND_LEAVE,
    INBOUND_MSG,
    OUTBOUND_ERROR,
    OUTBOUND_EVENT,
    Config,
    HelloPayload,
    Inbound,
    JoinPayload,
    MessageEvent,
    MsgPayload,
    Outbound,
    UserEvent,
)

Handler = Callable[[Any], Awaitable[None] | None]
ErrorHandler = Callable[[WirechatError], Awaitable[None] | None]


class WirechatClient:
    """Async WireChat client built on top of websockets."""

    def __init__(self, config: Config):
        self._cfg = self._apply_defaults(config)
        self._ws: Optional[WebSocketClientProtocol] = None
        self._write_queue: Optional[asyncio.Queue[Inbound]] = None
        self._read_task: Optional[asyncio.Task[None]] = None
        self._write_task: Optional[asyncio.Task[None]] = None
        self._connected = False
        self._closing = False
        self._lock = asyncio.Lock()
        self._logger = logging.getLogger("wirechat")

        self._on_message: Optional[Handler] = None
        self._on_user_joined: Optional[Handler] = None
        self._on_user_left: Optional[Handler] = None
        self._on_error: Optional[ErrorHandler] = None

    @property
    def connected(self) -> bool:
        return self._connected

    def on_message(self, fn: Handler) -> Handler:
        self._on_message = fn
        return fn

    def on_user_joined(self, fn: Handler) -> Handler:
        self._on_user_joined = fn
        return fn

    def on_user_left(self, fn: Handler) -> Handler:
        self._on_user_left = fn
        return fn

    def on_error(self, fn: ErrorHandler) -> ErrorHandler:
        self._on_error = fn
        return fn

    async def connect(self) -> None:
        async with self._lock:
            if self._connected:
                raise WirechatError("already connected")
            if not self._cfg.url:
                raise WirechatError("URL is required")

            self._write_queue = asyncio.Queue(maxsize=16)
            self._closing = False

            try:
                self._ws = await self._dial()
                await self._write_direct(
                    Inbound(
                        INBOUND_HELLO,
                        HelloPayload(token=self._cfg.token, user=self._cfg.user).to_payload(),
                    )
                )
            except Exception as exc:
                await self._cleanup_connection()
                self._write_queue = None
                raise WirechatConnectionError(f"failed to connect to {self._cfg.url}") from exc

            self._connected = True
            self._read_task = asyncio.create_task(self._read_loop(), name="wirechat-read-loop")
            self._write_task = asyncio.create_task(self._write_loop(), name="wirechat-write-loop")

    async def join(self, room: str) -> None:
        await self._enqueue(Inbound(INBOUND_JOIN, JoinPayload(room=room).to_payload()))

    async def leave(self, room: str) -> None:
        await self._enqueue(Inbound(INBOUND_LEAVE, JoinPayload(room=room).to_payload()))

    async def send(self, room: str, text: str) -> None:
        await self._enqueue(Inbound(INBOUND_MSG, MsgPayload(room=room, text=text).to_payload()))

    async def close(self) -> None:
        async with self._lock:
            if not self._connected and self._ws is None:
                return
            self._closing = True
            self._connected = False

        await self._cancel_task(self._read_task)
        await self._cancel_task(self._write_task)
        await self._cleanup_connection()
        self._write_queue = None
        self._read_task = None
        self._write_task = None
        self._closing = False

    async def __aenter__(self) -> "WirechatClient":
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc, tb) -> None:
        await self.close()

    async def _enqueue(self, inbound: Inbound) -> None:
        if not self._connected or self._write_queue is None:
            raise NotConnectedError("client is not connected")
        try:
            self._write_queue.put_nowait(inbound)
        except asyncio.QueueFull:
            await self._emit_error(WirechatError("write queue is full"))

    async def _dial(self) -> WebSocketClientProtocol:
        kwargs: dict[str, Any] = {}
        if self._cfg.handshake_timeout and self._cfg.handshake_timeout > 0:
            kwargs["open_timeout"] = self._cfg.handshake_timeout
        return await websockets.connect(self._cfg.url, **kwargs)

    async def _read_loop(self) -> None:
        assert self._ws is not None
        try:
            while self._connected:
                try:
                    outbound = await self._read_outbound()
                except asyncio.CancelledError:
                    raise
                except ConnectionClosedOK:
                    break
                except (ConnectionClosed, ConnectionClosedError) as exc:
                    await self._emit_error(WirechatConnectionError(str(exc)))
                    break
                except Exception as exc:
                    await self._emit_error(wrap_error(exc))
                    break
                await self._dispatch(outbound)
        finally:
            if not self._closing:
                await self.close()

    async def _write_loop(self) -> None:
        assert self._ws is not None
        assert self._write_queue is not None
        try:
            while self._connected:
                try:
                    inbound = await self._write_queue.get()
                except asyncio.CancelledError:
                    raise

                try:
                    await self._write_direct(inbound)
                except asyncio.CancelledError:
                    raise
                except Exception as exc:
                    await self._emit_error(wrap_error(exc))
                    break
        finally:
            if not self._closing:
                await self.close()

    async def _read_outbound(self) -> Outbound:
        assert self._ws is not None
        if self._cfg.read_timeout and self._cfg.read_timeout > 0:
            raw = await asyncio.wait_for(self._ws.recv(), timeout=self._cfg.read_timeout)
        else:
            raw = await self._ws.recv()
        if isinstance(raw, bytes):
            raw = raw.decode("utf-8")
        obj = json.loads(raw)
        if not isinstance(obj, dict):
            raise WirechatProtocolError("unexpected payload type")
        return Outbound.from_obj(obj)

    async def _write_direct(self, inbound: Inbound) -> None:
        assert self._ws is not None
        payload = inbound.to_wire()
        data = json.dumps(payload, separators=(",", ":"))
        if self._cfg.write_timeout and self._cfg.write_timeout > 0:
            await asyncio.wait_for(self._ws.send(data), timeout=self._cfg.write_timeout)
        else:
            await self._ws.send(data)

    async def _dispatch(self, outbound: Outbound) -> None:
        if outbound.type == OUTBOUND_ERROR and outbound.error:
            await self._emit_error(WirechatServerError(outbound.error))
            return
        if outbound.type != OUTBOUND_EVENT:
            await self._emit_error(WirechatProtocolError(f"unexpected outbound type: {outbound.type!r}"))
            return

        try:
            if outbound.event == EVENT_MESSAGE and self._on_message:
                await self._invoke_handler(self._on_message, self._decode_payload(outbound.data, MessageEvent))
            elif outbound.event == EVENT_USER_JOINED and self._on_user_joined:
                await self._invoke_handler(self._on_user_joined, self._decode_payload(outbound.data, UserEvent))
            elif outbound.event == EVENT_USER_LEFT and self._on_user_left:
                await self._invoke_handler(self._on_user_left, self._decode_payload(outbound.data, UserEvent))
        except WirechatError as exc:
            await self._emit_error(exc)

    async def _invoke_handler(self, handler: Handler, arg: Any) -> None:
        try:
            result = handler(arg)
            if inspect.isawaitable(result):
                await result
        except Exception as exc:
            await self._emit_error(wrap_error(exc))

    def _decode_payload(self, data: Optional[dict[str, Any]], cls: type) -> Any:
        if not isinstance(data, dict):
            raise WirechatProtocolError("missing event payload")
        try:
            return cls(**data)
        except TypeError as exc:
            raise WirechatProtocolError(f"invalid payload for {cls.__name__}") from exc

    async def _emit_error(self, err: BaseException) -> None:
        wire_err = wrap_error(err)
        if self._on_error is None:
            self._logger.debug("wirechat error: %s", wire_err)
            return
        try:
            result = self._on_error(wire_err)
            if inspect.isawaitable(result):
                await result
        except Exception:
            self._logger.error("error handler failed", exc_info=True)

    async def _cleanup_connection(self) -> None:
        if self._ws is not None:
            try:
                await self._ws.close()
            except Exception:
                pass
            self._ws = None

    async def _cancel_task(self, task: Optional[asyncio.Task[Any]]) -> None:
        if task is None or task.done():
            return
        if task is asyncio.current_task():
            return
        task.cancel()
        try:
            await task
        except asyncio.CancelledError:
            pass

    @staticmethod
    def _apply_defaults(cfg: Config) -> Config:
        # Allow 0 to disable timeouts, only apply defaults for negative values
        handshake = cfg.handshake_timeout
        if handshake is None or handshake < 0:
            handshake = CONFIG_DEFAULT_HANDSHAKE_TIMEOUT

        read = cfg.read_timeout
        if read is None or read < 0:
            read = CONFIG_DEFAULT_READ_TIMEOUT

        write = cfg.write_timeout
        if write is None or write < 0:
            write = CONFIG_DEFAULT_WRITE_TIMEOUT
        return Config(
            url=cfg.url,
            token=cfg.token,
            user=cfg.user,
            handshake_timeout=handshake,
            read_timeout=read,
            write_timeout=write,
        )


__all__ = ["WirechatClient"]
