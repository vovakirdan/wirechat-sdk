# WireChat SDK v1 – Comprehensive Spec

## 0. Цели и границы

**Цель SDK:**
Дать *единый* высокоуровневый клиентский API поверх WireChat Protocol v1:

* Управление WebSocket-соединением.
* Команды `hello/join/leave/msg`.
* Получение событий (`message/user_joined/user_left/history/error`).
* Удобные обёртки над REST API (Authentication, Rooms, History).
* Авто-переподключение (опционально).
* Минимум “ручной возни” с протоколом – приложение живёт через события и методы клиента.

**Не делает:**

* Не рендерит UI.
* Не решает авторизацию (кроме передачи готового JWT).
* Не реализует persistence на клиенте (кеши, локальные БД и т.п. – вне скопа).

---

## 1. Базовые концепции

SDK крутится вокруг нескольких сущностей:

* `Config` – неизменяемая конфигурация клиента.
* `Client` – основной объект, держит WebSocket и знает про REST.
* `Event Handlers` – колбэки/слушатели, куда летят события.
* `Models` – типы `Message`, `HistoryEvent`, `UserEvent`, `RoomInfo`, `HistoryChunk`, `WirechatError`, `ConnectionState` и др.

Все SDK на разных языках должны реализовывать это с сохранением сути.

---

## 2. Типы и модели данных

Ниже типы в псевдосинтаксисе:

### 2.1. Конфигурация клиента

```text
type Duration = int64  // миллисекунды или языкоспецифичный тип

type Config struct {
    // WebSocket
    WSURL            string    // "ws://host:8080/ws"
    Protocol         int       // версия протокола, сейчас всегда 1

    // REST
    RESTBaseURL      string    // "http://host:8080/api"
    HTTPClient       HttpClient?  // опционально, языкоспецифичный тип

    // Auth
    Token            string?   // JWT для авторизованных
    User             string?   // имя гостя для guest-mode

    // Timeouts
    HandshakeTimeout Duration  // таймаут установления WS-соединения
    WriteTimeout     Duration  // таймаут записи в сокет
    // ВАЖНО: ReadTimeout либо отсутствует, либо == 0 (бесконечный)

    // Auto reconnect
    AutoReconnect       bool
    ReconnectMinDelay   Duration  // min backoff, напр. 1s
    ReconnectMaxDelay   Duration  // max backoff, напр. 30s
    MaxReconnectRetries int?      // null/0 = бесконечно или языкоспецифично

    // Logging
    Logger Logger?      // интерфейс логгера, опционален
}
```

**DefaultConfig (рекомендуемые дефолты):**

* `Protocol = 1`
* `HandshakeTimeout = 10s`
* `WriteTimeout = 10s`
* `AutoReconnect = false`
* `ReconnectMinDelay = 1s`
* `ReconnectMaxDelay = 30s`
* `MaxReconnectRetries = 0` (интерпретация: “без ограничений” или “до остановки контекста”)
* `RESTBaseURL` может быть выведен из `WSURL` (но лучше явно задавать).

---

### 2.2. Состояния соединения

```text
enum ConnectionState {
    Disconnected,  // клиент создан, но нет активного соединения
    Connecting,    // в процессе Connect / Reconnect
    Connected,     // hello успешен, read-loop активен
    Closing,       // в процессе graceful close
    Closed,        // закрыто по инициативе клиента
    Error          // фатальная ошибка, без AutoReconnect или после исчерпания попыток
}

type StateEvent struct {
    State ConnectionState
    Error error?   // последняя ошибка (для Error/Disconnected)
}
```

---

### 2.3. Сообщения и события

#### Message

```text
type Message struct {
    Room string  // room name
    User string  // username
    Text string  // message text
    ID   int64   // message id from DB, 0 for guest messages
    TS   int64   // unix timestamp (seconds)
}
```

#### Events

```text
// По сути то же, что Message
type MessageEvent = Message

type HistoryEvent struct {
    Room     string
    Messages []Message  // последние N сообщений, упорядочены для UI (обычно по возрастанию TS/ID)
}

type UserEvent struct {
    Room string
    User string
}
```

