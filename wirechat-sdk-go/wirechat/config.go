package wirechat

import "time"

// Config controls how the SDK connects.
type Config struct {
	URL              string
	Protocol         int           // Protocol version (default: 1)
	Token            string        // JWT for hello
	User             string        // Username (used when JWT is not required)
	HandshakeTimeout time.Duration // 0 = no timeout, positive = custom timeout
	ReadTimeout      time.Duration // 0 = no timeout, positive = custom timeout
	WriteTimeout     time.Duration // 0 = no timeout, positive = custom timeout
}

// DefaultConfig returns sensible defaults.
// Protocol is set to 1 (current protocol version).
// ReadTimeout is set to 0 (infinite) because chat messages arrive sporadically.
// The server handles connection keepalive via ping/pong frames.
// HandshakeTimeout and WriteTimeout are set to reasonable values to detect network issues during active operations.
func DefaultConfig() Config {
	return Config{
		Protocol:         1,
		HandshakeTimeout: 10 * time.Second,
		ReadTimeout:      0, // 0 = infinite, wait for server ping/pong
		WriteTimeout:     10 * time.Second,
	}
}
