package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	// Создаем контекст с обработкой сигналов для корректного завершения
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Настраиваем конфигурацию клиента
	cfg := wirechat.DefaultConfig()
	cfg.URL = "ws://localhost:8080/ws"
	cfg.User = "hello-user" // Имя пользователя (используется, если JWT не требуется)
	cfg.Token = ""          // JWT токен (оставьте пустым, если JWT не требуется на сервере)

	// Создаем клиент
	client := wirechat.NewClient(cfg)

	// Настраиваем обработчики событий
	client.OnMessage(func(ev wirechat.MessageEvent) {
		if ev.ID != 0 {
			fmt.Printf("[%s] %s (ID:%d): %s\n", ev.Room, ev.User, ev.ID, ev.Text)
		} else {
			fmt.Printf("[%s] %s: %s\n", ev.Room, ev.User, ev.Text)
		}
	})

	client.OnHistory(func(ev wirechat.HistoryEvent) {
		fmt.Printf("\n=== History for room '%s' (%d messages) ===\n", ev.Room, len(ev.Messages))
		for _, msg := range ev.Messages {
			if msg.ID != 0 {
				fmt.Printf("  [%d] %s: %s\n", msg.ID, msg.User, msg.Text)
			} else {
				fmt.Printf("  %s: %s\n", msg.User, msg.Text)
			}
		}
		fmt.Println("=== End of history ===")
	})

	client.OnUserJoined(func(ev wirechat.UserEvent) {
		fmt.Printf(">>> %s joined room %s\n", ev.User, ev.Room)
	})

	client.OnUserLeft(func(ev wirechat.UserEvent) {
		fmt.Printf("<<< %s left room %s\n", ev.User, ev.Room)
	})

	client.OnError(func(err error) {
		log.Printf("error: %v", err)
	})

	// Подключаемся к серверу
	fmt.Printf("Connecting to %s...\n", cfg.URL)
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	fmt.Println("Connected successfully!")

	// Присоединяемся к комнате
	room := "general"
	fmt.Printf("Joining room '%s'...\n", room)
	if err := client.Join(ctx, room); err != nil {
		_ = client.Close()
		return fmt.Errorf("join: %w", err)
	}
	fmt.Printf("Joined room '%s'\n", room)

	// Ждем немного, чтобы убедиться, что соединение установлено
	time.Sleep(500 * time.Millisecond)

	// Отправляем сообщение "Hello, World!"
	message := "Hello from Go SDK!"
	fmt.Printf("Sending message: %s\n", message)
	if err := client.Send(ctx, room, message); err != nil {
		_ = client.Close()
		return fmt.Errorf("send: %w", err)
	}
	fmt.Println("Message sent!")

	// Ждем сигнала завершения или таймаут
	fmt.Println("Waiting for messages (Ctrl+C to exit)...")
	select {
	case <-ctx.Done():
		fmt.Println("\nShutting down...")
	case <-time.After(10 * time.Second):
		fmt.Println("\nTimeout reached, shutting down...")
	}

	// Закрываем соединение
	if err := client.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	fmt.Println("Disconnected")

	return nil
}
