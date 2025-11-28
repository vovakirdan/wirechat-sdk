package internal

import (
	"context"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Conn wraps websocket.Conn with timeouts.
type Conn struct {
	ws           *websocket.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func NewConn(ws *websocket.Conn, readTimeout, writeTimeout time.Duration) *Conn {
	return &Conn{ws: ws, readTimeout: readTimeout, writeTimeout: writeTimeout}
}

func (c *Conn) Read(ctx context.Context, v any) error {
	if c.readTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.readTimeout)
		defer cancel()
	}
	return wsjson.Read(ctx, c.ws, v)
}

func (c *Conn) Write(ctx context.Context, v any) error {
	if c.writeTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.writeTimeout)
		defer cancel()
	}
	return wsjson.Write(ctx, c.ws, v)
}

func (c *Conn) Close(code websocket.StatusCode, reason string) error {
	return c.ws.Close(code, reason)
}
