package wirechat

import "encoding/json"

const (
	ProtocolVersion = 1

	inboundHello = "hello"
	inboundJoin  = "join"
	inboundLeave = "leave"
	inboundMsg   = "msg"

	outboundEvent = "event"
	outboundError = "error"

	eventMessage    = "message"
	eventUserJoined = "user_joined"
	eventUserLeft   = "user_left"
)

// Inbound represents the envelope from client to server.
type Inbound struct {
	Type string      `json:"type"`
	Data any `json:"data,omitempty"`
}

// Outbound is the envelope server -> client.
type Outbound struct {
	Type  string          `json:"type"`
	Event string          `json:"event,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error *Error          `json:"error,omitempty"`
}

// HelloPayload initiates the session.
type HelloPayload struct {
	Protocol int    `json:"protocol,omitempty"`
	Token    string `json:"token,omitempty"`
	User     string `json:"user,omitempty"`
}

// JoinPayload subscribes to a room.
type JoinPayload struct {
	Room string `json:"room"`
}

// MsgPayload sends a message to a room.
type MsgPayload struct {
	Room string `json:"room"`
	Text string `json:"text"`
}

// Error describes a protocol error.
type Error struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Code + ": " + e.Msg
}

// UnmarshalData decodes RawMessage into target.
func UnmarshalData(data json.RawMessage, v any) error {
	return json.Unmarshal(data, v)
}