---

### 2.4. Комнаты и история (REST)

```text
enum RoomType { Public, Private, Direct }

type RoomInfo struct {
    ID        int64
    Name      string
    Type      RoomType
    OwnerID   int64?    // null для публичных/не-владельческих
    CreatedAt Timestamp  // языкоспецифичная дата/время
}

type HistoryMessage struct {
    ID        int64
    RoomID    int64
    UserID    int64
    User      string
    Body      string
    CreatedAt Timestamp
}

type HistoryChunk struct {
    Messages []HistoryMessage
    HasMore  bool
}
```

---

### 2.5. Ошибки

```text
type ErrorCode string

const (
    ErrUnsupportedVersion ErrorCode = "unsupported_version"
    ErrUnauthorized       ErrorCode = "unauthorized"
    ErrInvalidMessage     ErrorCode = "invalid_message"
    ErrBadRequest         ErrorCode = "bad_request"
    ErrRoomNotFound       ErrorCode = "room_not_found"
    ErrAlreadyJoined      ErrorCode = "already_joined"
    ErrNotInRoom          ErrorCode = "not_in_room"
    ErrAccessDenied       ErrorCode = "access_denied"
    ErrRateLimited        ErrorCode = "rate_limited"
    ErrInternalError      ErrorCode = "internal_error"
)

type WirechatError struct {
    Code      ErrorCode?  // null, если нет кода сервера (локальная/сет. ошибка)
    Message   string      // человекочитаемый текст
    Temporary bool        // можно ли пробовать ретраи/переподключение
    Cause     error?      // вложенная ошибка, если есть
}
```

**Правила:**

* Любой `type:"error"` от сервера → `WirechatError` с `Code != null`.
* Любая ошибка транспорта (network/JSON/Handshake) → `WirechatError` с:

  * `Code = ErrInternalError` или `null` (на усмотрение реализации),
  * `Temporary = true` для network-временных ошибок (timeout, connection reset),
  * `Temporary = false` для жёстких ошибок конфигурации (невалидный URL и т.п.).

---

## 3. Публичный API клиента

Обозначения:

* `Client` – основной тип.
* `Ctx` / `context` – языкоспецифичный механизм отмены/таймаутов:

  * в Go – `context.Context`,
  * в Python – `asyncio` + таймауты/отмена задач,
  * в Rust – `Future` + cancel, etc.

### 3.1. Конструирование и конфиг

```text
func DefaultConfig() Config

func NewClient(cfg Config) *Client
```

* `DefaultConfig()` задаёт дефолты, но **не** подставляет `WSURL` и `RESTBaseURL`.
* `NewClient(cfg)`:

  * сохраняет копию `Config` внутри клиента,
  * устанавливает начальное состояние `ConnectionState.Disconnected`,
  * не делает никаких сетевых вызовов.

---

### 3.2. Жизненный цикл соединения

#### Connect

```text
func (c *Client) Connect(ctx) error
```

**Поведение:**

1. Состояние → `Connecting` + `OnStateChanged`.

2. Открывает WebSocket по `cfg.WSURL` с таймаутом `HandshakeTimeout`.

3. После успешного upgrade:

   * отправляет `hello`:

     ```json
     {
       "type": "hello",
       "data": {
         "protocol": 1,
         "token": "<cfg.Token>" или опущен,
         "user":  "<cfg.User>"  или опущен
       }
     }
     ```

4. Если сервер вернул ошибку `unsupported_version` или `unauthorized`:

   * закрыть соединение,
   * вернуть `WirechatError` с соответствующим `Code` и `Temporary=false`.

5. При успехе:

   * запускает read-loop и write-loop (если есть) в фоновом режиме.
   * Состояние → `Connected` + вызывается `OnStateChanged`.

**Гарантии:**

* `Connect` нельзя вызывать параллельно или повторно на уже подключённом клиенте (можно сделать защиту и возвращать ошибку).
* При `AutoReconnect=false`:

  * любое падение read/write-loop ставит `State=Error` и требует явного `Close()` + создания нового клиента.
