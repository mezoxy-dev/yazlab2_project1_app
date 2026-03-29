package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
	"github.com/golang-jwt/jwt/v5" 
)

// AuthMiddleware: JWT kontrolü yapar
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Login rotasına herkes erişebilmeli (Token alabilmek için)
		if (r.URL.Path == "/login" || r.URL.Path == "/register"){	
			next.ServeHTTP(w, r)
			return
		}

		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Token bulunamadı"}`))
			return
		}

		// Token Doğrulama
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "gizli_anahtar_oguzhan" // Eğer Docker'dan gelmezse yedek
		}
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Geçersiz token"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ProxyHandler: İstekleri ilgili servislere yönlendirir
func ProxyHandler(w http.ResponseWriter, r *http.Request, target string) {
	dest, err := url.Parse(target)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(dest)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		log.Printf("Hata: Servis ulaşılamıyor: %v", e)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error": "Hedef servis ulaşılamaz durumda"}`))
	}
	proxy.ServeHTTP(w, r)
}

func main() {
    mux := http.NewServeMux()

    // "/" yerine spesifik rotalar tanımlamak her zaman daha güvenlidir
    mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
        ProxyHandler(w, r, "http://auth-service:8081")
    })

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		ProxyHandler(w, r, "http://auth-service:8081")
	})

    mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
        ProxyHandler(w, r, "http://event-service:8082")
    })

    mux.HandleFunc("/booking", func(w http.ResponseWriter, r *http.Request) {
        ProxyHandler(w, r, "http://booking-service:8083")
    })

    // Loglama için bir middleware daha ekleyelim (Opsiyonel ama rapor için güzel durur)
    loggingMiddleware := func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            log.Printf("İSTEK GELDİ: [%s] %s %s", r.Method, r.RemoteAddr, r.URL.Path)
            next.ServeHTTP(w, r)
            log.Printf("İSTEK BİTTİ: %s | Süre: %v", r.URL.Path, time.Since(start))
        })
    }	

    // Sıralama: Önce Logla -> Sonra Auth Kontrol Et -> Sonra Rotalara Gönder
    finalHandler := loggingMiddleware(AuthMiddleware(mux))

    log.Println("Dispatcher 8080 portunda çalışıyor...")
    log.Fatal(http.ListenAndServe(":8080", finalHandler))
}