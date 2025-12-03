package wirechat

// ConnectionState represents the current state of the WebSocket connection.
type ConnectionState int

const (
	// StateDisconnected means the client is not connected.
	StateDisconnected ConnectionState = iota

	// StateConnecting means the client is establishing a connection.
	StateConnecting

	// StateConnected means the client is connected and ready.
	StateConnected

	// StateReconnecting means the client is attempting to reconnect after a disconnect.
	StateReconnecting

	// StateError means the client encountered an error.
	StateError

	// StateClosed means the client has been explicitly closed by the user.
	StateClosed
)

// String returns the string representation of a ConnectionState.
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateError:
		return "error"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// StateEvent represents a state change event.
type StateEvent struct {
	OldState ConnectionState
	NewState ConnectionState
	Error    error // Optional error that caused the state change
}
