package wirechat

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat/internal"
	"github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat/rest"

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

	// REST API client
	REST *rest.Client

	mu               sync.Mutex
	state            ConnectionState
	connected        bool
	cancel           context.CancelFunc
	joinedRooms      map[string]bool // Track joined rooms for auto-reconnect
	reconnectAttempt int             // Current reconnection attempt count
	messageBuffer    []Inbound       // Buffer for outgoing messages during disconnect
}

// NewClient constructs a client with provided config.
// Use DefaultConfig() as a starting point and modify as needed.
// Set timeout to 0 to disable it.
func NewClient(cfg *Config) *Client {
	c := &Client{
		cfg:         *cfg,
		logger:      noopLogger{},
		writeCh:     make(chan Inbound, 16),
		state:       StateDisconnected,
		joinedRooms: make(map[string]bool),
	}

	// Initialize REST client if RESTBaseURL is provided
	if cfg.RESTBaseURL != "" {
		c.REST = rest.NewClient(cfg.RESTBaseURL)
		if cfg.Token != "" {
			c.REST.SetToken(cfg.Token)
		}
	}

	return c
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

// OnHistory registers callback for history events (received after joining a room).
func (c *Client) OnHistory(fn func(HistoryEvent)) { c.dispatcher.SetOnHistory(fn) }

// OnError registers callback for errors.
func (c *Client) OnError(fn func(error)) { c.dispatcher.SetOnError(fn) }

// OnStateChanged registers callback for connection state changes.
func (c *Client) OnStateChanged(fn func(StateEvent)) { c.dispatcher.SetOnStateChanged(fn) }

// State returns the current connection state.
func (c *Client) State() ConnectionState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// setState transitions to a new state and fires the state change callback.
func (c *Client) setState(newState ConnectionState, err error) {
	c.mu.Lock()
	oldState := c.state
	c.state = newState
	c.mu.Unlock()

	// Fire callback outside of lock to avoid deadlocks
	c.dispatcher.fireStateChange(oldState, newState, err)
}

// Connect dials the server, sends hello, and starts internal loops.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return NewError(ErrorInvalidConfig, "already connected")
	}
	c.mu.Unlock()

	c.setState(StateConnecting, nil)

	if c.cfg.URL == "" {
		err := NewError(ErrorInvalidConfig, "empty URL")
		c.setState(StateError, err)
		return err
	}
	u, err := url.Parse(c.cfg.URL)
	if err != nil {
		wrappedErr := WrapError(ErrorInvalidConfig, "invalid WebSocket URL", err)
		c.setState(StateError, wrappedErr)
		return wrappedErr
	}

	dialCtx := ctx
	if c.cfg.HandshakeTimeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, c.cfg.HandshakeTimeout)
		defer cancel()
	}

	ws, _, err := websocket.Dial(dialCtx, u.String(), nil)
	if err != nil {
		wrappedErr := WrapError(ErrorConnection, "failed to dial WebSocket", err)
		c.setState(StateError, wrappedErr)
		return wrappedErr
	}

	c.rawConn = ws
	c.conn = internal.NewConn(ws, c.cfg.ReadTimeout, c.cfg.WriteTimeout)

	// Use protocol from config, fallback to constant if not set
	protocol := c.cfg.Protocol
	if protocol == 0 {
		protocol = ProtocolVersion
	}

	hello := Inbound{
		Type: inboundHello,
		Data: HelloPayload{
			Protocol: protocol,
			Token:    c.cfg.Token,
			User:     c.cfg.User,
		},
	}
	if err := c.conn.Write(ctx, hello); err != nil {
		_ = c.conn.Close(websocket.StatusInternalError, "handshake error")
		wrappedErr := WrapError(ErrorConnection, "failed to send hello handshake", err)
		c.setState(StateError, wrappedErr)
		return wrappedErr
	}

	runCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.mu.Lock()
	c.connected = true
	c.reconnectAttempt = 0 // Reset reconnect counter on successful connect
	c.mu.Unlock()

	c.setState(StateConnected, nil)

	go c.readLoop(runCtx)
	go c.writeLoop(runCtx)
	return nil
}

