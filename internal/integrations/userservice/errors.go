package userservice

import "errors"

var (
	// ErrUserNotFound возвращается, когда пользователь не найден
	ErrUserNotFound = errors.New("user not found")

	// ErrInternal возвращается при внутренних ошибках клиента
	ErrInternal = errors.New("userservice client: internal error")

	// ErrInvalidResponse возвращается при некорректном ответе от сервиса
	ErrInvalidResponse = errors.New("userservice client: invalid response")
)