* При `AutoReconnect=true` поведение описано ниже (см. раздел 4).

---

#### Close

```text
func (c *Client) Close() error
```

**Поведение:**

1. Состояние → `Closing` + `OnStateChanged`.
2. Останавливает read/write-loop.
3. Посылает WebSocket close frame (если протокол языка позволяет).
4. Закрывает соединение.
5. Состояние → `Closed` + `OnStateChanged`.
6. Авто-reconnect при этом **не** запускается.

Можно считать `Client` после `Close()` логически завершённым объектом.

---

#### State

```text
func (c *Client) State() ConnectionState
```

Возвращает текущий `ConnectionState` (thread-safe).

---

### 3.3. Работа с комнатами (WebSocket)

Все эти методы используют уже установленное соединение.

#### Join

```text
func (c *Client) Join(ctx, room string) error
```

**Поведение:**

1. Локальная валидация:

   * пустое/невалидное имя комнаты → локальная ошибка `WirechatError{Code:ErrBadRequest,...}`.
   * если `State != Connected` → ошибка `WirechatError{Code:ErrInternalError,...}` или отдельный код.

2. Отправка:

   ```json
   {
     "type": "join",
     "data": { "room": "<room>" }
   }
   ```

3. Ожидание подтверждения:

   * WireChat протокол не возвращает отдельный `ok`, поэтому SDK **либо**:

     * опирается на отсутствие ошибки в разумный таймаут, **либо**
     * читает первые сигналы (`user_joined`/`history`) и считает join успешным.
   * Рекомендуемая модель для SDK: `Join` отправляет команду и ждёт не более N секунд:

     * если приходит `type:"error"` с `code` → вернуть соответствующий `WirechatError`.
     * если по таймауту нет ни ошибки, ни сетевого сбоя → вернуть локальную ошибку `Temporary=true`.

4. При успехе:

   * добавить комнату в `joinedRooms` внутри клиента для последующего авто-reconnect.
   * дальше события (`user_joined`, `history`, `message`) прилетают через Registered handlers.

#### Leave

```text
func (c *Client) Leave(ctx, room string) error
```

**Поведение:**

* Аналогично `Join`, только с:

  ```json
  {
    "type": "leave",
    "data": { "room": "<room>" }
  }
  ```

* При успехе – убрать комнату из локального списка joinedRooms.

* Сервер разошлёт `user_left` другим участникам; мы получим через `OnUserLeft`.

#### SendMessage

```text
func (c *Client) SendMessage(ctx, room string, text string) error
```

**Поведение:**

* Локальная проверка пустых полей.

* Если клиент не в `Connected` → ошибка.

* Команда:

  ```json
  {
    "type": "msg",
    "data": {
      "room": "<room>",
      "text": "<text>"
    }
  }
  ```

* Не ожидает подтверждения; ошибки (`not_in_room`, `rate_limited`, …) приходят как `type:"error"` и оборачиваются в `WirechatError` + `OnError`.

---

### 3.4. REST-обёртки

> Это расширение SDK, но для v1 рекомендуется иметь хотя бы историю и список комнат.

Методы:

```text
func (c *Client) ListRooms(ctx) ([]RoomInfo, error)
func (c *Client) CreateRoom(ctx, name string, roomType RoomType) (RoomInfo, error)
func (c *Client) CreateDirectRoom(ctx, peerUserID int64) (RoomInfo, error)
func (c *Client) GetHistory(ctx, roomID int64, limit int, before *int64) (HistoryChunk, error)
```

**Общие правила:**

* Все запросы идут на `cfg.RESTBaseURL`.
* Если `cfg.Token != ""`, в заголовок: `Authorization: Bearer <token>`.
* Ошибки HTTP 4xx/5xx:

  * попытаться распарсить тело как `{ "error": { "code": "...", "msg": "..." } }`,
  * если удалось → `WirechatError` с `Code = error.code`.
  * если нет → generic `WirechatError{Code:ErrInternalError}`.

---

#### ListRooms

* `GET /api/rooms`
* Возвращает список всех комнат, доступных пользователю (см. протокол).

#### CreateRoom

