package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/m04kA/SMC-NotificationService/pkg/metrics"
)

// MetricsMiddleware собирает метрики для HTTP запросов
func MetricsMiddleware(metrics *metrics.Metrics, serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Засекаем время начала запроса
			start := time.Now()

			// Создаём ResponseWriter обёртку для захвата status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // По умолчанию 200
			}

			// Выполняем следующий handler
			next.ServeHTTP(rw, r)

			// Вычисляем длительность
			duration := time.Since(start).Seconds()

			// Получаем данные для метрик
			method := r.Method
			endpoint := r.URL.Path
			statusCode := strconv.Itoa(rw.statusCode)

			// Записываем метрики
			metrics.RecordHTTPRequest(serviceName, method, endpoint, statusCode, duration)

			// Если ошибка - записываем дополнительную метрику
			if rw.statusCode >= 400 {
				errorType := categorizeError(rw.statusCode)
				metrics.RecordHTTPError(serviceName, method, endpoint, statusCode, errorType)
			}
		})
	}
}

// responseWriter обёртка над http.ResponseWriter для захвата status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader перехватывает status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// categorizeError категоризирует ошибки по типам
func categorizeError(statusCode int) string {
	switch {
	case statusCode == 400:
		return "bad_request"
	case statusCode == 401:
		return "unauthorized"
	case statusCode == 403:
		return "forbidden"
	case statusCode == 404:
		return "not_found"
	case statusCode == 409:
		return "conflict"
	case statusCode >= 400 && statusCode < 500:
		return "client_error"
	case statusCode == 500:
		return "internal_error"
	case statusCode == 502:
		return "bad_gateway"
	case statusCode == 503:
		return "service_unavailable"
	case statusCode == 504:
		return "gateway_timeout"
	case statusCode >= 500:
		return "server_error"
	default:
		return "unknown"
	}
}
