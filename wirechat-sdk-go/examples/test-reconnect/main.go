package main

import (
	"context"
	"fmt"
	"time"

	"github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat"
)

func main() {
	fmt.Println("=== Simple Auto-Reconnect Test ===\n")

	cfg := wirechat.DefaultConfig()
	cfg.URL = "ws://localhost:8080/ws"
	cfg.User = "reconnect-test"
	cfg.AutoReconnect = true
	cfg.ReconnectInterval = 2 * time.Second
	cfg.MaxReconnectDelay = 10 * time.Second
	cfg.MaxReconnectTries = 5 // Try 5 times

	client := wirechat.NewClient(&cfg)

	// Track state changes
	client.OnStateChanged(func(ev wirechat.StateEvent) {
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("[%s] STATE: %s -> %s\n", timestamp, ev.OldState, ev.NewState)
		if ev.Error != nil {
			fmt.Printf("           Error: %v\n", ev.Error)
		}
	})

	client.OnError(func(err error) {
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("[%s] ERROR: %v\n", timestamp, err)
	})

	fmt.Println("Connecting to ws://localhost:8080/ws...")
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	fmt.Println("\n✓ Connected!")
	fmt.Println("\nNow kill the server with: pkill -f wirechat-server")
	fmt.Println("Watch the client attempt to reconnect automatically.")
	fmt.Println("Then restart the server and watch it reconnect.\n")
	fmt.Println("Press Ctrl+C to exit\n")

	// Keep running for 2 minutes
	time.Sleep(2 * time.Minute)

	fmt.Println("\nClosing...")
	client.Close()
}
