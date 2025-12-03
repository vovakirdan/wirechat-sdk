package wirechat

import "time"

// Config controls how the SDK connects.
type Config struct {
	// WebSocket configuration
	URL              string
	Protocol         int           // Protocol version (default: 1)
	Token            string        // JWT for hello
	User             string        // Username (used when JWT is not required)
	HandshakeTimeout time.Duration // 0 = no timeout, positive = custom timeout
	ReadTimeout      time.Duration // 0 = no timeout, positive = custom timeout
	WriteTimeout     time.Duration // 0 = no timeout, positive = custom timeout

	// REST API configuration
	RESTBaseURL string // REST API base URL (e.g., "http://localhost:8080/api")

	// Auto-reconnect configuration
	AutoReconnect     bool          // Enable automatic reconnection on disconnect
	ReconnectInterval time.Duration // Initial reconnect delay (default: 1s)
	MaxReconnectDelay time.Duration // Maximum reconnect delay (default: 30s)
	MaxReconnectTries int           // Maximum reconnect attempts (0 = infinite, default: 0)

	// Message buffering configuration
	BufferMessages bool // Enable buffering of outgoing messages during disconnect
	MaxBufferSize  int  // Maximum number of messages to buffer (default: 100)
}

// DefaultConfig returns sensible defaults.
// Protocol is set to 1 (current protocol version).
// ReadTimeout is set to 0 (infinite) because chat messages arrive sporadically.
// The server handles connection keepalive via ping/pong frames.
// HandshakeTimeout and WriteTimeout are set to reasonable values to detect network issues during active operations.
// AutoReconnect is disabled by default - clients must opt-in.
// BufferMessages is disabled by default - clients must opt-in.
func DefaultConfig() Config {
	return Config{
		Protocol:          1,
		HandshakeTimeout:  10 * time.Second,
		ReadTimeout:       0, // 0 = infinite, wait for server ping/pong
		WriteTimeout:      10 * time.Second,
		AutoReconnect:     false, // Disabled by default
		ReconnectInterval: 1 * time.Second,
		MaxReconnectDelay: 30 * time.Second,
		MaxReconnectTries: 0,     // 0 = infinite retries
		BufferMessages:    false, // Disabled by default
		MaxBufferSize:     100,
	}
}
