# Hello Example

Минимальный рабочий пример WireChat Go SDK.

## Что демонстрирует

- Подключение к WireChat серверу в guest режиме
- Присоединение к комнате `general`
- Отправка сообщения
- Обработка событий:
  - `OnMessage` - входящие сообщения с Message ID
  - `OnHistory` - история сообщений при join (если есть)
  - `OnUserJoined` - уведомления о присоединении пользователей
  - `OnUserLeft` - уведомления о выходе пользователей
  - `OnError` - ошибки протокола и соединения

## Запуск

```bash
# Из корня wirechat-sdk-go/
go run ./examples/hello

# Или скомпилировать
go build ./examples/hello
./hello
```

## Ожидаемый вывод

```
Connecting to ws://localhost:8080/ws...
Connected successfully!
Joining room 'general'...
Joined room 'general'
>>> hello-user joined room general
Sending message: Hello from Go SDK!
Message sent!
Waiting for messages (Ctrl+C to exit)...
[general] hello-user: Hello from Go SDK!

Shutting down...
Disconnected
```

## Если есть история

При повторном запуске, когда в комнате уже есть сохраненные сообщения:

```
=== History for room 'general' (5 messages) ===
  [1] testuser: Test message #1
  [2] testuser: Test message #2
  ...
=== End of history ===
```

**Примечание**: История отправляется только для authenticated пользователей. Guest сообщения не сохраняются в БД.

## Таймаут

Пример автоматически завершается через 10 секунд или по Ctrl+C.

## Настройка

Можно изменить URL и имя пользователя в коде:

```go
cfg.URL = "ws://your-server:8080/ws"
cfg.User = "your-name"
```

Или использовать JWT токен для authenticated режима:

```go
cfg.Token = "your-jwt-token"
cfg.User = "" // не нужен при использовании JWT
```
