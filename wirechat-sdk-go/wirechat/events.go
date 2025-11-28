package wirechat

// MessageEvent emitted when someone sends message.
type MessageEvent struct {
	Room string `json:"room"`
	User string `json:"user"`
	Text string `json:"text"`
	TS   int64  `json:"ts"`
}

// UserEvent emitted when user joins/leaves.
type UserEvent struct {
	Room string `json:"room"`
	User string `json:"user"`
}
