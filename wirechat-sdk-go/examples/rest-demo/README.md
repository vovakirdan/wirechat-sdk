# REST API Demo

Полный пример использования REST API клиента WireChat SDK.

## Что демонстрирует

Этот пример показывает полный цикл работы с WireChat через REST API и WebSocket:

1. **Authentication**: Регистрация нового пользователя
2. **Room Management**: Создание публичной комнаты
3. **List Rooms**: Получение списка доступных комнат
4. **WebSocket Integration**: Подключение и отправка сообщений
5. **Message History**: Получение истории через REST API
6. **Pagination**: Работа с cursor-based пагинацией

## Запуск

```bash
# Из корня wirechat-sdk-go/
go run ./examples/rest-demo

# Или скомпилировать
go build ./examples/rest-demo
./rest-demo
```

## Требования

- WireChat сервер запущен на `http://localhost:8080`
- REST API доступен на `/api`
- WebSocket доступен на `/ws`

## Ожидаемый вывод

```
=== WireChat REST API Demo ===

1. Registering new user...
   ✓ Registered as 'demo-user-1733228845'
   Token: eyJhbGciOiJIUzI1NiI...

2. Creating public room...
   ✓ Created room 'demo-room-1733228845' (ID: 42)

3. Listing all rooms...
   Found 2 room(s):
   - general (ID: 1, Type: public)
   - demo-room-1733228845 (ID: 42, Type: public)

4. Connecting via WebSocket...
   ✓ Connected via WebSocket
   ✓ Joined room 'demo-room-1733228845'

5. Sending messages...
   [WS] Message ID:123 from demo-user-1733228845: Test message #1
   [WS] Message ID:124 from demo-user-1733228845: Test message #2
   [WS] Message ID:125 from demo-user-1733228845: Test message #3

6. Fetching message history via REST API...
   Found 3 message(s):
   - [ID:123] demo-user-1733228845: Test message #1
   - [ID:124] demo-user-1733228845: Test message #2
   - [ID:125] demo-user-1733228845: Test message #3

=== Demo Complete ===
```

## Основные концепции

### Unified Client

SDK предоставляет единый клиент с доступом к обоим API:

```go
cfg := wirechat.DefaultConfig()
cfg.URL = "ws://localhost:8080/ws"           // WebSocket
cfg.RESTBaseURL = "http://localhost:8080/api" // REST API

client := wirechat.NewClient(&cfg)

// REST API доступен через client.REST
user, err := client.REST.Register(ctx, ...)
rooms, err := client.REST.ListRooms(ctx)
history, err := client.REST.GetMessages(ctx, roomID, limit, before)
```

### Authentication Flow

```go
// 1. Register
resp, err := client.REST.Register(ctx, rest.RegisterRequest{
    Username: "alice",
    Password: "secret123",
})

// 2. Update token for subsequent requests
client.REST.SetToken(resp.Token)

// 3. Use token for WebSocket connection
cfg.Token = resp.Token
wsClient := wirechat.NewClient(&cfg)
```

### Message History Pagination

История сообщений использует cursor-based пагинацию:

```go
// Fetch first page (latest 20 messages)
page1, err := client.REST.GetMessages(ctx, roomID, 20, nil)

// Fetch next page (older messages)
if page1.HasMore {
    lastID := page1.Messages[len(page1.Messages)-1].ID
    page2, err := client.REST.GetMessages(ctx, roomID, 20, &lastID)
}
```

### Room Types

SDK поддерживает все типы комнат:

```go
// Public room (anyone can join)
client.REST.CreateRoom(ctx, rest.CreateRoomRequest{
    Name: "general",
    Type: rest.RoomTypePublic,
})

// Private room (invite-only)
client.REST.CreateRoom(ctx, rest.CreateRoomRequest{
    Name: "private-chat",
    Type: rest.RoomTypePrivate,
})

// Direct message room (1-on-1)
client.REST.CreateDirectRoom(ctx, rest.CreateDirectRoomRequest{
    UserID: 42, // peer user ID
})
```

## Обработка ошибок

REST клиент возвращает осмысленные ошибки:

```go
room, err := client.REST.CreateRoom(ctx, req)
if err != nil {
    // Error format: "api error (status 400): room name already exists"
    log.Printf("Failed: %v", err)
    return err
}
```

## Для разработчиков

Этот пример показывает:
- Интеграцию REST и WebSocket API
- Правильную работу с токенами
- Pagination pattern для истории
- Best practices для unified client

Используйте как основу для полнофункциональных чат-приложений с регистрацией, управлением комнатами и историей сообщений.
