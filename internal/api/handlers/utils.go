package handlers

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse структура для ответа с ошибкой
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RespondJSON отправляет JSON ответ
func RespondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		json.NewEncoder(w).Encode(payload)
	}
}

// RespondError отправляет ошибку в формате JSON
func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, ErrorResponse{
		Code:    status,
		Message: message,
	})
}

// DecodeJSON парсит JSON из request body
func DecodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// RespondBadRequest отправляет ошибку 400
func RespondBadRequest(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusBadRequest, message)
}

// RespondUnauthorized отправляет ошибку 401
func RespondUnauthorized(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusUnauthorized, message)
}

// RespondForbidden отправляет ошибку 403
func RespondForbidden(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusForbidden, message)
}

// RespondNotFound отправляет ошибку 404
func RespondNotFound(w http.ResponseWriter, message string) {
	RespondError(w, http.StatusNotFound, message)
}

// RespondInternalError отправляет ошибку 500
func RespondInternalError(w http.ResponseWriter) {
	RespondError(w, http.StatusInternalServerError, "internal server error")
}
