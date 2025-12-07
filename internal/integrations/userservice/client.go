package userservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client клиент для работы с UserService
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient создает новый экземпляр клиента UserService
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetUser получает информацию о пользователе по Telegram user ID
// Используется для валидации существования пользователя перед отправкой уведомления
func (c *Client) GetUser(ctx context.Context, tgUserID int64) (*User, error) {
	url := fmt.Sprintf("%s/internal/users/%d", c.baseURL, tgUserID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrInternal, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to execute request: %v", ErrInternal, err)
	}
	defer resp.Body.Close()

	// Обработка статус-кодов
	switch resp.StatusCode {
	case http.StatusOK:
		// Продолжаем обработку
	case http.StatusNotFound:
		return nil, ErrUserNotFound
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: unexpected status code %d: %s", ErrInvalidResponse, resp.StatusCode, string(body))
	}

	// Парсим ответ
	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response: %v", ErrInvalidResponse, err)
	}

	return &user, nil
}

// CreateUser создает нового пользователя в UserService
// Используется при обработке команды /start если пользователь не найден в системе
func (c *Client) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
	url := fmt.Sprintf("%s/users", c.baseURL)

	// Кодируем тело запроса
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal request: %v", ErrInternal, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrInternal, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to execute request: %v", ErrInternal, err)
	}
	defer resp.Body.Close()

	// Обработка статус-кодов
	switch resp.StatusCode {
	case http.StatusCreated:
		// Продолжаем обработку
	case http.StatusBadRequest:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: bad request: %s", ErrInvalidResponse, string(body))
	case http.StatusConflict:
		return nil, fmt.Errorf("%w: user already exists", ErrInvalidResponse)
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: unexpected status code %d: %s", ErrInvalidResponse, resp.StatusCode, string(body))
	}

	// Парсим ответ
	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response: %v", ErrInvalidResponse, err)
	}

	return &user, nil
}
