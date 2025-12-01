from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Dict, Optional

ProtocolVersion = 1
CONFIG_DEFAULT_HANDSHAKE_TIMEOUT = 10.0
CONFIG_DEFAULT_READ_TIMEOUT = 30.0
CONFIG_DEFAULT_WRITE_TIMEOUT = 10.0

INBOUND_HELLO = "hello"
INBOUND_JOIN = "join"
INBOUND_LEAVE = "leave"
INBOUND_MSG = "msg"

OUTBOUND_EVENT = "event"
OUTBOUND_ERROR = "error"

EVENT_MESSAGE = "message"
EVENT_USER_JOINED = "user_joined"
EVENT_USER_LEFT = "user_left"


@dataclass
class Config:
    """WireChat client configuration.

    Timeout behavior:
    - Positive value: Use custom timeout in seconds
    - 0: Disable timeout (wait indefinitely)
    - Negative: Use default timeout
    """
    url: str
    token: Optional[str] = None
    user: Optional[str] = None
    handshake_timeout: float = CONFIG_DEFAULT_HANDSHAKE_TIMEOUT
    read_timeout: float = CONFIG_DEFAULT_READ_TIMEOUT
    write_timeout: float = CONFIG_DEFAULT_WRITE_TIMEOUT


@dataclass
class HelloPayload:
    protocol: int = ProtocolVersion
    token: Optional[str] = None
    user: Optional[str] = None

    def to_payload(self) -> Dict[str, Any]:
        return _compact_dict(
            {
                "protocol": self.protocol,
                "token": self.token,
                "user": self.user,
            }
        )


@dataclass
class JoinPayload:
    room: str

    def to_payload(self) -> Dict[str, Any]:
        return {"room": self.room}


@dataclass
class MsgPayload:
    room: str
    text: str

    def to_payload(self) -> Dict[str, Any]:
        return {"room": self.room, "text": self.text}


@dataclass
class Inbound:
    type: str
    data: Optional[Dict[str, Any]] = None

    def to_wire(self) -> Dict[str, Any]:
        payload: Dict[str, Any] = {"type": self.type}
        if self.data:
            payload["data"] = self.data
        return payload


@dataclass
class Error:
    code: str
    msg: str

    def __str__(self) -> str:
        return f"{self.code}: {self.msg}"


@dataclass
class Outbound:
    type: str
    event: Optional[str] = None
    data: Optional[Dict[str, Any]] = None
    error: Optional[Error] = None

    @classmethod
    def from_obj(cls, obj: Dict[str, Any]) -> "Outbound":
        err: Optional[Error] = None
        if "error" in obj and obj["error"] is not None:
            err_obj = obj["error"]
            if isinstance(err_obj, dict):
                code = err_obj.get("code")
                msg = err_obj.get("msg")
                if code is not None and msg is not None:
                    err = Error(code=code, msg=msg)
        return cls(
            type=obj.get("type", ""),
            event=obj.get("event"),
            data=obj.get("data"),
            error=err,
        )


@dataclass
class MessageEvent:
    room: str
    user: str
    text: str
    ts: int


@dataclass
class UserEvent:
    room: str
    user: str


def _compact_dict(payload: Dict[str, Any]) -> Dict[str, Any]:
    return {k: v for k, v in payload.items() if v is not None}


__all__ = [
    "Config",
    "HelloPayload",
    "JoinPayload",
    "MsgPayload",
    "Inbound",
    "Outbound",
    "MessageEvent",
    "UserEvent",
    "Error",
    "ProtocolVersion",
    "CONFIG_DEFAULT_HANDSHAKE_TIMEOUT",
    "CONFIG_DEFAULT_READ_TIMEOUT",
    "CONFIG_DEFAULT_WRITE_TIMEOUT",
    "INBOUND_HELLO",
    "INBOUND_JOIN",
    "INBOUND_LEAVE",
    "INBOUND_MSG",
    "OUTBOUND_EVENT",
    "OUTBOUND_ERROR",
    "EVENT_MESSAGE",
    "EVENT_USER_JOINED",
    "EVENT_USER_LEFT",
]
