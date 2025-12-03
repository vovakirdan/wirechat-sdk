package wirechat

// MessageEvent emitted when someone sends message.
type MessageEvent struct {
	ID   int64  `json:"id"` // Message ID from database (0 for guest messages)
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

// HistoryEvent emitted when joining a room with message history.
type HistoryEvent struct {
	Room     string         `json:"room"`
	Messages []MessageEvent `json:"messages"`
}
