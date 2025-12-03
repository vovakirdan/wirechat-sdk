# WireChat SDK для Go

Официальный Go SDK для подключения к WireChat серверу. Предоставляет высокоуровневый API для работы с WebSocket соединением, комнатами, сообщениями и REST API.

## Возможности

- **WebSocket API**: Real-time подключение для чата
- **REST API**: Управление комнатами, аутентификация, история сообщений
- **Unified Client**: Единый клиент для WebSocket и REST API
- **Event-driven**: Обработчики событий для сообщений, присоединений, истории
- **Protocol v1**: Полная поддержка WireChat Protocol Version 1
- **Type-safe**: Строгая типизация всех API

## Установка

```bash
go get github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go
```

## Быстрый старт

### WebSocket API

```go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat"
)

func main() {
    // Создаем конфигурацию
    cfg := wirechat.DefaultConfig()
    cfg.URL = "ws://localhost:8080/ws"
    cfg.User = "my-user" // или cfg.Token для JWT авторизации

    // Создаем клиент
    client := wirechat.NewClient(&cfg)

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

### REST API

```go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat"
    "github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go/wirechat/rest"
)

func main() {
    ctx := context.Background()

    // Создаем клиент с REST API
    cfg := wirechat.DefaultConfig()
    cfg.URL = "ws://localhost:8080/ws"
    cfg.RESTBaseURL = "http://localhost:8080/api"

    client := wirechat.NewClient(&cfg)

    // Регистрируем нового пользователя
    resp, err := client.REST.Register(ctx, rest.RegisterRequest{
        Username: "alice",
        Password: "secret123",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Обновляем токен для WebSocket
    client.REST.SetToken(resp.Token)
    cfg.Token = resp.Token

    // Создаем комнату
    room, err := client.REST.CreateRoom(ctx, rest.CreateRoomRequest{
        Name: "my-room",
        Type: rest.RoomTypePublic,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Created room: %s (ID: %d)\n", room.Name, room.ID)

    // Получаем историю сообщений
    history, err := client.REST.GetMessages(ctx, room.ID, 20, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d messages\n", len(history.Messages))
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
    Protocol         int           // Версия протокола (по умолчанию 1)
    RESTBaseURL      string        // REST API base URL (например, "http://localhost:8080/api")
    HandshakeTimeout time.Duration // Таймаут установления соединения
    ReadTimeout      time.Duration // Таймаут чтения сообщений (0 = infinite, рекомендуется)
    WriteTimeout     time.Duration // Таймаут отправки сообщений
}
```

#### DefaultConfig()

Возвращает конфигурацию с разумными значениями по умолчанию:
- `HandshakeTimeout`: 10 секунд
- `ReadTimeout`: 0 (infinite - рекомендуется для long-lived соединений)
- `WriteTimeout`: 10 секунд
- `Protocol`: 1 (текущая версия протокола)

**Важно о ReadTimeout**: Значение 0 означает бесконечное ожидание. Это рекомендуемое значение для чат-соединений, так как сервер управляет liveness через WebSocket ping/pong механизм.

Пример:

```go
cfg := wirechat.DefaultConfig()
cfg.URL = "ws://localhost:8080/ws"
cfg.RESTBaseURL = "http://localhost:8080/api"
cfg.User = "alice"
```

### Client

`Client` — основной тип SDK, предоставляющий методы для работы с сервером.

**Потокобезопасность:** Методы `Join`, `Leave`, `Send` и обработчики событий могут вызываться из разных горутин. Однако `Connect` и `Close` должны вызываться последовательно и не должны вызываться одновременно.

**REST API**: Клиент предоставляет доступ к REST API через поле `REST *rest.Client`. REST клиент автоматически инициализируется, если в `Config` указан `RESTBaseURL`.

#### NewClient(cfg *Config) *Client

Создает новый клиент с указанной конфигурацией.

```go
cfg := wirechat.DefaultConfig()
cfg.URL = "ws://localhost:8080/ws"
cfg.RESTBaseURL = "http://localhost:8080/api"
client := wirechat.NewClient(&cfg)

// WebSocket API
client.OnMessage(...)
client.Connect(ctx)

// REST API
client.REST.Register(ctx, ...)
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

#### OnHistory(fn func(HistoryEvent))

Регистрирует обработчик события истории сообщений. Сервер отправляет историю (последние 20 сообщений) при присоединении к комнате.

**Важно**: История отправляется только для authenticated пользователей в комнатах с сохраненными сообщениями.

```go
client.OnHistory(func(ev wirechat.HistoryEvent) {
    fmt.Printf("History for %s: %d messages\n", ev.Room, len(ev.Messages))
    for _, msg := range ev.Messages {
        fmt.Printf("  [ID:%d] %s: %s\n", msg.ID, msg.User, msg.Text)
    }
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
    ID   int64  `json:"id"`   // ID сообщения (0 для guest сообщений, >0 для сохраненных)
    Room string `json:"room"` // Название комнаты
    User string `json:"user"` // Имя отправителя
    Text string `json:"text"` // Текст сообщения
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

#### HistoryEvent

Событие истории сообщений (отправляется при присоединении к комнате).

```go
type HistoryEvent struct {
    Room     string         `json:"room"`     // Название комнаты
    Messages []MessageEvent `json:"messages"` // Массив сообщений (до 20 последних)
}
```

## REST API

SDK предоставляет полнофункциональный REST API клиент для управления комнатами, аутентификации и получения истории сообщений.

### Доступ к REST API

```go
cfg := wirechat.DefaultConfig()
cfg.RESTBaseURL = "http://localhost:8080/api"
client := wirechat.NewClient(&cfg)

// REST API доступен через client.REST
resp, err := client.REST.Register(ctx, ...)
```

### Authentication API

#### Register

Регистрация нового пользователя.

```go
resp, err := client.REST.Register(ctx, rest.RegisterRequest{
    Username: "alice",
    Password: "secret123",
})
// resp.Token содержит JWT токен
```

#### Login

Вход существующего пользователя.

```go
resp, err := client.REST.Login(ctx, rest.LoginRequest{
    Username: "alice",
    Password: "secret123",
})
```

#### GuestLogin

Получение guest токена.

```go
resp, err := client.REST.GuestLogin(ctx)
```

#### SetToken

Обновление токена для последующих запросов.

```go
client.REST.SetToken(token)
// Также обновите cfg.Token для WebSocket
cfg.Token = token
```

### Room Management API

#### CreateRoom

Создание публичной или приватной комнаты.

```go
room, err := client.REST.CreateRoom(ctx, rest.CreateRoomRequest{
    Name: "my-room",
    Type: rest.RoomTypePublic, // или rest.RoomTypePrivate
})
// room.ID, room.Name, room.Type
```

#### ListRooms

Получение списка доступных комнат.

```go
rooms, err := client.REST.ListRooms(ctx)
for _, room := range rooms {
    fmt.Printf("Room: %s (ID: %d, Type: %s)\n", room.Name, room.ID, room.Type)
}
```

#### CreateDirectRoom

Создание комнаты для direct-сообщений (1-on-1).

```go
room, err := client.REST.CreateDirectRoom(ctx, rest.CreateDirectRoomRequest{
    UserID: 42, // ID пользователя-собеседника
})
```

### Message History API

#### GetMessages

Получение истории сообщений с cursor-based пагинацией.

```go
// Первая страница (последние 20 сообщений)
history, err := client.REST.GetMessages(ctx, roomID, 20, nil)

fmt.Printf("Found %d messages\n", len(history.Messages))
for _, msg := range history.Messages {
    fmt.Printf("[ID:%d] %s: %s\n", msg.ID, msg.User, msg.Body)
}

// Следующая страница (более старые сообщения)
if history.HasMore {
    lastID := history.Messages[len(history.Messages)-1].ID
    olderHistory, err := client.REST.GetMessages(ctx, roomID, 20, &lastID)
}
```

Параметры:
- `roomID int64`: ID комнаты
- `limit int`: Количество сообщений (max 100)
- `before *int64`: Курсор (ID сообщения), с которого начинать выборку (nil = с конца)

### Unified Client Pattern

Используйте WebSocket и REST API в одном клиенте:

```go
cfg := wirechat.DefaultConfig()
cfg.URL = "ws://localhost:8080/ws"
cfg.RESTBaseURL = "http://localhost:8080/api"

client := wirechat.NewClient(&cfg)

// 1. Register via REST
resp, _ := client.REST.Register(ctx, rest.RegisterRequest{...})
client.REST.SetToken(resp.Token)
cfg.Token = resp.Token

// 2. Create room via REST
room, _ := client.REST.CreateRoom(ctx, rest.CreateRoomRequest{...})

// 3. Connect via WebSocket
client.OnMessage(func(ev wirechat.MessageEvent) { ... })
client.Connect(ctx)
client.Join(ctx, room.Name)

// 4. Send messages via WebSocket
client.Send(ctx, room.Name, "Hello!")

// 5. Fetch history via REST
history, _ := client.REST.GetMessages(ctx, room.ID, 20, nil)
```

См. [examples/rest-demo](examples/rest-demo) для полного примера.

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

    client := wirechat.NewClient(&cfg)

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

client := wirechat.NewClient(&cfg)
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

### Примеры из репозитория

SDK включает несколько полнофункциональных примеров:

- **[examples/hello](examples/hello)**: Базовый пример с WebSocket (подключение, join, send, history)
- **[examples/join-chat](examples/join-chat)**: Интерактивный CLI чат-клиент
- **[examples/test-history](examples/test-history)**: Демонстрация History Event
- **[examples/rest-demo](examples/rest-demo)**: Полный пример REST API + WebSocket integration

См. [examples/README.md](examples/README.md) для деталей.

## Обработка ошибок

SDK возвращает ошибки в следующих случаях:

- **`Connect`**: ошибки установления соединения, неверный URL, ошибки handshake
- **`Join`/`Leave`/`Send`**: ошибки протокола (например, `rate_limited`, `not_in_room`), ошибки записи в соединение, клиент не подключен

Все ошибки протокола также передаются в обработчик `OnError`. Ошибки соединения автоматически обрабатываются внутренними циклами и передаются в `OnError`.

## Требования

- Go 1.25+
- [WireChat сервер](https://github.com/vovakirdan/wirechat-server)

## Лицензия
