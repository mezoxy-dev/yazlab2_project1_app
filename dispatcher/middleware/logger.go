package middleware

import (
	"bytes"
	"dispatcher/service"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status >= 400 || r.status == 201 {
		r.body.Write(b)
	}
	return r.ResponseWriter.Write(b)
}

// Her HTTP isteğini asenkron olarak MongoDB'ye loglar.
func Logger(logSvc service.LogService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Kendi stats isteğimizi loglamayalım (Mirror Effect engelleme)
			if r.URL.Path == "/admin/stats" {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			msg := ""
			if rec.status >= 400 || rec.status == 201 {
				msg = rec.body.String()
				if len(msg) > 200 {
					msg = msg[:197] + "..."
				}
			}

			logSvc.RecordAsync(
				r.Method,
				r.URL.Path,
				r.RemoteAddr,
				rec.status,
				time.Since(start).Milliseconds(),
				msg,
			)
		})
	}
}