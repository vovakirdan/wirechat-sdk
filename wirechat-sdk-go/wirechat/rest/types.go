package rest

import "time"

// Authentication types

// RegisterRequest is the request body for user registration.
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest is the request body for user login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TokenResponse contains the JWT token returned after successful authentication.
type TokenResponse struct {
	Token string `json:"token"`
}

// Room types

// RoomType represents the type of a room.
type RoomType string

const (
	RoomTypePublic  RoomType = "public"
	RoomTypePrivate RoomType = "private"
	RoomTypeDirect  RoomType = "direct"
)

// RoomInfo represents room metadata.
type RoomInfo struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      RoomType  `json:"type"`
	OwnerID   *int64    `json:"owner_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateRoomRequest is the request body for creating a room.
type CreateRoomRequest struct {
	Name string   `json:"name"`
	Type RoomType `json:"type,omitempty"` // defaults to "public" if not specified
}

// CreateDirectRoomRequest is the request body for creating a direct message room.
type CreateDirectRoomRequest struct {
	UserID int64 `json:"user_id"`
}

// Message history types

// MessageInfo represents a single message in the history.
type MessageInfo struct {
	ID        int64     `json:"id"`
	RoomID    int64     `json:"room_id"`
	UserID    int64     `json:"user_id"`
	User      string    `json:"user"` // username
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// MessagesResponse contains a page of messages with pagination info.
type MessagesResponse struct {
	Messages []MessageInfo `json:"messages"`
	HasMore  bool          `json:"has_more"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error string `json:"error"`
}
