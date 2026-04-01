package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TrafficLog struct {
	Method    string    `json:"method" bson:"method"`
	Path      string    `json:"path" bson:"path"`
	Status    int       `json:"status" bson:"status"`
	Duration  int64     `json:"duration_ms" bson:"duration_ms"`
	IP        string    `json:"ip" bson:"ip"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

var logCollection *mongo.Collection

// Güvenli CORS Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigins := map[string]bool{
			"http://localhost:3000": true,
			"http://127.0.0.1:3000": true,
		}

		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin == "" {
			// Postman veya curl gibi browser dışı araçlar için opsiyonel izin
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Internal-Secret")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Login ve Register korumasız (Public)
		if r.URL.Path == "/login" || r.URL.Path == "/register" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		secret := os.Getenv("JWT_SECRET")

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			r.Header.Set("X-User-Name", claims["sub"].(string))
			r.Header.Set("X-User-Role", claims["role"].(string))
		}

		next.ServeHTTP(w, r)
	})
}

func ProxyHandler(w http.ResponseWriter, r *http.Request, target string) {
	dest, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(dest)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Internal-Secret", os.Getenv("INTERNAL_GATEWAY_KEY"))
	}

	proxy.ServeHTTP(w, r)
}

type StatusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://db-dispatcher:27017"
	}

	var client *mongo.Client
	var err error
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		if err == nil {
			err = client.Ping(ctx, nil)
			if err == nil {
				cancel()
				break
			}
		}
		cancel()
		log.Printf("MongoDB henüz hazır değil, bekleniyor... (%d/5)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatal("Dispatcher Log DB bağlantı hatası:", err)
	}
	logCollection = client.Database("dispatcher_logs").Collection("traffic")

	mux := http.NewServeMux()

	// Dashboard Veri Kaynağı (Sadece Admin)
	mux.HandleFunc("/admin/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-User-Role") != "admin" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		opts := options.Find().SetSort(bson.M{"timestamp": -1}).SetLimit(100)
		cursor, _ := logCollection.Find(context.TODO(), bson.M{}, opts)
		var logs []TrafficLog
		cursor.All(context.TODO(), &logs)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(logs)
	})

	// --- GÜNCELLENEN ROTALAR ---
	// Auth Servisi Rotaları
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) { ProxyHandler(w, r, "http://auth-service:8081") })
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) { ProxyHandler(w, r, "http://auth-service:8081") })
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) { ProxyHandler(w, r, "http://auth-service:8081") })

	// Event Servisi Rotaları (Çoğul - Taksimli ve Taksimsiz)
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) { ProxyHandler(w, r, "http://event-service:8082") })
	mux.HandleFunc("/events/", func(w http.ResponseWriter, r *http.Request) { ProxyHandler(w, r, "http://event-service:8082") })

	// Booking Servisi Rotaları (Çoğul - Taksimli ve Taksimsiz)
	mux.HandleFunc("/bookings", func(w http.ResponseWriter, r *http.Request) { ProxyHandler(w, r, "http://booking-service:8083") })
	mux.HandleFunc("/bookings/", func(w http.ResponseWriter, r *http.Request) { ProxyHandler(w, r, "http://booking-service:8083") })

	dbLoggingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &StatusRecorder{ResponseWriter: w, Status: 200}
			next.ServeHTTP(recorder, r)

			entry := TrafficLog{
				Method:    r.Method,
				Path:      r.URL.Path,
				Status:    recorder.Status,
				Duration:  time.Since(start).Milliseconds(),
				IP:        r.RemoteAddr,
				Timestamp: time.Now(),
			}

			go func(l TrafficLog) {
				_, _ = logCollection.InsertOne(context.Background(), l)
			}(entry)
		})
	}

	// MiddleWare Sıralaması:  CORS (Güvenlik Kapısı) -> Logger -> Auth (Kimlik) -> Mux (Yönlendirme)
	finalHandler := corsMiddleware(dbLoggingMiddleware(AuthMiddleware(mux)))

	log.Println("Dispatcher 8080 portunda (Güvenli CORS Aktif) çalışıyor...")
	log.Fatal(http.ListenAndServe(":8080", finalHandler))
}
