package wirechat

import "time"

// Config controls how the SDK connects.
type Config struct {
	URL              string
	Token            string        // JWT for hello
	User             string        // Username (used when JWT is not required)
	HandshakeTimeout time.Duration // 0 = no timeout, positive = custom timeout
	ReadTimeout      time.Duration // 0 = no timeout, positive = custom timeout
	WriteTimeout     time.Duration // 0 = no timeout, positive = custom timeout
}

// DefaultConfig returns sensible defaults.
// Start with this and modify fields as needed. To disable a timeout, set it to 0.
func DefaultConfig() Config {
	return Config{
		HandshakeTimeout: 10 * time.Second,
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     10 * time.Second,
	}
}
