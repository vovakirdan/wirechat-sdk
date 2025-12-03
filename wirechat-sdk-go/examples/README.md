# WireChat Go SDK Examples

Примеры использования WireChat Go SDK для различных сценариев.

## Запуск примеров

Все примеры можно запускать из корня SDK:

```bash
# Из корня wirechat-sdk-go/
go run ./examples/hello
go run ./examples/join-chat
go run ./examples/test-history
```

Или скомпилировать:

```bash
go build ./examples/hello
./hello
```

## Требования

Перед запуском примеров убедитесь, что:
1. **WireChat сервер запущен**: `http://localhost:8080`
2. Сервер доступен по WebSocket: `ws://localhost:8080/ws`

## Примеры

### [hello](./hello) - Базовый пример

Демонстрирует минимальный рабочий пример:
- Подключение к серверу
- Присоединение к комнате
- Отправка сообщения
- Получение событий (message, user_joined, user_left, history)

```bash
go run ./examples/hello
```

### [join-chat](./join-chat) - Интерактивный чат

Полнофункциональный CLI чат-клиент с поддержкой:
- Нескольких комнат
- Команд в стиле IRC (аргументы из CLI или env переменных)
- Real-time обработки событий

```bash
go run ./examples/join-chat ws://localhost:8080/ws myuser general
```

### [test-history](./test-history) - Тестирование History Event

Специальный пример для тестирования History Event:
- Использует JWT токен (authenticated пользователь)
- Populate режим: заполняет комнату сообщениями
- Test режим: проверяет получение истории

```bash
# Заполнить комнату сообщениями
go run ./examples/test-history populate

# Проверить получение истории
go run ./examples/test-history
```

**Примечание**: Для test-history нужен валидный JWT токен. Создайте пользователя через REST API:

```bash
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test123"}'
```

Полученный токен вставьте в `examples/test-history/main.go` (константа `token`).

## Советы

- **Guest mode**: Оставьте `cfg.Token = ""` для подключения как guest
- **Authenticated mode**: Получите JWT через `/api/register` или `/api/login` и установите `cfg.Token`
- **History event**: История отправляется только для комнат с сохраненными сообщениями (authenticated пользователи)
- **ReadTimeout**: По умолчанию `0` (infinite) - сервер управляет keepalive через ping/pong

## Отладка

Включите debug логи в примерах:

```go
client.SetLogger(MyDebugLogger{})
```

Или запустите сервер с `--log-level debug` для просмотра серверных логов.
