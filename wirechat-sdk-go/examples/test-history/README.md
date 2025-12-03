# Test History Example

Специальный пример для тестирования History Event и Message ID полей в Go SDK.

## Что тестирует

- **History Event**: Проверяет получение истории сообщений при join
- **Message ID**: Подтверждает, что сообщения содержат ID из БД
- **Authenticated mode**: Использует JWT токен для сохранения сообщений

## Два режима работы

### Populate Mode

Заполняет комнату `general` тестовыми сообщениями от authenticated пользователя:

```bash
go run ./examples/test-history populate
```

Вывод:
```
=== Populating room with messages ===
  Sent message ID:1
  Sent message ID:2
  Sent message ID:3
  Sent message ID:4
  Sent message ID:5
=== Done populating ===
```

### Test Mode (по умолчанию)

Подключается к комнате и проверяет получение истории:

```bash
go run ./examples/test-history
```

Вывод при успехе:
```
=== Testing History Event ===
Joining room 'general'...
  >>> testuser joined

✓ History event received for room 'general'
  Total messages: 5
  [1] ID:1 User:testuser Text:Test message #1
  [2] ID:2 User:testuser Text:Test message #2
  ...

=== Test complete ===
```

## Подготовка

### 1. Создайте пользователя

```bash
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test123"}'
```

Ответ:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### 2. Обновите токен в коде

Скопируйте токен из ответа и вставьте в `main.go`:

```go
const token = "YOUR_JWT_TOKEN_HERE"
```

### 3. Запустите тесты

```bash
# Заполнить комнату
go run ./examples/test-history populate

# Проверить историю
go run ./examples/test-history
```

## Как это работает

### History Event

Согласно WireChat Protocol v1:
- History отправляется **только при join** в комнату
- Содержит последние 20 сообщений из БД
- Сообщения упорядочены хронологически
- История отправляется только если:
  - В БД есть сохраненные сообщения для этой комнаты
  - Пользователь имеет доступ к комнате

### Message ID

- **Authenticated пользователи**: ID присваивается БД (> 0)
- **Guest пользователи**: ID = 0 (сообщения не сохраняются)

## Устранение неполадок

### "History event NOT received"

Это нормально если:
- Комната пустая (нет сохраненных сообщений)
- В комнате были только guest сообщения (ID = 0)

**Решение**: Запустите `populate` режим для создания authenticated сообщений.

### "connection refused"

Сервер не запущен. Запустите:

```bash
cd /path/to/wirechat-server
make run
```

### "unauthorized" / "invalid token"

Токен истек или невалиден. Создайте нового пользователя и обновите токен в коде.

## Для разработчиков

Этот пример полезен для:
- Проверки корректности парсинга History Event
- Валидации Message ID поля
- Тестирования authenticated режима
- Отладки WebSocket протокола

При добавлении новых фич в протокол (например, read receipts, typing indicators), создавайте аналогичные тестовые примеры.
