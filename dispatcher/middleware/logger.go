package middleware

import (
	"dispatcher/service"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Her HTTP isteğini asenkron olarak MongoDB'ye loglar.
func Logger(logSvc service.LogService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			logSvc.RecordAsync(
				r.Method,
				r.URL.Path,
				r.RemoteAddr,
				rec.status,
				time.Since(start).Milliseconds(),
			)
		})
	}
}