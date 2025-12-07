package userservice

import "time"

// User модель пользователя из UserService
type User struct {
	TgUserID    int64     `json:"tg_user_id"`
	Name        string    `json:"name"`
	PhoneNumber string    `json:"phone_number"`
	TgLink      *string   `json:"tg_link"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

// ErrorResponse модель ошибки от UserService
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CreateUserRequest запрос на создание пользователя
type CreateUserRequest struct {
	TgUserID    int64   `json:"tg_user_id"`
	Name        string  `json:"name"`
	PhoneNumber *string `json:"phone_number,omitempty"` // Опционально
	TgLink      *string `json:"tg_link,omitempty"`
	Role        string  `json:"role"`
}