* `POST /api/rooms` с JSON `{ "name": "...", "type": "public"|"private" }`.
* Возвращает `RoomInfo`.

#### CreateDirectRoom

* `POST /api/rooms/direct` с `{ "user_id": <peer> }`.
* Возвращает `RoomInfo` direct-комнаты.
* Идемпотентен.

#### GetHistory

* `GET /api/rooms/:id/messages?limit=&before=`.
* Возвращает `HistoryChunk` с `Messages` и `HasMore`.

---

### 3.5. Регистрация обработчиков событий

Все обработчики должны быть потокобезопасны с точки зрения регистрации и вызова (SDK сам решает, синхронно или через очередь вызывать callbacks).

```text
func (c *Client) OnMessage(fn func(MessageEvent))
func (c *Client) OnHistory(fn func(HistoryEvent))
func (c *Client) OnUserJoined(fn func(UserEvent))
func (c *Client) OnUserLeft(fn func(UserEvent))
func (c *Client) OnError(fn func(error))
func (c *Client) OnStateChanged(fn func(StateEvent))
```

**Поведение:**

* Регистрация перезаписывает предыдущий обработчик (простой сценарий).
* SDK НЕ должен паниковать/кидать исключения, если callback кидает/паникует – надо ловить и логировать.
* `OnError` вызывается:

  * при `type:"error"` от сервера (протокольные ошибки),
  * при сетевых/JSON/других внутренних ошибках.
* `OnStateChanged` вызывается на каждый переход `ConnectionState`.

---

## 4. Авто-переподключение

### 4.1. Когда срабатывает

Авто-reconnect включается, если:

* `cfg.AutoReconnect == true`,
* соединение разорвано **не** через явный `Close()`:

  * network error,
  * idle timeout с стороны сервера,
  * `ping/pong` failure,
  * `ctx` в connect/read-loop отменён извне (опционально).

### 4.2. Алгоритм

Псевдокод:

```text
onConnectionLost(err):
    lastError = err
    emit OnStateChanged(StateEvent{State: Error, Error: err})

    if !cfg.AutoReconnect:
        return

    delay = cfg.ReconnectMinDelay
    retries = 0

    while !ctx.Done():
        if cfg.MaxReconnectRetries > 0 && retries >= cfg.MaxReconnectRetries:
            break

        sleep(delay)
        retries++

        emit OnStateChanged(StateEvent{State: Connecting, Error: err})

        err = tryReconnect()
        if err == nil:
            // успешный reconnect
            emit OnStateChanged(StateEvent{State: Connected, Error: nil})
            // переподключить комнаты
            rejoinAllRooms()
            return

        // увеличиваем delay
        delay = min(delay * 2, cfg.ReconnectMaxDelay)

    // если сюда дошли — считаем, что переподключиться не удалось
    emit OnStateChanged(StateEvent{State: Error, Error: lastError})
```

**`tryReconnect()`** делает то же, что `Connect`, но:

* не создаёт новый `Client`, а переоткрывает соединение внутри текущего экземпляра,
* повторно отправляет `hello`,
* не трогает зарегистрированные callbacks,
* не очищает `joinedRooms`.

**`rejoinAllRooms()`**:

* берёт snapshot комнат, которые были успешно `Join` до потери связи,
* по очереди делает `join` (без публичного API, внутренние вызовы),
* ошибки логирует и кидает в `OnError`, но сам reconnect считается состоявшимся.

### 4.3. Поведение публичных методов во время reconnect

Пока клиент в состоянии `Connecting` (после потери связи):

* `Join/Leave/SendMessage` могут:

  * либо немедленно возвращать `WirechatError` (“not connected”),
  * либо ставить команды в очередь, чтобы отправить после успешного reconnect.
* Для v1 минималистичный вариант:

  * **не** буферизовать,
  * возвращать ошибку `WirechatError{Code:null, Temporary:true}`.

Это надо зафиксировать в доке SDK: “в период reconnect операции записи могут завершаться ошибкой; приложение может само решать, делать ли retry”.

---

## 5. Threading / Concurrency модель

### 5.1. Гарантии

