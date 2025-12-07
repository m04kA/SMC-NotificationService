package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// NotificationType представляет тип уведомления
type NotificationType string

const (
	NotificationTypeWelcome          NotificationType = "welcome"
	NotificationTypeBookingCreated   NotificationType = "booking_created"
	NotificationTypeBookingConfirmed NotificationType = "booking_confirmed"
	NotificationTypeBookingReminder  NotificationType = "booking_reminder"
	NotificationTypeBookingCancelled NotificationType = "booking_cancelled"
	NotificationTypePromo            NotificationType = "promo"
)

// NotificationStatus представляет статус уведомления
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"   // Ожидает отправки
	NotificationStatusScheduled NotificationStatus = "scheduled" // Запланировано
	NotificationStatusSent      NotificationStatus = "sent"      // Отправлено
	NotificationStatusFailed    NotificationStatus = "failed"    // Ошибка
	NotificationStatusCancelled NotificationStatus = "cancelled" // Отменено
)

// InlineButton представляет inline-кнопку в Telegram
type InlineButton struct {
	Text string `json:"text"` // Текст кнопки
	URL  string `json:"url"`  // URL для перехода
}

// InlineButtons - массив inline-кнопок для хранения в БД
type InlineButtons []InlineButton

// Value реализует driver.Valuer для записи в БД
func (b InlineButtons) Value() (driver.Value, error) {
	if b == nil {
		return nil, nil
	}
	return json.Marshal(b)
}

// Scan реализует sql.Scanner для чтения из БД
func (b *InlineButtons) Scan(value interface{}) error {
	if value == nil {
		*b = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan InlineButtons: expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, b)
}

// Notification представляет уведомление в системе
type Notification struct {
	ID             int64              `db:"id"`
	TelegramUserID *int64             `db:"telegram_user_id"`
	ChatID         *int64             `db:"chat_id"`
	SpanID         *string            `db:"span_id"` // UUID для группировки массовых рассылок
	MessageText    string             `db:"message_text"`
	ImageURLs      pq.StringArray     `db:"image_urls"` // Массив URL изображений (до 10)
	InlineButtons  InlineButtons      `db:"inline_buttons"`
	Type           NotificationType   `db:"notification_type"`
	Status         NotificationStatus `db:"status"`
	ScheduledFor   *time.Time         `db:"scheduled_for"`
	SentAt         *time.Time         `db:"sent_at"`
	Metadata       Metadata           `db:"metadata"`
	ErrorMessage   *string            `db:"error_message"`
	RetryCount     int                `db:"retry_count"`
	CreatedAt      time.Time          `db:"created_at"`
	UpdatedAt      time.Time          `db:"updated_at"`
}

// Metadata представляет дополнительные данные уведомления
type Metadata map[string]interface{}

// Value реализует driver.Valuer для записи в БД
func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

// Scan реализует sql.Scanner для чтения из БД
func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = make(Metadata)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan Metadata: expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, m)
}

// GetChatID возвращает chat_id с приоритетом: chat_id > telegram_user_id
func (n *Notification) GetChatID() int64 {
	if n.ChatID != nil {
		return *n.ChatID
	}
	if n.TelegramUserID != nil {
		return *n.TelegramUserID
	}
	return 0
}

// IsScheduled проверяет, является ли уведомление отложенным
func (n *Notification) IsScheduled() bool {
	return n.Status == NotificationStatusScheduled && n.ScheduledFor != nil
}

// IsPending проверяет, ожидает ли уведомление отправки
func (n *Notification) IsPending() bool {
	return n.Status == NotificationStatusPending
}

// CanBeCancelled проверяет, можно ли отменить уведомление
func (n *Notification) CanBeCancelled() bool {
	return n.Status == NotificationStatusPending || n.Status == NotificationStatusScheduled
}

// HasImages проверяет, есть ли у уведомления изображения
func (n *Notification) HasImages() bool {
	return len(n.ImageURLs) > 0
}

// HasButtons проверяет, есть ли у уведомления inline-кнопки
func (n *Notification) HasButtons() bool {
	return len(n.InlineButtons) > 0
}

// IsMediaGroup проверяет, нужно ли отправлять как MediaGroup (2+ изображения)
func (n *Notification) IsMediaGroup() bool {
	return len(n.ImageURLs) > 1
}
