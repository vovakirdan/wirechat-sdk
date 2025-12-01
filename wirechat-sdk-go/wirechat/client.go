package wirechat

import (
	"context"
	"errors"
	"io"
	"net/url"
	"sync"

	"github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat/internal"

	"github.com/coder/websocket"
)

// Client provides high-level SDK for WireChat.
type Client struct {
	cfg        Config
	logger     Logger
	conn       *internal.Conn
	rawConn    *websocket.Conn
	writeCh    chan Inbound
	dispatcher Dispatcher

	mu        sync.Mutex
	connected bool
	cancel    context.CancelFunc
}

// NewClient constructs a client with provided config.
// Use DefaultConfig() as a starting point and modify as needed.
// Set timeout to 0 to disable it.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:     cfg,
		logger:  noopLogger{},
		writeCh: make(chan Inbound, 16),
	}
}

// SetLogger overrides logger (optional).
func (c *Client) SetLogger(l Logger) {
	if l == nil {
		return
	}
	c.logger = l
}

// OnMessage registers callback for message events.
func (c *Client) OnMessage(fn func(MessageEvent)) { c.dispatcher.SetOnMessage(fn) }

// OnUserJoined registers callback for user joined events.
func (c *Client) OnUserJoined(fn func(UserEvent)) { c.dispatcher.SetOnUserJoined(fn) }

// OnUserLeft registers callback for user left events.
func (c *Client) OnUserLeft(fn func(UserEvent)) { c.dispatcher.SetOnUserLeft(fn) }

// OnError registers callback for errors.
func (c *Client) OnError(fn func(error)) { c.dispatcher.SetOnError(fn) }

// Connect dials the server, sends hello, and starts internal loops.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return errors.New("already connected")
	}
	c.mu.Unlock()

	if c.cfg.URL == "" {
		return errors.New("empty URL")
	}
	u, err := url.Parse(c.cfg.URL)
	if err != nil {
		return err
	}

	dialCtx := ctx
	if c.cfg.HandshakeTimeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, c.cfg.HandshakeTimeout)
		defer cancel()
	}

	ws, _, err := websocket.Dial(dialCtx, u.String(), nil)
	if err != nil {
		return err
	}

	c.rawConn = ws
	c.conn = internal.NewConn(ws, c.cfg.ReadTimeout, c.cfg.WriteTimeout)

	hello := Inbound{
		Type: inboundHello,
		Data: HelloPayload{
			Protocol: ProtocolVersion,
			Token:    c.cfg.Token,
			User:     c.cfg.User,
		},
	}
	if err := c.conn.Write(ctx, hello); err != nil {
		_ = c.conn.Close(websocket.StatusInternalError, "handshake error")
		return err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()

	go c.readLoop(runCtx)
	go c.writeLoop(runCtx)
	return nil
}

// Join subscribes to a room.
func (c *Client) Join(ctx context.Context, room string) error {
	return c.send(ctx, Inbound{Type: inboundJoin, Data: JoinPayload{Room: room}})
}

// Leave unsubscribes from a room.
func (c *Client) Leave(ctx context.Context, room string) error {
	return c.send(ctx, Inbound{Type: inboundLeave, Data: JoinPayload{Room: room}})
}

// Send publishes a message to a room.
func (c *Client) Send(ctx context.Context, room, text string) error {
	return c.send(ctx, Inbound{Type: inboundMsg, Data: MsgPayload{Room: room, Text: text}})
}

// Close shuts down client and closes WebSocket.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
	}
	c.connected = false
	c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close(websocket.StatusNormalClosure, "client close")
	}
	return nil
}

func (c *Client) send(ctx context.Context, in Inbound) error {
	c.mu.Lock()
	connected := c.connected
	c.mu.Unlock()
	if !connected {
		return errors.New("not connected")
	}

	select {
	case c.writeCh <- in:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) readLoop(ctx context.Context) {
	for {
		var out Outbound
		if err := c.conn.Read(ctx, &out); err != nil {
			if isExpectedDisconnect(ctx, err) {
				return
			}
			c.dispatcher.Dispatch(Outbound{Type: outboundError, Error: &Error{Code: "read_error", Msg: err.Error()}})
			c.logger.Warn("read loop exit", map[string]interface{}{"error": err.Error()})
			return
		}
		c.dispatcher.Dispatch(out)
	}
}

func (c *Client) writeLoop(ctx context.Context) {
	for {
		select {
		case in := <-c.writeCh:
			if err := c.conn.Write(ctx, in); err != nil {
				c.dispatcher.Dispatch(Outbound{Type: outboundError, Error: &Error{Code: "write_error", Msg: err.Error()}})
				c.logger.Warn("write loop exit", map[string]interface{}{"error": err.Error()})
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func isExpectedDisconnect(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if ctx != nil && ctx.Err() != nil {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
		return true
	}
	switch websocket.CloseStatus(err) {
	case websocket.StatusNormalClosure, websocket.StatusGoingAway:
		return true
	default:
		return false
	}
}
