# Test Auto-Reconnect Example

Простой тестовый клиент для демонстрации и тестирования автоматического переподключения.

## Возможности

- **Auto-reconnect**: Автоматическое переподключение с exponential backoff
- **State monitoring**: Отслеживание изменений состояния соединения
- **Error handling**: Обработка ошибок соединения
- **Configurable retries**: Настраиваемое количество попыток переподключения

## Запуск

```bash
# Убедитесь, что WireChat сервер запущен
cd /path/to/wirechat-server
make run

# Запустите тестовый клиент
cd examples/test-reconnect
go run main.go
```

## Тестирование

После запуска клиент подключится к серверу и будет ждать событий. Для тестирования переподключения:

1. **Клиент подключен**:
   ```
   [16:19:00] STATE: disconnected -> connecting
   [16:19:00] STATE: connecting -> connected
   ✓ Connected!
   ```

2. **Убейте сервер**:
   ```bash
   pkill -f wirechat-server
   ```

3. **Клиент обнаружит разрыв и начнет переподключение**:
   ```
   [16:19:02] ERROR: connection_error: read error (wrapped: EOF)
   [16:19:02] STATE: connected -> disconnected
   [16:19:04] STATE: disconnected -> reconnecting
   [16:19:08] STATE: reconnecting -> reconnecting  # exponential backoff
   ```

4. **Перезапустите сервер**:
   ```bash
   cd /path/to/wirechat-server
   make run
   ```

5. **Клиент успешно переподключится**:
   ```
   [16:19:16] STATE: reconnecting -> connected
   ```

## Конфигурация

В `main.go` можно настроить параметры переподключения:

```go
cfg.AutoReconnect = true                     // Включить auto-reconnect
cfg.ReconnectInterval = 2 * time.Second      // Начальная задержка (2s)
cfg.MaxReconnectDelay = 10 * time.Second     // Максимальная задержка (10s)
cfg.MaxReconnectTries = 5                    // Максимум 5 попыток (0 = бесконечно)
```

## Exponential Backoff

Задержка между попытками увеличивается экспоненциально:

- Попытка 1: 2s
- Попытка 2: 4s
- Попытка 3: 8s
- Попытка 4+: 10s (capped at MaxReconnectDelay)

## См. также

- [TESTING_RECONNECT.md](../../TESTING_RECONNECT.md) - Подробное руководство по тестированию
- [README.md](../../README.md) - Документация SDK с полным описанием auto-reconnect
