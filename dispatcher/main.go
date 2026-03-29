package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
	"github.com/golang-jwt/jwt/v5" 
)

// AuthMiddleware: JWT kontrolü yapar
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS ön kontrol isteklerine izin ver
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            return
        }

		// Login rotasına herkes erişebilmeli (Token alabilmek için)
		if (r.URL.Path == "/login" || r.URL.Path == "/register"){	
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Token bulunamadı"}`))
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Token Doğrulama
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			log.Fatal("KRITIK: JWT_SECRET ortam değişkeni tanımlı değil.")
		}
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Geçersiz token"}`))
			return
		}

		// Kullanıcı bilgisini Header'a ekleyerek iç servislere pasla
		// İç servisler sadece token doğrulamakla kalmaz, aynı zamanda kullanıcı bilgisine de ihtiyaç duyabilirler (örneğin, kullanıcı adı veya rolü). 
		// Bu bilgiyi token'dan çıkarıp HTTP header'ına ekleyerek, iç servislerin bu bilgilere kolayca erişmesini sağlarız. 
		// Böylece, iç servisler sadece token doğrulamakla kalmaz, aynı zamanda kullanıcıya özel işlemler yapabilirler (örneğin, belirli bir rolün erişimine izin vermek gibi).
        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            r.Header.Set("X-User-Name", claims["sub"].(string))
            r.Header.Set("X-User-Role", claims["role"].(string))
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

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Sen dispatcher mısın kontrolü yaparak iç servisler arası güvenliği artırıyoruz
		internalKey := os.Getenv("INTERNAL_GATEWAY_KEY")
		req.Header.Set("X-Internal-Secret", internalKey)
	}


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