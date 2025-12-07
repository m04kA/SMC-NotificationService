# SMC-NotificationService

Микросервис для управления уведомлениями и отправки сообщений через Telegram Bot API для системы автомоек SMC.

## Обзор проекта

**Назначение:** Централизованный сервис для отправки уведомлений пользователям и менеджерам через Telegram.

**Порт:** 8085
**База данных:** PostgreSQL (порт 5440, БД: smc_notificationservice)

### Ключевые возможности:
- ✅ Приветственные сообщения при /start с WebApp кнопками
- ✅ Очередь немедленных уведомлений (pending)
- ✅ Планирование отложенных уведомлений через gocron
- ✅ Массовые рассылки (batch notifications) с группировкой по span_id
- ✅ Отмена одиночных и групповых уведомлений
- ✅ Отправка текста, фото и медиагрупп (до 10 изображений)
- ✅ Множественные inline-кнопки (JSONB массив)
- ✅ Интеграция с UserService (валидация + автосоздание пользователей)
- ✅ Long Polling режим для обработки /start команды
- ✅ Метрики Prometheus
- ✅ Персистентность отложенных задач (не теряем при перезапуске)

---

## Диаграмма архитектуры сервиса

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SMC-NotificationService                              │
│                              (Port 8085)                                      │
│                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                          HTTP Server (Gorilla Mux)                   │   │
│  │                                                                       │   │
│  │  PUBLIC ENDPOINTS:                                                   │   │
│  │  • GET  /health               → Health Check                         │   │
│  │  • GET  /metrics              → Prometheus Metrics                   │   │
│  │  • POST /webhook/telegram     → Telegram Bot Webhook (/start)        │   │
│  │                                                                       │   │
│  │  API v1 ENDPOINTS:                                                   │   │
│  │  • POST   /api/v1/notifications              → Create Notification  │   │
│  │  • POST   /api/v1/notifications/batch        → Batch Notification   │   │
│  │  • GET    /api/v1/notifications              → List Notifications   │   │
│  │  • DELETE /api/v1/notifications/:id          → Cancel Single        │   │
│  │  • DELETE /api/v1/notifications/batch/:span  → Cancel Batch         │   │
│  └────────────────────────┬──────────────────────────────────────────────┘   │
│                           │                                                   │
│  ┌────────────────────────▼──────────────────────────────────────────────┐   │
│  │                         Handler Layer                                  │   │
│  │  • create_notification/  • create_batch_notification/                 │   │
│  │  • list_notifications/  • cancel_notification/                       │   │
│  │  • cancel_batch_notification/  • telegram_webhook/  • health/        │   │
│  └────────────────────────┬──────────────────────────────────────────────┘   │
│                           │                                                   │
│  ┌────────────────────────▼──────────────────────────────────────────────┐   │
│  │                        Service Layer                                   │   │
│  │                                                                         │   │
│  │  ┌─────────────────────┐          ┌─────────────────────┐             │   │
│  │  │ Notifications Svc   │          │   Telegram Svc      │             │   │
│  │  │                     │          │                     │             │   │
│  │  │ • Create()          │          │ • SendNotification()│             │   │
│  │  │ • CreateBatch()     │          │ • SendWelcome()     │             │   │
│  │  │ • GetByID()         │          │ • SetWebhook()      │             │   │
│  │  │ • GetBySpanID()     │          │ • DeleteWebhook()   │             │   │
│  │  │ • List()            │          │ • GetUpdatesChan()  │             │   │
│  │  │ • Cancel()          │          │                     │             │   │
│  │  │ • CancelBySpanID()  │          │                     │             │   │
│  │  └──────────┬──────────┘          └──────────┬──────────┘             │   │
│  └─────────────┼────────────────────────────────┼────────────────────────┘   │
│                │                                 │                            │
│  ┌─────────────▼─────────────────────────────────▼────────────────────────┐  │
│  │                      Integration Layer                                  │  │
│  │                                                                          │  │
│  │  ┌──────────────────────┐                                               │  │
│  │  │  UserService Client  │  → GET /internal/users/{tg_user_id}           │  │
│  │  │                      │     POST /users (create on /start)            │  │
│  │  └──────────────────────┘                                               │  │
│  └──────────────────────────────────────────────────────────────────────────┘  │
│                                                                               │
│  ┌───────────────────────────────────────────────────────────────────────┐   │
│  │                        Repository Layer                                │   │
│  │                                                                         │   │
│  │  NotificationRepository:                                                │   │
│  │  • Create()  • CreateBatch()  • GetByID()  • GetBySpanID()            │   │
│  │  • List()  • GetPending()  • GetScheduled()  • UpdateStatus()         │   │
│  │  • MarkAsSent()  • Cancel()  • CancelBySpanID()  • IncrementRetry()   │   │
│  └────────────────────────┬────────────────────────────────────────────────┘   │
│                           │                                                   │
│  ┌────────────────────────▼──────────────────────────────────────────────┐   │
│  │                     PostgreSQL Database                                │   │
│  │                    (Port 5440, smc_notificationservice)                │   │
│  │                                                                         │   │
│  │  TABLE: notifications                                                  │   │
│  │  • id, telegram_user_id, chat_id, span_id, message_text               │   │
│  │  • image_urls (TEXT[]), inline_buttons (JSONB)                        │   │
│  │  • notification_type, status, scheduled_for, sent_at                  │   │
│  │  • metadata (JSONB), error_message, retry_count                       │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                               │
│  ┌───────────────────────────────────────────────────────────────────────┐   │
│  │                         Worker Layer                                   │   │
│  │                      (Background Goroutines)                           │   │
│  │                                                                         │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐           │   │
│  │  │  Scheduler   │  │  Processor   │  │  Polling Handler     │           │   │
│  │  │  (gocron)    │  │  (queue)     │  │  (Telegram updates)  │           │   │
│  │  │              │  │              │  │                      │           │   │
│  │  │ • Start()    │  │ • Start()    │  │ • Start()            │           │   │
│  │  │ • LoadDB()   │  │ • Poll 30s   │  │ • Handle /start      │           │   │
│  │  │ • Schedule() │  │ • Batch 50   │  │ • Use start_message  │           │   │
│  │  │ • Cancel()   │  │ • Send       │  │   usecase            │           │   │
│  │  │ • Stop()     │  │ • Stop()     │  │ • Stop()             │           │   │
│  │  └──────┬───────┘  └──────┬───────┘  └───────────┬──────────┘           │   │
│  │         │                 │                       │                      │   │
│  │         └─────────────────┴───────────────────────┘                      │   │
│  │                             │                                            │   │
│  │                             ▼                                            │   │
│  │                   Telegram Bot API                                       │   │
│  │              https://api.telegram.org/bot{TOKEN}/                        │   │
│  └───────────────────────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────────────────────────┘

                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
                    ▼               ▼               ▼
            ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
            │ BookingSvc   │ │  UserService │ │   Telegram   │
            │ :8083        │ │    :8080     │ │     Bot      │
            │              │ │              │ │              │
            │ Calls POST   │ │ Validates    │ │ Receives     │
            │ /notifications│ │  users via   │ │  messages    │
            │              │ │ /internal/*  │ │              │
            └──────────────┘ └──────────────┘ └──────────────┘

ПОТОК ДАННЫХ:

1. НЕМЕДЛЕННОЕ УВЕДОМЛЕНИЕ:
   BookingService → POST /api/v1/notifications → Handler → Service
   → Repository (status='pending') → Processor (polling) → Telegram API

2. ОТЛОЖЕННОЕ УВЕДОМЛЕНИЕ:
   BookingService → POST /api/v1/notifications → Handler → Service
   → Repository (status='scheduled') → Scheduler (gocron) → Telegram API (в назначенное время)

3. ПРИВЕТСТВИЕ (/start):
   Telegram → POST /webhook/telegram → Handler → Telegram Service
   → Telegram API (sendMessage with inline button)

4. ОТМЕНА УВЕДОМЛЕНИЯ:
   Client → DELETE /api/v1/notifications/:id → Handler → Service
   → Repository (status='cancelled') → Scheduler.Cancel() (remove from gocron)
```

---

## Архитектура

### Технологический стек

- **Language**: Go 1.25.3+
- **Architecture**: Clean Architecture (Domain, Service, Repository, Handlers, Worker)
- **Database**: PostgreSQL 16 + database/sql
- **HTTP Router**: Gorilla Mux
- **Query Builder**: Squirrel (psqlbuilder wrapper)
- **Scheduler**: gocron (для отложенных уведомлений)
- **Telegram**: go-telegram-bot-api/v5 (обертка над Bot API)
- **Monitoring**: Prometheus + Grafana
- **Logging**: Custom logger (console + file)
- **Containerization**: Docker Compose

### Архитектурные решения

1. **Хранилище очереди**: PostgreSQL (миграция на message broker позже)
2. **Worker**: В том же процессе (scheduler + processor как goroutines)
3. **Планировщик**: gocron с загрузкой из БД при старте
4. **Интеграция**: Прямые HTTP вызовы от BookingService/UserService
5. **Персистентность**: Отложенные задачи восстанавливаются из БД при перезапуске

---

## База данных

### Таблица: notifications

```sql
CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,

    -- Получатель (одно из двух обязательно)
    telegram_user_id BIGINT,           -- Telegram user ID (личное сообщение)
    chat_id BIGINT,                     -- Chat ID (группа/канал)

    -- Группировка массовых рассылок
    span_id UUID,                       -- UUID для группировки массовых рассылок

    -- Контент
    message_text TEXT NOT NULL,
    image_urls TEXT[],                  -- Массив URL изображений (до 10 для MediaGroup)
    inline_buttons JSONB,               -- Массив inline-кнопок в JSON формате

    -- Классификация
    notification_type notification_type NOT NULL,  -- ENUM

    -- Статус и планирование
    status notification_status NOT NULL DEFAULT 'pending',  -- ENUM
    scheduled_for TIMESTAMP,            -- Время отправки (для отложенных)
    sent_at TIMESTAMP,                  -- Фактическое время отправки

    -- Метаданные и ошибки
    metadata JSONB DEFAULT '{}'::jsonb, -- booking_id, company_id, etc.
    error_message TEXT,
    retry_count INT DEFAULT 0,

    -- Аудит
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Ограничения
    CONSTRAINT chk_recipient CHECK (
        telegram_user_id IS NOT NULL OR chat_id IS NOT NULL
    ),
    CONSTRAINT chk_scheduled CHECK (
        (status = 'scheduled' AND scheduled_for IS NOT NULL) OR
        (status != 'scheduled')
    ),
    CONSTRAINT chk_media_group_limit CHECK (
        image_urls IS NULL OR array_length(image_urls, 1) <= 10
    )
);
```

**ENUM типы:**
```sql
-- notification_type
'welcome', 'booking_created', 'booking_confirmed',
'booking_reminder', 'booking_cancelled', 'promo'

-- notification_status
'pending', 'scheduled', 'sent', 'failed', 'cancelled'
```

**Индексы:**
- `idx_notifications_pending ON created_at` WHERE status = 'pending' - для processor (немедленные)
- `idx_notifications_scheduled ON scheduled_for` WHERE status = 'scheduled' - для scheduler (отложенные)
- `idx_notifications_list ON (status, created_at DESC)` - для API листинга

---

## Clean Architecture Layers

### 1. Domain Layer (`internal/domain/`)

**notification.go** - Доменные модели:
```go
type Notification struct {
    ID              int64
    TelegramUserID  *int64
    ChatID          *int64
    SpanID          *string             // UUID для группировки массовых рассылок
    MessageText     string
    ImageURLs       pq.StringArray      // Массив URL изображений (до 10)
    InlineButtons   InlineButtons       // Массив inline-кнопок
    Type            NotificationType
    Status          NotificationStatus
    ScheduledFor    *time.Time
    SentAt          *time.Time
    Metadata        Metadata            // JSONB поле
    ErrorMessage    *string
    RetryCount      int
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type InlineButton struct {
    Text string `json:"text"` // Текст кнопки
    URL  string `json:"url"`  // URL для перехода
}

type InlineButtons []InlineButton  // Реализует driver.Valuer и sql.Scanner

type Metadata map[string]interface{}  // Реализует driver.Valuer и sql.Scanner
```

**Методы Notification:**
- `GetChatID()` - возвращает chat_id с приоритетом (chat_id > telegram_user_id)
- `IsScheduled()` - проверяет, отложенное ли уведомление
- `IsPending()` - проверяет, ожидает ли отправки
- `CanBeCancelled()` - можно ли отменить
- `HasImages()` - есть ли изображения
- `HasButtons()` - есть ли кнопки
- `IsMediaGroup()` - нужно ли отправлять как MediaGroup (2+ изображения)

### 2. Repository Layer (`internal/infra/storage/notification/`)

**repository.go** - Операции с БД:
- `Create()` - создать одно уведомление
- `CreateBatch()` - массовая вставка уведомлений
- `GetByID()` - получить по ID
- `GetBySpanID()` - получить все уведомления массовой рассылки
- `List()` - список с фильтрацией и пагинацией
- `GetPendingNotifications()` - для processor (немедленные)
- `GetScheduledNotifications()` - для scheduler (отложенные)
- `UpdateStatus()` - обновить статус
- `MarkAsSent()` - пометить как отправленное
- `Cancel()` - отменить одно запланированное
- `CancelBySpanID()` - отменить группу уведомлений
- `IncrementRetryCount()` - увеличить счётчик попыток

### 3. Service Layer

#### Notifications Service (`internal/service/notifications/`)

**service.go** - Бизнес-логика:
- `Create()` - создание одного уведомления с валидацией пользователя
- `CreateBatch()` - создание массовой рассылки (множество получателей)
- `GetByID()` - получение уведомления по ID
- `GetBySpanID()` - получение всех уведомлений массовой рассылки
- `List()` - список с фильтрами (status, type, user_id, span_id)
- `Cancel()` - отмена одного запланированного
- `CancelBySpanID()` - отмена всех уведомлений массовой рассылки

**models/models.go** - DTOs:
```go
type CreateNotificationRequest struct {
    TelegramUserID   *int64
    ChatID           *int64
    MessageText      string
    ImageURL         *string
    ButtonText       *string
    ButtonURL        *string
    NotificationType string
    ScheduledFor     *time.Time
    Metadata         map[string]interface{}
}
```

#### Telegram Service (`internal/service/telegram/`)

**service.go** - Работа с Telegram Bot API:
- `SendNotification()` - универсальная отправка (текст/фото/медиагруппа + кнопки)
- `SendWelcomeMessage()` - приветствие при /start с WebApp кнопкой
- `SetWebhook()` - установка webhook для production
- `DeleteWebhook()` - удаление webhook
- `GetUpdatesChan()` - получение канала обновлений для Long Polling

### 4. Integration Layer (`internal/integrations/userservice/`)

**client.go** - HTTP клиент для UserService:
- `GetUser()` - получение пользователя по telegram_user_id
- `CreateUser()` - создание пользователя (для /start команды)

### 5. Worker Layer (`internal/worker/`)

#### Scheduler (`scheduler.go`)

**Ответственность:** Управление отложенными уведомлениями через gocron

**Ключевые методы:**
- `LoadScheduledNotifications()` - загружает все scheduled из БД при старте
- `ScheduleNotification()` - планирует задачу на отправку
- `CancelNotification()` - отменяет запланированную задачу
- `Stop()` - корректная остановка scheduler

**Важно:** При старте приложения загружаем все `status='scheduled'` из БД, чтобы не потерять отложенные задачи при перезапуске!

#### Processor (`processor.go`)

**Ответственность:** Обработка немедленных уведомлений (polling)

**Логика:**
- Каждые 30 секунд проверяет `status='pending'` в БД
- Берёт батч до 50 уведомлений
- Отправляет через Telegram Service
- Обновляет статусы (sent/failed)

#### Polling Handler (`polling.go`)

**Ответственность:** Обработка входящих сообщений от Telegram в режиме Long Polling

**Логика:**
- Получает обновления через канал tgbotapi.UpdatesChannel
- Обрабатывает команду /start
- Использует start_message usecase для отправки приветствия и создания пользователя
- Работает до отмены контекста

### 6. UseCase Layer (`internal/usecase/`)

#### Start Message UseCase (`start_message/`)

**Ответственность:** Централизованная обработка команды /start

**Логика:**
- Отправляет приветственное сообщение с WebApp кнопкой
- Проверяет существование пользователя в UserService
- Создаёт пользователя если не существует
- Чистое оборачивание ошибок без логирования (логирование в handlers)

### 7. Handler Layer (`internal/api/handlers/`)

**Структура:**
```
handlers/
├── create_notification/        # POST /api/v1/notifications
│   ├── handler.go
│   ├── contract.go
│   └── models/models.go
├── create_batch_notification/  # POST /api/v1/notifications/batch
│   ├── handler.go
│   ├── contract.go
│   └── models/models.go
├── list_notifications/         # GET /api/v1/notifications
│   ├── handler.go
│   └── contract.go
├── cancel_notification/        # DELETE /api/v1/notifications/{id}
│   ├── handler.go
│   └── contract.go
├── cancel_batch_notification/  # DELETE /api/v1/notifications/batch/{span_id}
│   ├── handler.go
│   └── contract.go
├── telegram_webhook/           # POST /webhook/telegram
│   ├── handler.go
│   └── contract.go
└── health/                     # GET /health
    └── handler.go
```

---

## API Endpoints

### Публичные endpoints

```
GET  /health              - Health check
GET  /metrics             - Prometheus метрики
POST /webhook/telegram    - Webhook для обработки /start команды
```

### API v1 endpoints

```
POST   /api/v1/notifications              - Создать одно уведомление
POST   /api/v1/notifications/batch        - Создать массовую рассылку
GET    /api/v1/notifications              - Список уведомлений (фильтры: status, type, user_id, span_id, page, limit)
DELETE /api/v1/notifications/{id}         - Отменить одно отложенное уведомление
DELETE /api/v1/notifications/batch/{span_id} - Отменить группу уведомлений
```

### Примеры запросов

#### 1. Создание немедленного уведомления

```bash
curl -X POST http://localhost:8085/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "telegram_user_id": 123456789,
    "message_text": "Ваша запись подтверждена на 15.12.2024 в 10:00",
    "button_text": "Отменить запись",
    "button_url": "https://app.com/bookings/123/cancel",
    "notification_type": "booking_confirmed",
    "metadata": {
      "booking_id": 123,
      "company_id": 456
    }
  }'
```

**Ответ:**
```json
{
  "id": 1,
  "telegram_user_id": 123456789,
  "message_text": "Ваша запись подтверждена...",
  "button_text": "Отменить запись",
  "button_url": "https://app.com/bookings/123/cancel",
  "notification_type": "booking_confirmed",
  "status": "pending",
  "metadata": {"booking_id": 123, "company_id": 456},
  "created_at": "2024-12-15T08:00:00Z"
}
```

#### 2. Создание отложенного уведомления (напоминание)

```bash
curl -X POST http://localhost:8085/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "telegram_user_id": 123456789,
    "message_text": "Напоминание: через 1 час ваша запись на автомойку",
    "notification_type": "booking_reminder",
    "scheduled_for": "2024-12-15T09:00:00Z",
    "metadata": {
      "booking_id": 123
    }
  }'
```

**Ответ:**
```json
{
  "id": 2,
  "telegram_user_id": 123456789,
  "message_text": "Напоминание: через 1 час...",
  "notification_type": "booking_reminder",
  "status": "scheduled",
  "scheduled_for": "2024-12-15T09:00:00Z",
  "metadata": {"booking_id": 123},
  "created_at": "2024-12-14T12:00:00Z"
}
```

#### 3. Список уведомлений с фильтрацией

```bash
curl "http://localhost:8085/api/v1/notifications?status=scheduled&telegram_user_id=123456789&page=1&limit=20"
```

#### 4. Отмена отложенного уведомления

```bash
curl -X DELETE http://localhost:8085/api/v1/notifications/2
```

---

## Интеграция с другими сервисами

### UserService (http://localhost:8080)

**Используется для:**
- Валидация `telegram_user_id` при создании уведомления
- Проверка существования пользователя

**Endpoint:**
```
GET /internal/users/{tg_user_id}
```

### BookingService (http://localhost:8083)

**Вызывает NotificationService при:**
- Создании бронирования → уведомление менеджеру
- Подтверждении бронирования → уведомление клиенту
- За 1 час до записи → напоминание клиенту
- Отмене бронирования → уведомление обеим сторонам

**Пример вызова из BookingService:**
```go
// После создания бронирования
notificationClient.CreateNotification(ctx, &NotificationRequest{
    TelegramUserID:   managerTelegramID,
    MessageText:      "Новая запись на 15.12 в 10:00",
    NotificationType: "booking_created",
    Metadata: map[string]interface{}{
        "booking_id": bookingID,
        "user_id":    userID,
    },
})
```

---

## Telegram Bot Integration

### Приветственное сообщение (/start)

**Триггер:** Пользователь отправляет `/start` боту

**Webhook URL:** `POST /webhook/telegram`

**Сообщение:**
```
Добро пожаловать!

Для удобного доступа к нашему сервису, вы можете создать иконку приложения на главном экране вашего устройства.

[Кнопка: "Создать иконку" → https://auto-theme-chro.vercel.app/]
```

**Реализация:**
- Inline-кнопка через `reply_markup` в Telegram API
- Отправка через `sendMessage` метод

### Отправка с картинкой

Если `image_url` указан, используется метод `sendPhoto`:
```json
{
  "chat_id": 123456789,
  "photo": "https://example.com/promo.jpg",
  "caption": "Акция: скидка 20% на мойку!",
  "reply_markup": {
    "inline_keyboard": [[
      {"text": "Записаться", "url": "https://app.com/booking"}
    ]]
  }
}
```

---

## Worker Lifecycle

### При старте приложения (cmd/main.go)

```go
// 1. Инициализация репозиториев и сервисов
notificationRepo := notification.NewRepository(wrappedDB)
telegramSvc := telegram.NewService(cfg.Telegram.BotToken, log)
notificationSvc := notifications.NewService(notificationRepo, userServiceClient)

// 2. Создание Worker компонентов
scheduler := worker.NewScheduler(notificationRepo, telegramSvc, log)
processor := worker.NewProcessor(notificationRepo, telegramSvc, log)

// 3. КРИТИЧНО: Загрузка отложенных задач из БД
if err := scheduler.LoadScheduledNotifications(ctx); err != nil {
    log.Error("Failed to load scheduled notifications: %v", err)
}

// 4. Запуск processor в фоне
go processor.Start(ctx)

// 5. Запуск HTTP сервера
go srv.ListenAndServe()
```

### При остановке (Graceful Shutdown)

```go
// 1. Получаем сигнал SIGINT/SIGTERM
<-quit

// 2. Останавливаем Worker ПЕРЕД сервером
processor.Stop()
scheduler.Stop()

// 3. Останавливаем метрики
close(stopMetricsCh)

// 4. Graceful shutdown HTTP сервера
srv.Shutdown(shutdownCtx)
```

**Важно:** Worker останавливается ДО сервера, чтобы завершить обработку текущих уведомлений!

---

## Конфигурация

### config.toml

```toml
# Логирование
[logs]
level = "info"
file = "./logs/app.log"

# HTTP сервер
[server]
http_port = 8085
read_timeout = 15
write_timeout = 15
idle_timeout = 60
shutdown_timeout = 10

# База данных PostgreSQL
[database]
host = "localhost"
port = 5440
user = "postgres"
password = "postgres"
dbname = "smc_notificationservice"
sslmode = "disable"
max_open_conns = 25
max_idle_conns = 5
conn_max_lifetime = 300

# Метрики Prometheus
[metrics]
enabled = true
path = "/metrics"
service_name = "notificationservice"

# Telegram Bot
[telegram]
bot_token = ""              # Переопределяется через .env
webhook_url = ""            # Опционально для production

# UserService Integration
[userservice]
url = "http://localhost:8080"
timeout = 10  # секунды
```

### .env

```bash
# Telegram Bot
TELEGRAM_BOT_TOKEN=your_bot_token_here
TELEGRAM_WEBHOOK_URL=  # опционально

# UserService
USERSERVICE_URL=http://localhost:8080
USERSERVICE_TIMEOUT=10

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=smc_notificationservice
DB_SSLMODE=disable

# Server
HTTP_PORT=8085
LOG_LEVEL=info
```

---

## Развёртывание

### Локальный запуск

```bash
# 1. Установить зависимости
go mod download

# 2. Создать .env файл
cp .env.example .env
# Добавить TELEGRAM_BOT_TOKEN

# 3. Запустить PostgreSQL
make docker-up

# 4. Применить миграции
make migrate-up

# 5. Запустить приложение
make run
```

### Docker

```bash
# Запуск всех сервисов (app + postgres + migrate)
make docker-up

# Просмотр логов
make docker-logs-app

# Остановка
make docker-down
```

### Проверка работоспособности

```bash
# Health check
curl http://localhost:8085/health

# Метрики
curl http://localhost:8085/metrics

# Создание немедленного тестового уведомления
curl -X POST http://localhost:8085/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "telegram_user_id": YOUR_TELEGRAM_ID,
    "message_text": "Тест немедленного уведомления",
    "type": "promo"
  }'

# Создание отложенного уведомления (через 1 минуту)
curl -X POST http://localhost:8085/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d @test_data/test_scheduled.json
```

---

## Мониторинг

### Prometheus метрики

**HTTP метрики:**
- `http_requests_total` - количество запросов
- `http_request_duration_seconds` - длительность запросов
- `http_errors_total` - количество ошибок

**Database метрики:**
- `db_queries_total` - количество запросов к БД
- `db_query_duration_seconds` - время выполнения запросов
- `db_connections_active` - активные соединения
- `db_errors_total` - ошибки БД

**Custom метрики (можно добавить):**
- `notifications_sent_total{type, status}` - отправленные уведомления
- `notifications_queue_size{status}` - размер очереди
- `telegram_api_errors_total` - ошибки Telegram API

### Интеграция с SMC-Monitoring

Сервис автоматически экспортирует метрики на `/metrics`.

Добавьте в Prometheus (`SMC-Monitoring/prometheus/prometheus.yml`):
```yaml
scrape_configs:
  - job_name: 'notificationservice'
    static_configs:
      - targets: ['localhost:8085']
```

---

## Критические моменты реализации

### 🔥 1. Не теряем отложенные задачи при перезапуске

```go
// В main.go обязательно:
scheduler := worker.NewScheduler(repo, telegramSvc, log)

// КРИТИЧНО: Загружаем все scheduled из БД
if err := scheduler.LoadScheduledNotifications(ctx); err != nil {
    log.Error("Failed to load scheduled notifications: %v", err)
}
```

### 🔥 2. Graceful Shutdown для Worker

```go
// Останавливаем worker ПЕРЕД HTTP сервером:
processor.Stop()   // Завершает текущий батч
scheduler.Stop()   // Останавливает gocron

// Только потом:
srv.Shutdown(shutdownCtx)
```

### 🔥 3. Определение chat_id

```go
// Приоритет: chat_id > telegram_user_id
// Для личных сообщений в Telegram: chat_id == user_id

func determineChatID(n *domain.Notification) int64 {
    if n.ChatID != nil {
        return *n.ChatID  // Группа/канал
    }
    if n.TelegramUserID != nil {
        return *n.TelegramUserID  // Личное сообщение
    }
    return 0  // Ошибка
}
```

### 🔥 4. Обработка ошибок Telegram API

```go
// При ошибке отправки:
// 1. Увеличиваем retry_count
// 2. Обновляем status = 'failed'
// 3. Сохраняем error_message в БД

if err := telegramSvc.SendMessage(ctx, msg); err != nil {
    repo.IncrementRetryCount(ctx, notificationID)
    repo.UpdateStatus(ctx, notificationID, domain.NotificationStatusFailed, &errMsg)
}
```

---

## Зависимости

```go
require (
    github.com/BurntSushi/toml v1.4.0
    github.com/Masterminds/squirrel v1.5.4
    github.com/go-co-op/gocron v1.37.0                      // Планировщик для отложенных уведомлений
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1  // Telegram Bot API wrapper
    github.com/gorilla/mux v1.8.1
    github.com/lib/pq v1.10.9
    github.com/prometheus/client_golang v1.20.5
)
```

**Установка:**
```bash
go get github.com/go-co-op/gocron@v1.37.0
go get github.com/go-telegram-bot-api/telegram-bot-api/v5@v5.5.1
go mod tidy
```

---

## Порядок реализации

### Этап 1: Инфраструктура (5 файлов)
1. `migrations/001_create_notifications_table.up.sql`
2. `migrations/001_create_notifications_table.down.sql`
3. Обновить `config.toml`
4. Обновить `.env.example`
5. Обновить `go.mod`

### Этап 2: Domain Layer (2 файла)
6. `internal/domain/notification.go`
7. `internal/domain/telegram_message.go`

### Этап 3: Repository Layer (2 файла)
8. `internal/infra/storage/notification/repository.go`
9. `internal/infra/storage/notification/errors.go`

### Этап 4: Telegram Service (3 файла)
10. `internal/service/telegram/service.go`
11. `internal/service/telegram/contract.go`
12. `internal/service/telegram/errors.go`

### Этап 5: UserService Integration (1 файл)
13. `internal/integrations/userservice/client.go`

### Этап 6: Notifications Service (4 файла)
14. `internal/service/notifications/service.go`
15. `internal/service/notifications/contracts.go`
16. `internal/service/notifications/errors.go`
17. `internal/service/notifications/models/models.go`

### Этап 7: Worker Layer (3 файла) 🔥 КРИТИЧНО
18. `internal/worker/scheduler.go`
19. `internal/worker/processor.go`
20. `internal/worker/contracts.go`

### Этап 8: Handler Layer (11 файлов)
21-26. Handlers (create, list, cancel, webhook, health)

### Этап 9: Configuration (1 файл)
27. Обновить `internal/config/config.go`

### Этап 10: Main Application (1 файл)
28. Полностью переписать `cmd/main.go`

### Этап 11: Docker & Build (3 файла)
29. `Dockerfile.notificationservice`
30. Обновить `docker-compose.yml`
31. Обновить `Makefile`

**Итого:** 31 файл (27 новых + 4 обновлённых)

---

## Паттерны Clean Architecture

### Typed Errors с wrapping

**Repository Layer:**
```go
var ErrNotificationNotFound = errors.New("repository: notification not found")

func (r *Repository) GetByID(ctx context.Context, id int64) (*domain.Notification, error) {
    if err == sql.ErrNoRows {
        return nil, ErrNotificationNotFound
    }
    return nil, fmt.Errorf("%w: %v", ErrExecQuery, err)
}
```

**Service Layer:**
```go
var ErrNotificationNotFound = errors.New("notification not found")

func (s *Service) GetByID(ctx context.Context, id int64) (*models.NotificationResponse, error) {
    notification, err := s.repo.GetByID(ctx, id)
    if errors.Is(err, notificationRepo.ErrNotificationNotFound) {
        return nil, ErrNotificationNotFound
    }
    return models.FromDomainNotification(notification), nil
}
```

**Handler Layer:**
```go
notification, err := h.service.GetByID(r.Context(), id)
if errors.Is(err, notifications.ErrNotificationNotFound) {
    handlers.RespondNotFound(w, "notification not found")
    return
}
```

### Dependency Injection через интерфейсы

**contract.go:**
```go
type NotificationService interface {
    Create(ctx context.Context, req *models.CreateNotificationRequest) (*models.NotificationResponse, error)
}

type Scheduler interface {
    ScheduleNotification(ctx context.Context, notification *domain.Notification) error
}
```

**handler.go:**
```go
type Handler struct {
    service   NotificationService  // Интерфейс
    scheduler Scheduler             // Интерфейс
    logger    Logger
}
```

---

## ✅ Статус реализации: ЗАВЕРШЕНО

**Дата завершения:** 2025-12-07

Все этапы реализации успешно завершены. Сервис полностью функционален и протестирован.

### ✅ Реализованные возможности:

1. **HTTP API** - REST API на порту 8085 с endpoints для управления уведомлениями
2. **Немедленные уведомления** - Processor обрабатывает pending уведомления каждые 30 секунд
3. **Отложенные уведомления** - Scheduler отправляет уведомления точно в запланированное время
4. **Telegram Bot Integration** - Long Polling для обработки команды /start
5. **WebApp кнопки** - Inline кнопки открывают веб-приложение внутри Telegram
6. **Auto-создание пользователей** - При /start пользователь автоматически создаётся в UserService
7. **Массовые рассылки** - Поддержка отправки уведомлений множеству пользователей
8. **Метрики Prometheus** - Сбор метрик HTTP, БД и производительности
9. **Graceful Shutdown** - Корректная остановка всех компонентов
10. **Docker Support** - Полная контейнеризация с docker-compose

### ✅ Этап 0: Планирование
- [x] Создать детальный план архитектуры
- [x] Утвердить решения с пользователем
- [x] Обновить CLAUDE.md с полным описанием

### ✅ Этап 1: Инфраструктура и БД (5 файлов)
- [x] Создать `migrations/001_create_notifications_table.up.sql`
- [x] Создать `migrations/001_create_notifications_table.down.sql`
- [x] Обновить `config.toml` (порт 8085, БД 5440, секции telegram/userservice)
- [x] Создать `.env.example` (добавить TELEGRAM_BOT_TOKEN, USERSERVICE_URL)
- [x] Обновить `go.mod` (добавить gocron v1.37.0 + telegram-bot-api v5)

### ✅ Этап 2: Domain Layer (2 файла)
- [x] Создать `internal/domain/notification.go`
- [x] Создать `internal/domain/telegram_message.go`

### ✅ Этап 3: Repository Layer (2 файла)
- [x] Создать `internal/infra/storage/notification/repository.go`
- [x] Создать `internal/infra/storage/notification/errors.go`

### ✅ Этап 4: Telegram Service (4 файла)
- [x] Создать `internal/service/telegram/service.go`
- [x] Создать `internal/service/telegram/contract.go`
- [x] Создать `internal/service/telegram/errors.go`
- [x] Создать `internal/service/telegram/templates/welcome.go`

### ✅ Этап 5: UserService Integration (2 файла)
- [x] Создать `internal/integrations/userservice/client.go`
- [x] Создать `internal/integrations/userservice/models.go`

### ✅ Этап 6: Notifications Service (4 файла)
- [x] Создать `internal/service/notifications/service.go`
- [x] Создать `internal/service/notifications/contracts.go`
- [x] Создать `internal/service/notifications/errors.go`
- [x] Создать `internal/service/notifications/models/models.go`

### ✅ Этап 7: Worker Layer (4 файла)
- [x] Создать `internal/worker/scheduler.go` (gocron + LoadScheduledNotifications)
- [x] Создать `internal/worker/processor.go` (polling pending каждые 30s)
- [x] Создать `internal/worker/polling.go` (Long Polling для Telegram)
- [x] Создать `internal/worker/contracts.go`

### ✅ Этап 8: Handler Layer (16 файлов)
- [x] Создать `internal/api/handlers/create_notification/handler.go`
- [x] Создать `internal/api/handlers/create_notification/contract.go`
- [x] Создать `internal/api/handlers/create_notification/models/models.go`
- [x] Создать `internal/api/handlers/create_batch_notification/handler.go`
- [x] Создать `internal/api/handlers/list_notifications/handler.go`
- [x] Создать `internal/api/handlers/cancel_notification/handler.go`
- [x] Создать `internal/api/handlers/cancel_batch_notification/handler.go`
- [x] Создать `internal/api/handlers/telegram_webhook/handler.go`
- [x] Создать `internal/api/handlers/health/handler.go`

### ✅ Этап 9: Use Cases (2 файла)
- [x] Создать `internal/usecase/start_message/usecase.go`
- [x] Создать `internal/usecase/start_message/contract.go`

### ✅ Этап 10: Configuration (1 файл)
- [x] Обновить `internal/config/config.go` (добавить TelegramConfig, UserServiceConfig, WorkerConfig)

### ✅ Этап 11: Main Application (1 файл)
- [x] Полностью переписать `cmd/main.go`:
  - [x] Инициализация NotificationRepository, TelegramService, NotificationService
  - [x] Создание Scheduler и Processor
  - [x] **ВАЖНО:** scheduler.Start() и LoadScheduledNotifications при старте
  - [x] Запуск Processor в goroutine
  - [x] Поддержка Webhook и Long Polling режимов
  - [x] Регистрация всех handlers
  - [x] Graceful shutdown (Context → Worker → Metrics → HTTP Server)

### ✅ Этап 12: Docker & Build (3 файла)
- [x] Создать `Dockerfile.notificationservice`
- [x] Обновить `docker-compose.yml` (порт 5440, имя БД, переменные окружения)
- [x] Обновить `Makefile` (имена сервисов, порты)

### ✅ Этап 13: Тестирование
- [x] Запустить `make docker-up`
- [x] Проверить health check: `curl http://localhost:8085/health`
- [x] Проверить метрики: `curl http://localhost:8085/metrics`
- [x] Создать тестовое немедленное уведомление через API
- [x] Создать тестовое отложенное уведомление через API
- [x] Проверить отправку через Telegram Bot
- [x] Проверить Long Polling обработку /start команды
- [x] Проверить WebApp кнопки в Telegram
- [x] Проверить отложенные уведомления (scheduler)
- [x] Проверить автосоздание пользователей в UserService
- [x] Проверить интеграцию между сервисами

### ✅ Этап 14: Документация
- [x] Создать README.md с примерами использования
- [x] Добавить тестовые данные в `test_data/`
- [x] Обновить CLAUDE.md с итоговым статусом

---

## Итоговая статистика

**Всего файлов создано/обновлено:** 45+

**Ключевые достижения:**
- ✅ Полностью рабочий сервис уведомлений
- ✅ Интеграция с Telegram Bot API (WebApp buttons support)
- ✅ Двухрежимная работа: немедленные + отложенные уведомления
- ✅ Интеграция с UserService
- ✅ Docker-контейнеризация
- ✅ Prometheus метрики
- ✅ Graceful shutdown
- ✅ Персистентность отложенных уведомлений

**Протестированные сценарии:**
1. ✅ Немедленное уведомление → отправлено через 7 секунд processor'ом
2. ✅ Отложенное уведомление → отправлено точно в запланированное время scheduler'ом
3. ✅ Команда /start → пользователь создан в UserService + получено приветствие с WebApp кнопкой
4. ✅ Перезапуск сервиса → отложенные уведомления не потеряны

---

## Важные технические детали реализации

### Telegram WebApp Buttons

Для поддержки WebApp кнопок (открытие веб-приложения внутри Telegram) была обновлена библиотека:

```bash
GOPROXY=https://proxy.golang.org,direct go get -u github.com/go-telegram-bot-api/telegram-bot-api/v5@master
```

Текущая версия: `v5.5.2-0.20221020003552-4126fa611266` (master ветка с поддержкой WebApp)

**Код создания WebApp кнопки:**
```go
button := tgbotapi.InlineKeyboardButton{
    Text: "Открыть приложение",
    WebApp: &tgbotapi.WebAppInfo{
        URL: "https://your-webapp.com",
    },
}
```

### Docker Networking

Для интеграции между сервисами в Docker и на хост-машине:

- **UserService на хосте** → используем `host.docker.internal:8080`
- **config.toml**: `url = "http://host.docker.internal:8080"`
- **PostgreSQL внутри Docker** → используем имя сервиса `postgres:5432`

### Критичные моменты Scheduler

1. **scheduler.Start() ОБЯЗАТЕЛЕН** - без него задачи не будут выполняться
2. **Порядок важен** - сначала `Start()`, потом `LoadScheduledNotifications()`
3. **Просроченные задачи** - gocron не выполняет задачи, время которых уже прошло
4. **Персистентность** - при перезапуске загружаем все `scheduled` из БД

### Context Management для Goroutines

```go
// Создаём контекст с возможностью отмены
ctx, cancelCtx := context.WithCancel(context.Background())
defer cancelCtx()

// Все фоновые goroutines получают этот контекст
go pollingHandler.Start(ctx, updatesChan)
go processor.Start() // processor создаёт свой контекст внутри
```

### Graceful Shutdown порядок

```go
1. cancelCtx()           // Отменяем контекст для всех goroutines
2. processor.Stop()      // Ждём завершения текущего батча
3. scheduler.Stop()      // Останавливаем gocron
4. close(stopMetricsCh)  // Останавливаем сбор метрик
5. srv.Shutdown(...)     // Graceful shutdown HTTP сервера
```

### Формат времени для scheduled_for

- **Формат:** ISO 8601 в UTC
- **Пример:** `"2025-12-07T14:26:00Z"`
- **Генерация (macOS):** `date -u -v+1M +"%Y-%m-%dT%H:%M:%SZ"`
- **Генерация (Linux):** `date -u -d "+1 minute" +"%Y-%m-%dT%H:%M:%SZ"`

### Отличия полей в API

- ❌ `notification_type` - НЕПРАВИЛЬНО (старое поле)
- ✅ `type` - ПРАВИЛЬНО (используется в API)

---

## Следующие шаги (опционально)

### Возможные улучшения

1. **Message Broker** - Переход с PostgreSQL на RabbitMQ/Kafka для очереди
2. **Batch API оптимизация** - Параллельная отправка через goroutines pool
3. **Retry механизм** - Exponential backoff для failed уведомлений
4. **Template Engine** - Шаблоны сообщений с переменными
5. **Rate Limiting** - Ограничение частоты отправки per user
6. **Delivery Reports** - Callback при доставке сообщения
7. **OpenAPI спецификация** - Автогенерация клиентов
8. **Unit Tests** - Покрытие тестами критичных компонентов
9. **Integration Tests** - E2E тесты с test containers
10. **Webhook Mode** - Production режим с HTTPS webhook вместо Long Polling

---

## Лицензия

MIT
