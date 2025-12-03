package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat"
)

//go:generate go build -o ../../bin/test-history .

const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo3LCJ1c2VybmFtZSI6InRlc3R1c2VyIiwiaXNfZ3Vlc3QiOmZhbHNlLCJhdWQiOlsiIl0sImV4cCI6MTc2NDg0MDE5NiwiaWF0IjoxNzY0NzUzNzk2fQ.Tw7aDHGHho2vA2IRRDHToZia0EOwcOB94U8BGC_YWlg"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "populate" {
		populateRoom()
	} else {
		testHistory()
	}
}

func populateRoom() {
	fmt.Println("=== Populating room with messages ===")

	cfg := wirechat.DefaultConfig()
	cfg.URL = "ws://localhost:8080/ws"
	cfg.Token = token

	client := wirechat.NewClient(cfg)

	client.OnMessage(func(ev wirechat.MessageEvent) {
		fmt.Printf("  Sent message ID:%d\n", ev.ID)
	})

	client.OnError(func(err error) {
		log.Printf("Error: %v", err)
	})

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Join and send multiple messages to "general" (public room, auto-created)
	if err := client.Join(ctx, "general"); err != nil {
		log.Fatal(err)
	}

	for i := 1; i <= 5; i++ {
		msg := fmt.Sprintf("Test message #%d", i)
		if err := client.Send(ctx, "general", msg); err != nil {
			log.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)
	fmt.Println("=== Done populating ===")
}

func testHistory() {
	fmt.Println("=== Testing History Event ===")

	cfg := wirechat.DefaultConfig()
	cfg.URL = "ws://localhost:8080/ws"
	cfg.Token = token

	client := wirechat.NewClient(cfg)

	historyReceived := false

	client.OnHistory(func(ev wirechat.HistoryEvent) {
		historyReceived = true
		fmt.Printf("\n✓ History event received for room '%s'\n", ev.Room)
		fmt.Printf("  Total messages: %d\n", len(ev.Messages))
		for i, msg := range ev.Messages {
			fmt.Printf("  [%d] ID:%d User:%s Text:%s\n", i+1, msg.ID, msg.User, msg.Text)
		}
		fmt.Println()
	})

	client.OnMessage(func(ev wirechat.MessageEvent) {
		fmt.Printf("  New message: ID:%d User:%s Text:%s\n", ev.ID, ev.User, ev.Text)
	})

	client.OnUserJoined(func(ev wirechat.UserEvent) {
		fmt.Printf("  >>> %s joined\n", ev.User)
	})

	client.OnError(func(err error) {
		log.Printf("Error: %v", err)
	})

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("Joining room 'general'...")
	if err := client.Join(ctx, "general"); err != nil {
		log.Fatal(err)
	}

	// Wait for events
	time.Sleep(1 * time.Second)

	if !historyReceived {
		fmt.Println("\n✗ History event NOT received!")
		fmt.Println("  This might be expected if room has no persisted messages.")
	}

	fmt.Println("\n=== Test complete ===")
}
