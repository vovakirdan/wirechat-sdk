# Тегирование версий Go SDK

## Доступные команды

```bash
# Посмотреть текущую версию
make tag-version

# Посмотреть все теги
make tag-list

# Создать тег v0.1.0 (по умолчанию)
make tag-create

# Создать тег другой версии
make tag-create VERSION=v0.2.0

# Отправить тег на GitHub
make tag-push

# Отправить другую версию
make tag-push VERSION=v0.2.0

# Создать и отправить тег одной командой
make tag

# Создать и отправить другую версию
make tag VERSION=v0.2.0

# Удалить тег (локально и на GitHub)
make tag-delete VERSION=v0.1.0
```

## Быстрый старт

### Первый релиз (v0.1.0)

```bash
cd /home/zov/projects/wirechat/wirechat-sdk/wirechat-sdk-go

# 1. Убедитесь что изменения закоммичены
git status

# 2. Создайте и отправьте тег
make tag
```

### Следующие релизы

```bash
# Например, v0.2.0
make tag VERSION=v0.2.0
```

## Формат тегов для подмодуля

Для Go модуля в подпапке монорепо используется формат:
```
wirechat-sdk-go/v<версия>
```

Примеры:
- `wirechat-sdk-go/v0.1.0` - первый релиз
- `wirechat-sdk-go/v0.2.0` - обновление с новыми возможностями
- `wirechat-sdk-go/v1.0.0` - первая стабильная версия

## После создания тега

Пользователи смогут установить SDK:

```bash
# Конкретная версия
go get github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go@wirechat-sdk-go/v0.1.0

# Последняя версия
go get github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go@latest
```

В `go.mod`:
```go
require (
    github.com/vovakirdan/wirechat-sdk/wirechat-sdk-go v0.1.0
)
```

## Семантическое версионирование

- `v0.x.y` - начальная разработка (breaking changes допустимы)
- `v1.0.0` - первая стабильная версия
- `v1.1.0` - новые возможности (обратно совместимо)
- `v1.1.1` - исправление багов
- `v2.0.0` - breaking changes
