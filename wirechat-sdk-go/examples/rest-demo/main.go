package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat"
	"github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat/rest"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	// Configure client with both WebSocket and REST API
	cfg := wirechat.DefaultConfig()
	cfg.URL = "ws://localhost:8080/ws"
	cfg.RESTBaseURL = "http://localhost:8080/api"

	fmt.Println("=== WireChat REST API Demo ===")
	fmt.Println()

	// Step 1: Register a new user
	fmt.Println("1. Registering new user...")
	client := wirechat.NewClient(&cfg)

	username := fmt.Sprintf("demo-user-%d", time.Now().Unix())
	password := "demo123"

	tokenResp, err := client.REST.Register(ctx, rest.RegisterRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}

	fmt.Printf("   ✓ Registered as '%s'\n", username)
	fmt.Printf("   Token: %s...\n\n", tokenResp.Token[:20])

	// Update token for subsequent requests
	client.REST.SetToken(tokenResp.Token)
	cfg.Token = tokenResp.Token

	// Step 2: Create a public room
	fmt.Println("2. Creating public room...")
	roomName := fmt.Sprintf("demo-room-%d", time.Now().Unix())
	room, err := client.REST.CreateRoom(ctx, rest.CreateRoomRequest{
		Name: roomName,
		Type: rest.RoomTypePublic,
	})
	if err != nil {
		return fmt.Errorf("create room: %w", err)
	}

	fmt.Printf("   ✓ Created room '%s' (ID: %d)\n\n", room.Name, room.ID)

	// Step 3: List all accessible rooms
	fmt.Println("3. Listing all rooms...")
	rooms, err := client.REST.ListRooms(ctx)
	if err != nil {
		return fmt.Errorf("list rooms: %w", err)
	}

	fmt.Printf("   Found %d room(s):\n", len(rooms))
	for _, r := range rooms {
		fmt.Printf("   - %s (ID: %d, Type: %s)\n", r.Name, r.ID, r.Type)
	}
	fmt.Println()

	// Step 4: Connect via WebSocket and send messages
	fmt.Println("4. Connecting via WebSocket...")
	wsClient := wirechat.NewClient(&cfg)

	wsClient.OnMessage(func(ev wirechat.MessageEvent) {
		fmt.Printf("   [WS] Message ID:%d from %s: %s\n", ev.ID, ev.User, ev.Text)
	})

	wsClient.OnHistory(func(ev wirechat.HistoryEvent) {
		fmt.Printf("   [WS] History received: %d messages\n", len(ev.Messages))
	})

	if err := wsClient.Connect(ctx); err != nil {
		return fmt.Errorf("ws connect: %w", err)
	}
	defer wsClient.Close()

	fmt.Println("   ✓ Connected via WebSocket")

	// Join the room we created
	if err := wsClient.Join(ctx, roomName); err != nil {
		return fmt.Errorf("join room: %w", err)
	}

	fmt.Printf("   ✓ Joined room '%s'\n\n", roomName)

	// Send a few messages
	fmt.Println("5. Sending messages...")
	for i := 1; i <= 3; i++ {
		msg := fmt.Sprintf("Test message #%d", i)
		if err := wsClient.Send(ctx, roomName, msg); err != nil {
			return fmt.Errorf("send message: %w", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)
	fmt.Println()

	// Step 5: Fetch message history via REST API
	fmt.Println("6. Fetching message history via REST API...")
	history, err := client.REST.GetMessages(ctx, room.ID, 20, nil)
	if err != nil {
		return fmt.Errorf("get messages: %w", err)
	}

	fmt.Printf("   Found %d message(s):\n", len(history.Messages))
	for _, msg := range history.Messages {
		fmt.Printf("   - [ID:%d] %s: %s\n", msg.ID, msg.User, msg.Body)
	}
	fmt.Println()

	// Step 6: Demonstrate pagination
	if len(history.Messages) > 0 && history.HasMore {
		fmt.Println("7. Testing pagination (fetch older messages)...")
		lastMsgID := history.Messages[len(history.Messages)-1].ID
		olderHistory, err := client.REST.GetMessages(ctx, room.ID, 20, &lastMsgID)
		if err != nil {
			return fmt.Errorf("get older messages: %w", err)
		}

		fmt.Printf("   Found %d older message(s)\n", len(olderHistory.Messages))
		fmt.Printf("   HasMore: %v\n\n", olderHistory.HasMore)
	}

	fmt.Println("=== Demo Complete ===")
	return nil
}
