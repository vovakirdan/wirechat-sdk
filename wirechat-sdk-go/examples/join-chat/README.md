# Join Chat Example

Интерактивный CLI чат-клиент с возможностью отправки сообщений через stdin.

## Что демонстрирует

- Интерактивный ввод сообщений через stdin
- Real-time обработка событий в отдельных горутинах
- Graceful shutdown по сигналу (Ctrl+C)
- Все event handlers SDK
- Правильная работа с ReadTimeout = 0

## Запуск

```bash
go run ./examples/join-chat
```

Или скомпилировать:

```bash
go build ./examples/join-chat
./join-chat
```

## Использование

После запуска клиент:
1. Подключается к `ws://localhost:8080/ws` как `join-and-chat`
2. Присоединяется к комнате `general`
3. Входит в интерактивный режим - можно вводить сообщения
4. Отображает все события в реальном времени
5. Завершает работу по Ctrl+C

### Пример сессии

```bash
$ go run ./examples/join-chat

Connecting to ws://localhost:8080/ws...
Connected!
Joining room 'general'...
Joined room 'general'
>>> join-and-chat joined general
Type messages (Ctrl+D or Ctrl+C to exit)...

Hello everyone!
[general] join-and-chat: Hello everyone!
>>> bob joined general
How are you?
[general] join-and-chat: How are you?
[general] bob: I'm good, thanks!
<<< bob left general
^C
Shutting down...
Disconnected
```

## Настройка

По умолчанию:
- **URL**: `ws://localhost:8080/ws`
- **User**: `join-and-chat`
- **Room**: `general`
- **ReadTimeout**: `0` (infinite - правильный выбор для long-lived соединений)

Можно изменить в коде:

```go
cfg.URL = "ws://your-server:8080/ws"
cfg.User = "your-name"

// В функции run(), изменить комнату:
room := "your-room"
```

## События

Клиент отображает все события:

- **Сообщения**: `[room] user: text`
- **Join**: `>>> user joined room`
- **Leave**: `<<< user left room`
- **Ошибки**: `ERROR: code: message`

## Authenticated режим

Для использования JWT токена вместо guest режима, измените код:

```go
cfg.Token = "your-jwt-token"
cfg.User = "" // не нужен при JWT
```

## Отличия от hello

| Аспект | hello | join-chat |
|--------|-------|-----------|
| Интерактивность | Нет | Да (stdin input) |
| Время работы | 10 сек | До Ctrl+C |
| Ввод сообщений | Одно hardcoded | Любое через stdin |
| Сложность | Минимальная | Средняя |

## Для разработчиков

Этот пример показывает:
- Правильную обработку сигналов с `signal.NotifyContext`
- Использование контекста для graceful shutdown
- Паттерн конфигурации через env/args
- Структуру типичного CLI чат-клиента

Используйте как основу для более сложных клиентов (TUI, GUI и т.д.).
