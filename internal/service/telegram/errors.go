package telegram

import "errors"

var (
	// ErrSendMessage возвращается при ошибке отправки сообщения
	ErrSendMessage = errors.New("service.telegram: failed to send message")

	// ErrSendPhoto возвращается при ошибке отправки фото
	ErrSendPhoto = errors.New("service.telegram: failed to send photo")

	// ErrSendMediaGroup возвращается при ошибке отправки media group
	ErrSendMediaGroup = errors.New("service.telegram: failed to send media group")

	// ErrInvalidChatID возвращается при некорректном chat_id
	ErrInvalidChatID = errors.New("service.telegram: invalid chat_id")

	// ErrEmptyMessage возвращается при пустом тексте сообщения
	ErrEmptyMessage = errors.New("service.telegram: message text is empty")

	// ErrSetWebhook возвращается при ошибке установки webhook
	ErrSetWebhook = errors.New("service.telegram: failed to set webhook")

	// ErrDeleteWebhook возвращается при ошибке удаления webhook
	ErrDeleteWebhook = errors.New("service.telegram: failed to delete webhook")
)
