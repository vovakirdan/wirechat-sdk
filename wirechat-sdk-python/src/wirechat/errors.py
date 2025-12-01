from __future__ import annotations

from dataclasses import dataclass

from .types import Error


class WirechatError(Exception):
    """Base exception for SDK errors."""


class WirechatConnectionError(WirechatError):
    """Raised when the WebSocket connection fails."""


class WirechatProtocolError(WirechatError):
    """Raised when an unexpected payload is received."""


class NotConnectedError(WirechatError):
    """Raised when an operation requires an active connection."""


@dataclass
class WirechatServerError(WirechatError):
    error: Error

    def __str__(self) -> str:
        return str(self.error)


def wrap_error(err: BaseException) -> WirechatError:
    if isinstance(err, WirechatError):
        return err
    return WirechatError(str(err))


__all__ = [
    "WirechatError",
    "WirechatConnectionError",
    "WirechatProtocolError",
    "WirechatServerError",
    "NotConnectedError",
    "wrap_error",
]
