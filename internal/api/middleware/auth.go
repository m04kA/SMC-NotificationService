package middleware

import (
	"context"
	"net/http"
	"strconv"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UserRoleKey contextKey = "user_role"
)

// Auth извлекает заголовки аутентификации и сохраняет их в контекст
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDStr := r.Header.Get("X-User-ID")
		userRole := r.Header.Get("X-User-Role")

		if userIDStr == "" || userRole == "" {
			http.Error(w, "missing authentication headers", http.StatusUnauthorized)
			return
		}

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid user ID", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, UserRoleKey, userRole)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID извлекает user ID из контекста
func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}

// GetUserRole извлекает user role из контекста
func GetUserRole(ctx context.Context) (string, bool) {
	userRole, ok := ctx.Value(UserRoleKey).(string)
	return userRole, ok
}
