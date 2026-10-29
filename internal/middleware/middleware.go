package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ResponseWriterInterceptor captura o status code para logging
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterInterceptor) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Logger middleware registra cada requisição HTTP com duração e status
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		interceptor := &responseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(interceptor, r)

		duration := time.Since(start)
		log.Printf("[%s] %s %s | Status: %d | Duração: %v",
			r.Method,
			r.RemoteAddr,
			r.URL.Path,
			interceptor.statusCode,
			duration,
		)
	})
}

// RequestID injeta um UUID único para cada requisição para rastreabilidade
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

// JSONContentType força a resposta como application/json
func JSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}