// Join subscribes to a room.
func (c *Client) Join(ctx context.Context, room string) error {
	if err := c.send(ctx, Inbound{Type: inboundJoin, Data: JoinPayload{Room: room}}); err != nil {
		return err
	}

	// Track joined rooms for auto-reconnect
	c.mu.Lock()
	c.joinedRooms[room] = true
	c.mu.Unlock()

	return nil
}

// Leave unsubscribes from a room.
func (c *Client) Leave(ctx context.Context, room string) error {
	if err := c.send(ctx, Inbound{Type: inboundLeave, Data: JoinPayload{Room: room}}); err != nil {
		return err
	}

	// Remove from joined rooms tracking
	c.mu.Lock()
	delete(c.joinedRooms, room)
	c.mu.Unlock()

	return nil
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

	c.setState(StateClosed, nil)

	if c.conn != nil {
		return c.conn.Close(websocket.StatusNormalClosure, "client close")
	}
	return nil
}

func (c *Client) send(ctx context.Context, in Inbound) error {
	c.mu.Lock()
	connected := c.connected

	// If not connected and buffering is enabled, buffer the message
	if !connected && c.cfg.BufferMessages {
		// Check buffer size limit
		if len(c.messageBuffer) >= c.cfg.MaxBufferSize {
			c.mu.Unlock()
			return NewError(ErrorNotConnected, "message buffer full")
		}
		// Add to buffer
		c.messageBuffer = append(c.messageBuffer, in)
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	if !connected {
		return NewError(ErrorNotConnected, "client not connected")
	}

	select {
	case c.writeCh <- in:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// reconnect attempts to reconnect with exponential backoff.
func (c *Client) reconnect(ctx context.Context) error {
	if !c.cfg.AutoReconnect {
		return NewError(ErrorDisconnected, "auto-reconnect disabled")
	}

	c.mu.Lock()
	c.reconnectAttempt++
	attempt := c.reconnectAttempt
	c.mu.Unlock()

	// Check max tries
	if c.cfg.MaxReconnectTries > 0 && attempt > c.cfg.MaxReconnectTries {
		err := NewError(ErrorDisconnected, "max reconnect attempts exceeded")
		c.setState(StateError, err)
		return err
	}

	// Calculate delay with exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (max)
	var delay time.Duration
	switch {
	case attempt <= 0:
		// Should never happen, but handle gracefully
		delay = c.cfg.ReconnectInterval
	case attempt <= 30:
		// Safe: attempt > 0, so attempt-1 >= 0
		delay = c.cfg.ReconnectInterval * (1 << (attempt - 1))
	default:
		// Cap at 2^30 to prevent overflow
		delay = c.cfg.ReconnectInterval * (1 << 30)
	}
	if delay > c.cfg.MaxReconnectDelay {
		delay = c.cfg.MaxReconnectDelay
	}

	c.logger.Warn("reconnecting after delay", map[string]interface{}{
		"attempt": attempt,
		"delay":   delay.String(),
	})

	// Wait before reconnect
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return ctx.Err()
	}

	// Attempt reconnection
	c.setState(StateReconnecting, nil)

	// Parse URL
	u, err := url.Parse(c.cfg.URL)
	if err != nil {
		return WrapError(ErrorInvalidConfig, "invalid WebSocket URL", err)
	}

	// Dial with handshake timeout
	dialCtx := ctx
	if c.cfg.HandshakeTimeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, c.cfg.HandshakeTimeout)
		defer cancel()
	}

	ws, _, err := websocket.Dial(dialCtx, u.String(), nil)
	if err != nil {
		return WrapError(ErrorConnection, "failed to dial WebSocket", err)
	}

	c.rawConn = ws
	c.conn = internal.NewConn(ws, c.cfg.ReadTimeout, c.cfg.WriteTimeout)

	// Send hello
	protocol := c.cfg.Protocol
	if protocol == 0 {
		protocol = ProtocolVersion
	}

	hello := Inbound{
		Type: inboundHello,
		Data: HelloPayload{
			Protocol: protocol,
			Token:    c.cfg.Token,
			User:     c.cfg.User,
		},
	}
	if err := c.conn.Write(ctx, hello); err != nil {
		_ = c.conn.Close(websocket.StatusInternalError, "handshake error")
		return WrapError(ErrorConnection, "failed to send hello handshake", err)
	}

	// Reconnection successful
	c.mu.Lock()
	c.connected = true
	c.reconnectAttempt = 0 // Reset counter on success
	c.mu.Unlock()

	c.setState(StateConnected, nil)

	// Re-join all rooms
	if err := c.rejoinRooms(ctx); err != nil {
		c.logger.Warn("failed to rejoin some rooms", map[string]interface{}{"error": err.Error()})
	}

	// Flush buffered messages
	if c.cfg.BufferMessages {
		if err := c.flushBuffer(ctx); err != nil {
			c.logger.Warn("failed to flush message buffer", map[string]interface{}{"error": err.Error()})
		}
	}

	return nil
}

