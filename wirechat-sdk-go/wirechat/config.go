package wirechat

import "time"

// Config controls how the SDK connects.
type Config struct {
	URL              string
	Token            string // JWT for hello
	User             string // Username (used when JWT is not required)
	HandshakeTimeout time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		HandshakeTimeout: 10 * time.Second,
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     10 * time.Second,
	}
}
