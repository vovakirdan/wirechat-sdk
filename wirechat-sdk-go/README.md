# WireChat SDK для Go

Официальный Go SDK для подключения к WireChat серверу. Предоставляет высокоуровневый API для работы с WebSocket соединением, комнатами и сообщениями.

## Установка

```bash
go get wirechat-sdk-go
```

## Быстрый старт

```go
package main

import (
    "context"
    "fmt"
    "log"
    "wirechat-sdk-go/wirechat"
)

func main() {
    // Создаем конфигурацию
    cfg := wirechat.DefaultConfig()
    cfg.URL = "ws://localhost:8080/ws"
    cfg.User = "my-user" // или cfg.Token для JWT авторизации

    // Создаем клиент
    client := wirechat.NewClient(cfg)

    // Настраиваем обработчики событий
    client.OnMessage(func(ev wirechat.MessageEvent) {
        fmt.Printf("[%s] %s: %s\n", ev.Room, ev.User, ev.Text)
    })

    // Подключаемся
    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Присоединяемся к комнате и отправляем сообщение
    client.Join(ctx, "general")
    client.Send(ctx, "general", "Hello, World!")
}
```

## API Reference

### Config

`Config` определяет параметры подключения к серверу.

```go
type Config struct {
    URL              string        // WebSocket URL (например, "ws://localhost:8080/ws")
    Token            string        // JWT токен для авторизации (если требуется)
    User             string        // Имя пользователя (используется, если JWT не требуется)
    HandshakeTimeout time.Duration // Таймаут установления соединения
    ReadTimeout      time.Duration // Таймаут чтения сообщений
    WriteTimeout     time.Duration // Таймаут отправки сообщений
}
```

#### DefaultConfig()

Возвращает конфигурацию с разумными значениями по умолчанию:
- `HandshakeTimeout`: 10 секунд
- `ReadTimeout`: 30 секунд
- `WriteTimeout`: 10 секунд

Пример:

```go
cfg := wirechat.DefaultConfig()
cfg.URL = "ws://localhost:8080/ws"
cfg.User = "alice"
```

### Client

`Client` — основной тип SDK, предоставляющий методы для работы с сервером.

**Потокобезопасность:** Методы `Join`, `Leave`, `Send` и обработчики событий могут вызываться из разных горутин. Однако `Connect` и `Close` должны вызываться последовательно и не должны вызываться одновременно.

#### NewClient(cfg Config) *Client

Создает новый клиент с указанной конфигурацией.

```go
client := wirechat.NewClient(cfg)
```

#### SetLogger(l Logger)

Устанавливает кастомный логгер (опционально). По умолчанию используется no-op логгер, который игнорирует все логи.

```go
type MyLogger struct{}

func (l MyLogger) Debug(msg string, fields map[string]any) { /* ... */ }
func (l MyLogger) Info(msg string, fields map[string]any)  { /* ... */ }
func (l MyLogger) Warn(msg string, fields map[string]any)  { /* ... */ }
func (l MyLogger) Error(msg string, fields map[string]any) { /* ... */ }

client.SetLogger(MyLogger{})
```

#### Connect(ctx context.Context) error

Устанавливает WebSocket соединение с сервером, отправляет hello сообщение и запускает внутренние циклы чтения/записи.

**Важно:** Вызывайте `Connect` только один раз. Для переподключения создайте новый клиент.

```go
ctx := context.Background()
if err := client.Connect(ctx); err != nil {
    return fmt.Errorf("failed to connect: %w", err)
}
```

#### Join(ctx context.Context, room string) error

Присоединяется к указанной комнате. После успешного присоединения клиент будет получать все сообщения из этой комнаты.

```go
if err := client.Join(ctx, "general"); err != nil {
    return err
}
```

#### Leave(ctx context.Context, room string) error

Покидает указанную комнату. После этого клиент перестанет получать сообщения из этой комнаты.

```go
if err := client.Leave(ctx, "general"); err != nil {
    return err
}
```

#### Send(ctx context.Context, room, text string) error

Отправляет текстовое сообщение в указанную комнату. Клиент должен быть присоединен к комнате перед отправкой.

```go
if err := client.Send(ctx, "general", "Hello, everyone!"); err != nil {
    return err
}
```

#### Close() error