// rejoinRooms re-joins all previously joined rooms after reconnection.
func (c *Client) rejoinRooms(ctx context.Context) error {
	c.mu.Lock()
	rooms := make([]string, 0, len(c.joinedRooms))
	for room := range c.joinedRooms {
		rooms = append(rooms, room)
	}
	c.mu.Unlock()

	for _, room := range rooms {
		// Send join without updating joinedRooms (already tracked)
		if err := c.send(ctx, Inbound{Type: inboundJoin, Data: JoinPayload{Room: room}}); err != nil {
			return WrapError(ErrorConnection, "failed to rejoin room: "+room, err)
		}
	}

	return nil
}

// flushBuffer sends all buffered messages after reconnection.
func (c *Client) flushBuffer(ctx context.Context) error {
	c.mu.Lock()
	buffered := make([]Inbound, len(c.messageBuffer))
	copy(buffered, c.messageBuffer)
	c.messageBuffer = c.messageBuffer[:0] // Clear buffer
	c.mu.Unlock()

	for _, msg := range buffered {
		// Send directly to writeCh (client is already connected)
		select {
		case c.writeCh <- msg:
			// Message sent successfully
		case <-ctx.Done():
			return ctx.Err()
		default:
			// writeCh full, this shouldn't happen but handle gracefully
			c.logger.Warn("write channel full during buffer flush", nil)
			return NewError(ErrorConnection, "failed to flush buffer: write channel full")
		}
	}

	return nil
}

func (c *Client) readLoop(ctx context.Context) {
	for {
		var out Outbound
		if err := c.conn.Read(ctx, &out); err != nil {
			// Check if this is user-initiated close (context cancelled)
			if ctx.Err() != nil {
				c.setState(StateDisconnected, nil)
				return
			}

			// Check if this is a normal WebSocket close from user's Close() call
			closeStatus := websocket.CloseStatus(err)
			if closeStatus == websocket.StatusNormalClosure {
				c.setState(StateDisconnected, nil)
				return
			}

			// Connection lost unexpectedly
			wireErr := WrapError(ErrorConnection, "read error", err)
			c.dispatcher.fireError(wireErr)
			c.logger.Warn("read loop: connection lost", map[string]interface{}{"error": err.Error()})

			// Mark as disconnected
			c.mu.Lock()
			c.connected = false
			c.mu.Unlock()
			c.setState(StateDisconnected, wireErr)

			// Attempt reconnection if enabled
			if !c.cfg.AutoReconnect {
				c.setState(StateError, wireErr)
				return
			}

			// Reconnection loop with exponential backoff
			for {
				if err := c.reconnect(ctx); err != nil {
					if ctx.Err() != nil {
						// Context cancelled, exit
						return
					}
					// Reconnection failed, will retry with backoff
					c.dispatcher.fireError(err)
					continue
				}

				// Reconnection successful, restart write loop
				go c.writeLoop(ctx)

				// Continue reading from new connection
				c.logger.Warn("read loop: reconnected successfully", nil)
				break
			}
		} else {
			c.dispatcher.Dispatch(out)
		}
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
