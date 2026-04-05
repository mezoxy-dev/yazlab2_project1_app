package middleware

import (
	"net/http"
	"os"
)

func InternalOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := os.Getenv("INTERNAL_GATEWAY_KEY")
		provided := r.Header.Get("X-Internal-Secret")

		if expected == "" || provided != expected {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Doğrudan erişim yasaktır. Gateway üzerinden erişiniz."}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}