package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := wirechat.DefaultConfig()
	cfg.URL = "ws://localhost:8080/ws"
	cfg.User = "join-and-chat"
	cfg.ReadTimeout = 0 // Disable read timeout - server handles idle detection with ping/pong

	client := wirechat.NewClient(cfg)

	client.OnMessage(func(ev wirechat.MessageEvent) {
		fmt.Printf("[%s] %s: %s\n", ev.Room, ev.User, ev.Text)
	})

	client.OnUserJoined(func(ev wirechat.UserEvent) {
		fmt.Printf(">>> %s joined %s\n", ev.User, ev.Room)
	})

	client.OnUserLeft(func(ev wirechat.UserEvent) {
		fmt.Printf("<<< %s left %s\n", ev.User, ev.Room)
	})

	client.OnError(func(err error) {
		fmt.Printf("error: %v\n", err)
	})

	fmt.Printf("Connecting to %s...\n", cfg.URL)
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Close()

	room := "general"
	if err := client.Join(ctx, room); err != nil {
		return fmt.Errorf("join: %w", err)
	}

	fmt.Println("Connected. Type messages to chat, /quit to exit.")

	inputCh := make(chan string)
	go readInput(inputCh)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nShutting down...")
			return nil
		case line, ok := <-inputCh:
			if !ok {
				fmt.Println("\nInput closed.")
				return nil
			}
			msg := strings.TrimSpace(line)
			if msg == "" {
				continue
			}
			if msg == "/quit" {
				fmt.Println("Bye!")
				return nil
			}
			if err := client.Send(ctx, room, msg); err != nil {
				return fmt.Errorf("send: %w", err)
			}
		}
	}
}

func readInput(dst chan<- string) {
	defer close(dst)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		dst <- scanner.Text()
	}
}
