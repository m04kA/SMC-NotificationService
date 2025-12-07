# SMC-NotificationService

Микросервис для управления уведомлениями и отправки сообщений через Telegram Bot API для системы автомоек SMC.

## Быстрый старт

### Запуск сервиса

```bash
# Запустить все сервисы (app + postgres + migrate)
make docker-up

# Проверить здоровье сервиса
curl http://localhost:8085/health

# Остановить сервисы
make docker-down
```

### Проверка работы

```bash
# Health check
curl http://localhost:8085/health

# Метрики Prometheus
curl http://localhost:8085/metrics
```

## Примеры использования API

### 1. Немедленное уведомление (отправляется сразу)

**Описание**: Создаёт уведомление со статусом `pending`, которое будет отправлено processor'ом в течение 30 секунд.

```bash
curl -X POST http://localhost:8085/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d @test_data/test_immediate.json
```

**Файл `test_data/test_immediate.json`:**
```json
{
  "telegram_user_id": 764461859,
  "message_text": "Привет! Это немедленное уведомление. Должно прийти сразу!",
  "type": "promo"
}
```

**Ответ:**
```json
{
  "id": 1,
  "telegram_user_id": 764461859,
  "message_text": "Привет! Это немедленное уведомление. Должно прийти сразу!",
  "type": "promo",
  "status": "pending",
  "retry_count": 0,
  "created_at": "2025-12-07T14:15:25.48927Z",
  "updated_at": "2025-12-07T14:15:25.48927Z"
}
```

### 2. Отложенное уведомление (отправляется в указанное время)

**Описание**: Создаёт уведомление со статусом `scheduled`, которое будет отправлено scheduler'ом точно в указанное время.

```bash
curl -X POST http://localhost:8085/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d @test_data/test_scheduled.json
```

**Файл `test_data/test_scheduled.json`:**
```json
{
  "telegram_user_id": 764461859,
  "message_text": "Привет! Это отложенное уведомление через 1 минуту.",
  "type": "promo",
  "scheduled_for": "2025-12-07T14:17:00Z"
}
```

**Примечание**: Время должно быть в формате ISO 8601 в UTC. Пример генерации времени на 1 минуту вперёд:
```bash
# macOS
date -u -v+1M +"%Y-%m-%dT%H:%M:%SZ"

# Linux
date -u -d "+1 minute" +"%Y-%m-%dT%H:%M:%SZ"
```

**Ответ:**
```json
{
  "id": 2,
  "telegram_user_id": 764461859,
  "message_text": "Привет! Это отложенное уведомление через 1 минуту.",
  "type": "promo",
  "status": "scheduled",
  "scheduled_for": "2025-12-07T14:17:00Z",
  "retry_count": 0,
  "created_at": "2025-12-07T14:14:21.44255Z",
  "updated_at": "2025-12-07T14:14:21.44255Z"
}
```

### 3. Уведомление с картинкой и кнопками

```bash
curl -X POST http://localhost:8085/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "telegram_user_id": 764461859,
    "message_text": "Специальное предложение! Скидка 20% на мойку.",
    "type": "promo",
    "image_urls": ["https://example.com/promo.jpg"],
    "inline_buttons": [
      {
        "text": "Записаться сейчас",
        "url": "https://auto-theme-chro.vercel.app/"
      }
    ]
  }'
```

### 4. Массовая рассылка

```bash
curl -X POST http://localhost:8085/api/v1/notifications/batch \
  -H "Content-Type: application/json" \
  -d '{
    "telegram_user_ids": [764461859, 986571288],
    "message_text": "Важное объявление для всех пользователей!",
    "type": "promo"
  }'
```

### 5. Получить список уведомлений

```bash
# Все уведомления
curl http://localhost:8085/api/v1/notifications

# С фильтрацией по статусу
curl "http://localhost:8085/api/v1/notifications?status=scheduled"

# С фильтрацией по пользователю
curl "http://localhost:8085/api/v1/notifications?telegram_user_id=764461859"

# С пагинацией
curl "http://localhost:8085/api/v1/notifications?page=1&limit=20"
```

### 6. Отменить отложенное уведомление

```bash
# Отменить одно уведомление
curl -X DELETE http://localhost:8085/api/v1/notifications/2

# Отменить всю массовую рассылку
curl -X DELETE http://localhost:8085/api/v1/notifications/batch/{span_id}
```

## Типы уведомлений

Поле `type` может принимать следующие значения:

- `welcome` - Приветственное сообщение
- `booking_created` - Бронирование создано
- `booking_confirmed` - Бронирование подтверждено
- `booking_reminder` - Напоминание о бронировании
- `booking_cancelled` - Бронирование отменено
- `promo` - Промо-сообщение

