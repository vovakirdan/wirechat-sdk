package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides REST API access to WireChat server.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new REST API client.
// baseURL should be the base URL of the API, e.g., "http://localhost:8080/api".
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetHTTPClient allows setting a custom HTTP client.
func (c *Client) SetHTTPClient(client *http.Client) {
	if client != nil {
		c.httpClient = client
	}
}

// SetToken sets the JWT token for authenticated requests.
func (c *Client) SetToken(token string) {
	c.token = token
}

// Authentication endpoints

// Register creates a new user account and returns a JWT token.
func (c *Client) Register(ctx context.Context, req RegisterRequest) (*TokenResponse, error) {
	var resp TokenResponse
	if err := c.post(ctx, "/register", req, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Login authenticates with existing credentials and returns a JWT token.
func (c *Client) Login(ctx context.Context, req LoginRequest) (*TokenResponse, error) {
	var resp TokenResponse
	if err := c.post(ctx, "/login", req, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GuestLogin creates a temporary guest user and returns a JWT token.
func (c *Client) GuestLogin(ctx context.Context) (*TokenResponse, error) {
	var resp TokenResponse
	if err := c.post(ctx, "/guest", nil, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Room management endpoints

// CreateRoom creates a new public or private room.
func (c *Client) CreateRoom(ctx context.Context, req CreateRoomRequest) (*RoomInfo, error) {
	var resp RoomInfo
	if err := c.post(ctx, "/rooms", req, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListRooms returns all accessible rooms for the authenticated user.
func (c *Client) ListRooms(ctx context.Context) ([]RoomInfo, error) {
	var resp []RoomInfo
	if err := c.get(ctx, "/rooms", &resp, true); err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateDirectRoom creates or returns an existing direct message room with another user.
// This endpoint is idempotent - calling it multiple times with the same peer returns the same room.
func (c *Client) CreateDirectRoom(ctx context.Context, req CreateDirectRoomRequest) (*RoomInfo, error) {
	var resp RoomInfo
	if err := c.post(ctx, "/rooms/direct", req, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Message history endpoints

// GetMessages retrieves message history for a room with cursor-based pagination.
// limit: maximum number of messages to return (default: 20, max: 100).
// before: if provided, returns messages before this message ID (for pagination).
func (c *Client) GetMessages(ctx context.Context, roomID int64, limit int, before *int64) (*MessagesResponse, error) {
	url := fmt.Sprintf("/rooms/%d/messages?limit=%d", roomID, limit)
	if before != nil {
		url += fmt.Sprintf("&before=%d", *before)
	}

	var resp MessagesResponse
	if err := c.get(ctx, url, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Helper methods

func (c *Client) post(ctx context.Context, path string, body, dest any, requireAuth bool) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if requireAuth && c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.do(req, dest)
}

func (c *Client) get(ctx context.Context, path string, dest any, requireAuth bool) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if requireAuth && c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.do(req, dest)
}

func (c *Client) do(req *http.Request, dest any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("api error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return fmt.Errorf("http error: %s (status %d)", string(body), resp.StatusCode)
	}

	// Unmarshal success response
	if dest != nil {
		if err := json.Unmarshal(body, dest); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}
