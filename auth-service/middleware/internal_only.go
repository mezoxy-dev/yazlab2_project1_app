package middleware
 
import (
	"net/http"
	"os"
)
 
// InternalOnly: Sadece API Gateway'den gelen isteklere izin verir.
// Her isteğin X-Internal-Secret header'ını INTERNAL_GATEWAY_KEY env değişkeni
// ile karşılaştırır. Eşleşmezse 403 döner.
//
// Bu middleware handler veya service bilmez — sadece HTTP katmanında çalışır.
func InternalOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := os.Getenv("INTERNAL_GATEWAY_KEY")
		provided := r.Header.Get("X-Internal-Secret")
 
		if expected == "" || provided != expected {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "Doğrudan erişim yasaktır. Lütfen Gateway üzerinden erişiniz."}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
 