## Статусы уведомлений

- `pending` - Ожидает отправки (обрабатывается processor каждые 30 секунд)
- `scheduled` - Запланировано (будет отправлено scheduler в указанное время)
- `sent` - Успешно отправлено
- `failed` - Ошибка при отправке
- `cancelled` - Отменено

## Telegram Bot

### Приветственное сообщение (/start)

При отправке команды `/start` боту:
1. Пользователь получает приветственное сообщение с WebApp кнопкой
2. Если пользователя нет в UserService - он автоматически создаётся с ролью `client`
3. Кнопка открывает веб-приложение внутри Telegram

### Настройка Telegram Bot

1. Получите токен бота у [@BotFather](https://t.me/BotFather)
2. Добавьте токен в `.env`:
   ```bash
   TELEGRAM_BOT_TOKEN=your_bot_token_here
   ```
3. Перезапустите сервис:
   ```bash
   docker-compose restart app
   ```

## Архитектура

### Компоненты

- **HTTP Server** - REST API на порту 8085
- **Processor** - Обрабатывает немедленные уведомления (status='pending') каждые 30 секунд
- **Scheduler** - Отправляет отложенные уведомления (status='scheduled') точно в указанное время
- **Polling Handler** - Обрабатывает входящие команды от Telegram (Long Polling)
- **PostgreSQL** - Хранилище уведомлений

### Workflow

**Немедленное уведомление:**
```
API Request → Create (status=pending) → Processor (каждые 30s) → Telegram API → Update (status=sent)
```

**Отложенное уведомление:**
```
API Request → Create (status=scheduled) → Scheduler (точно в scheduled_for) → Telegram API → Update (status=sent)
```

**Команда /start:**
```
Telegram → Long Polling → Polling Handler → Start Message UseCase → UserService + Telegram API
```

## Мониторинг

### Prometheus метрики

Доступны на `http://localhost:8085/metrics`:

- `http_requests_total` - Количество HTTP запросов
- `http_request_duration_seconds` - Длительность запросов
- `db_queries_total` - Количество запросов к БД
- `db_query_duration_seconds` - Время выполнения SQL запросов
- `db_connections_active` - Активные соединения к БД

### Логи

```bash
# Просмотр логов приложения
make docker-logs-app

# Просмотр логов базы данных
make docker-logs-db

# Следить за логами в реальном времени
docker logs notification-service-app -f
```

## Разработка

### Структура проекта

```
.
├── cmd/main.go                    # Точка входа
├── internal/
│   ├── api/handlers/              # HTTP handlers
│   ├── domain/                    # Доменные модели
│   ├── infra/storage/             # Репозитории (БД)
│   ├── integrations/              # Внешние сервисы (UserService)
│   ├── service/                   # Бизнес-логика
│   ├── usecase/                   # Use cases
│   └── worker/                    # Background workers
├── migrations/                    # SQL миграции
├── test_data/                     # Тестовые данные
└── config.toml                    # Конфигурация
```

### Команды Make

```bash
make docker-up          # Запустить все сервисы
make docker-down        # Остановить все сервисы
make docker-build       # Собрать Docker образ
make docker-restart     # Пересобрать и перезапустить
make docker-logs-app    # Логи приложения
make migrate-up         # Применить миграции
make migrate-down       # Откатить миграции
```

## Интеграция с другими сервисами

### UserService

NotificationService интегрируется с UserService для:
- Валидации `telegram_user_id` при создании уведомления
- Автоматического создания пользователей при команде `/start`

**URL**: `http://localhost:8080` (локально) или `http://host.docker.internal:8080` (из Docker)

### BookingService

BookingService может вызывать NotificationService для отправки уведомлений:
- При создании бронирования
- При подтверждении бронирования
- Напоминание за 1 час до визита
- При отмене бронирования

## Устранение неполадок

### Ошибка "connection refused" к UserService

Если в логах видите:
```
dial tcp 127.0.0.1:8080: connect: connection refused
```

Проверьте:
1. UserService запущен
2. В `config.toml` указан правильный URL: `http://host.docker.internal:8080` (для Docker)
3. Перезапустите сервис: `make docker-restart`

### Отложенные уведомления не отправляются

Проверьте логи на наличие:
```
[INFO] Starting notification scheduler
[INFO] Scheduled notification X for ...
```

Если нет - проверьте, что scheduler запущен в `cmd/main.go`.

### Telegram Bot не отвечает

1. Проверьте токен в `.env`
2. Проверьте логи: `docker logs notification-service-app -f`
3. Убедитесь, что Long Polling запущен: `[INFO] Using Long Polling mode`

## Лицензия

MIT