Корректно закрывает соединение и останавливает все внутренние горутины.

```go
defer client.Close()
```

### Обработка событий

SDK предоставляет методы для регистрации обработчиков различных событий.

#### OnMessage(fn func(MessageEvent))

Регистрирует обработчик входящих сообщений.

```go
client.OnMessage(func(ev wirechat.MessageEvent) {
    fmt.Printf("[%s] %s: %s\n", ev.Room, ev.User, ev.Text)
})
```

#### OnUserJoined(fn func(UserEvent))

Регистрирует обработчик события присоединения пользователя к комнате.

```go
client.OnUserJoined(func(ev wirechat.UserEvent) {
    fmt.Printf(">>> %s joined %s\n", ev.User, ev.Room)
})
```

#### OnUserLeft(fn func(UserEvent))

Регистрирует обработчик события выхода пользователя из комнаты.

```go
client.OnUserLeft(func(ev wirechat.UserEvent) {
    fmt.Printf("<<< %s left %s\n", ev.User, ev.Room)
})
```

#### OnError(fn func(error))

Регистрирует обработчик ошибок протокола и ошибок соединения. Обработчик вызывается при:
- Ошибках протокола (например, `unauthorized`, `rate_limited`)
- Ошибках чтения/записи WebSocket соединения
- Ошибках десериализации входящих сообщений

```go
client.OnError(func(err error) {
    log.Printf("SDK error: %v", err)
})
```

**Важно:** После ошибки соединения клиент может быть в неработоспособном состоянии. Рекомендуется пересоздать клиент для переподключения.

### Типы событий

#### MessageEvent

Событие входящего сообщения.

```go
type MessageEvent struct {
    Room string `json:"room"` // Название комнаты
    User string `json:"user"`  // Имя отправителя
    Text string `json:"text"`  // Текст сообщения
    TS   int64  `json:"ts"`   // Unix timestamp в секундах
}
```

#### UserEvent

Событие присоединения или выхода пользователя.

```go
type UserEvent struct {
    Room string `json:"room"` // Название комнаты
    User string `json:"user"` // Имя пользователя
}
```

## Примеры использования

### Базовый пример с обработкой сообщений

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "wirechat-sdk-go/wirechat"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    cfg := wirechat.DefaultConfig()
    cfg.URL = "ws://localhost:8080/ws"
    cfg.User = "my-user"

    client := wirechat.NewClient(cfg)

    client.OnMessage(func(ev wirechat.MessageEvent) {
        fmt.Printf("[%s] %s: %s\n", ev.Room, ev.User, ev.Text)
    })

    client.OnError(func(err error) {
        log.Printf("Error: %v", err)
    })

    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    if err := client.Join(ctx, "general"); err != nil {
        log.Fatal(err)
    }

    // Отправляем сообщение
    client.Send(ctx, "general", "Hello!")

    // Ждем сигнала завершения
    <-ctx.Done()
}
```

### Использование JWT авторизации

```go
cfg := wirechat.DefaultConfig()
cfg.URL = "ws://localhost:8080/ws"
cfg.Token = "your-jwt-token-here"
// cfg.User не требуется при использовании JWT

client := wirechat.NewClient(cfg)
```

### Работа с несколькими комнатами

```go
// Присоединяемся к нескольким комнатам
client.Join(ctx, "general")
client.Join(ctx, "random")
client.Join(ctx, "dev")

// Отправляем сообщения в разные комнаты
client.Send(ctx, "general", "Hello general!")
client.Send(ctx, "dev", "Hello dev!")

// Покидаем комнату
client.Leave(ctx, "random")
```

### Полный пример

См. [examples/hello/main.go](examples/hello/main.go) для полного рабочего примера.

## Обработка ошибок

SDK возвращает ошибки в следующих случаях:

- **`Connect`**: ошибки установления соединения, неверный URL, ошибки handshake
- **`Join`/`Leave`/`Send`**: ошибки протокола (например, `rate_limited`, `not_in_room`), ошибки записи в соединение, клиент не подключен

Все ошибки протокола также передаются в обработчик `OnError`. Ошибки соединения автоматически обрабатываются внутренними циклами и передаются в `OnError`.

## Требования

- Go 1.25+
- [WireChat сервер](https://github.com/vovakirdan/wirechat-server)

## Лицензия