* `Connect` и `Close` – **не** потокобезопасны, должны вызываться из одного потока/горoutines/таска.
* `Join`, `Leave`, `SendMessage` – **должны** быть потокобезопасны.
* Callbacks (`OnMessage`, `OnError`, etc.) могут вызываться:

  * либо напрямую в read-loop потоке (Go – отдельная goroutine),
  * либо через очередь событий в основном потоке (например, JS).
* SDK обязан сериализовать обработку входящих JSON-сообщений (чтобы два события не параллелились в хаотичном порядке, если это не оговорено отдельно).

### 5.2. Рекомендованный паттерн

* В языках с явными потоками (Go/Rust/C):

  * read-loop в отдельном потоке/горутине,
  * write операции через mutex/lock вокруг WebSocket.
* В языках с event-loop (JS/Python asyncio):

  * read/write – await’ящие таски в том же event loop,
  * callbacks всегда вызываются в контексте этого loop.

---

## 6. Logging

Интерфейс `Logger` (языкоспецифичен, но с общей идеей):

```text
type Logger interface {
    Debug(msg string, fields map[string]any)
    Info(msg string, fields map[string]any)
    Warn(msg string, fields map[string]any)
    Error(msg string, fields map[string]any)
}
```

* По умолчанию – no-op logger.
* SDK логирует:

  * попытки подключения/переподключения,
  * смену состояний,
  * ошибки протокола/сети,
  * необычные ситуации (unknown event type, malformed message).

---

## 7. Расширения под Iteration 6 (опциональный раздел)

Под будущие фичи (presence, typing, read receipts) SDK v1 можно сразу заложить “extension points”, не меняя core API:

### 7.1. Дополнительные события

* `OnPresence(fn func(PresenceEvent))`
* `OnTyping(fn func(TypingEvent))`
* `OnRead(fn func(ReadReceiptEvent))`

Типы:

```text
type PresenceStatus string
const (
    PresenceOnline  PresenceStatus = "online"
    PresenceOffline PresenceStatus = "offline"
    PresenceAway    PresenceStatus = "away"
)

type PresenceEvent struct {
    User   string
    Status PresenceStatus
    Room   string?    // для room-scoped presence
}

type TypingEvent struct {
    Room     string
    User     string
    IsTyping bool
}

type ReadReceiptEvent struct {
    Room       string
    User       string
    MessageID  int64
}
```

### 7.2. Инбаунд команды

* `SendTyping(room, isTyping)`
* `MarkRead(room, messageID)`

Всё это можно добавить **сверху** без ломания основного API.

---

## 8. Краткий usage-паттерн (идеальный happy-path)

1. Создать `Config`:

```text
cfg := DefaultConfig()
cfg.WSURL = "ws://localhost:8080/ws"
cfg.RESTBaseURL = "http://localhost:8080/api"
cfg.Token = "<jwt>"
cfg.AutoReconnect = true
```

2. Создать `Client` и повесить обработчики:

```text
client := NewClient(cfg)

client.OnMessage(func(ev MessageEvent) { /* ... */ })
client.OnHistory(func(ev HistoryEvent) { /* ... */ })
client.OnUserJoined(func(ev UserEvent) { /* ... */ })
client.OnUserLeft(func(ev UserEvent) { /* ... */ })
client.OnError(func(err error) { /* ... */ })
client.OnStateChanged(func(ev StateEvent) { /* ... */ })
```

3. Подключиться:

```text
err := client.Connect(ctx)
if err != nil { /* обработать ошибку */ }
```

4. (Опционально) через REST получить список комнат, создать нужные и т.п.

5. Подписаться на комнату и отправлять сообщения:

```text
client.Join(ctx, "general")
client.SendMessage(ctx, "general", "Hello, WireChat!")
```

6. По событию OS-сигнала / shutdown – вызвать `client.Close()`.

---

Эта спецификация покрывает:

* весь WebSocket протокол v1 (hello/join/leave/msg/history/error),
* REST-часть для комнат и истории,
* общую модель ошибок и авто-переподключения,
* единый набор типов и событий для всех языков.
