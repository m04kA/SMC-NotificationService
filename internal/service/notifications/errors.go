package notifications

import "errors"

var (
	// ErrNotificationNotFound возвращается, когда уведомление не найдено
	ErrNotificationNotFound = errors.New("service.notifications: notification not found")

	// ErrUserNotFound возвращается, когда пользователь не найден в UserService
	ErrUserNotFound = errors.New("service.notifications: user not found in UserService")

	// ErrInvalidInput возвращается при некорректных входных данных
	ErrInvalidInput = errors.New("service.notifications: invalid input data")

	// ErrInvalidRecipient возвращается когда не указан ни telegram_user_id, ни chat_id
	ErrInvalidRecipient = errors.New("service.notifications: either telegram_user_id or chat_id must be provided")

	// ErrCannotCancel возвращается, когда уведомление нельзя отменить
	ErrCannotCancel = errors.New("service.notifications: notification cannot be cancelled (already sent or failed)")

	// ErrInternal возвращается при внутренних ошибках сервиса
	ErrInternal = errors.New("service.notifications: internal error")
)
