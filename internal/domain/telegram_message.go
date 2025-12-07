package domain

// ParseMode константы для режимов парсинга текста в Telegram
const (
	ParseModeHTML     = "HTML"     // HTML форматирование (рекомендуется для шаблонов)
	ParseModeMarkdown = "Markdown" // Markdown форматирование (legacy)
	ParseModePlain    = ""         // Без форматирования (для пользовательского контента)
)

// TelegramMessage представляет сообщение для отправки через Telegram Bot API
type TelegramMessage struct {
	ChatID        int64          // ID чата получателя
	MessageText   string         // Текст сообщения
	ImageURLs     []string       // URL изображений (для MediaGroup или одиночного фото)
	InlineButtons []InlineButton // Inline-кнопки
	ParseMode     string         // Режим парсинга (HTML, Markdown, Plain)
}

// NewTelegramMessage создает новое сообщение из доменной модели Notification
// По умолчанию использует HTML для шаблонных сообщений
func NewTelegramMessage(notification *Notification) *TelegramMessage {
	return &TelegramMessage{
		ChatID:        notification.GetChatID(),
		MessageText:   notification.MessageText,
		ImageURLs:     notification.ImageURLs,
		InlineButtons: notification.InlineButtons,
		ParseMode:     ParseModeHTML, // HTML по умолчанию для шаблонных сообщений
	}
}

// NewPlainTelegramMessage создает сообщение без форматирования
// Используется для пользовательского контента (отзывы, комментарии)
func NewPlainTelegramMessage(notification *Notification) *TelegramMessage {
	msg := NewTelegramMessage(notification)
	msg.ParseMode = ParseModePlain
	return msg
}

// HasImages проверяет, есть ли изображения для отправки
func (m *TelegramMessage) HasImages() bool {
	return len(m.ImageURLs) > 0
}

// IsMediaGroup проверяет, нужно ли отправлять как MediaGroup
// MediaGroup используется для 2+ изображений в одном сообщении
func (m *TelegramMessage) IsMediaGroup() bool {
	return len(m.ImageURLs) > 1
}

// HasButtons проверяет, есть ли inline-кнопки
func (m *TelegramMessage) HasButtons() bool {
	return len(m.InlineButtons) > 0
}

// GetFirstImage возвращает URL первого изображения (для одиночного фото)
func (m *TelegramMessage) GetFirstImage() string {
	if len(m.ImageURLs) > 0 {
		return m.ImageURLs[0]
	}
	return ""
}

// WithParseMode устанавливает режим парсинга и возвращает сообщение (builder pattern)
func (m *TelegramMessage) WithParseMode(mode string) *TelegramMessage {
	m.ParseMode = mode
	return m
}
