package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"github.com/golang-jwt/jwt/v5"
)

// Test için kullanılacak gizli anahtar
const testSecret = "gizli_anahtar_oguzhan"

// Yardımcı fonksiyon: Test için geçerli bir Token üretir
func generateTestToken(username string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(testSecret))
	return tokenString
}

func TestDispatcherSecurity(t *testing.T) {
	// Test ortamı için Environment Variable ayarla
	os.Setenv("JWT_SECRET", testSecret)

	t.Run("Token Yoksa 401 Donmeli", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/events", nil)
		rr := httptest.NewRecorder()
		
		handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Token yokken 401 bekleniyordu, %v alindi", status)
		}
	})

	t.Run("Gecerli Token ile GET Basarili Olmali", func(t *testing.T) {
		token := generateTestToken("oguzhan")
		req, _ := http.NewRequest("GET", "/events", nil)
		req.Header.Set("Authorization", token) // Token'ı ekle

		rr := httptest.NewRecorder()
		handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Gecerli token ile 200 bekleniyordu, %v alindi", status)
		}
	})

	t.Run("Gecerli Token ile POST (Bodyli) Basarili Olmali", func(t *testing.T) {
		token := generateTestToken("oguzhan")
		jsonBody := []byte(`{"event_id": "konser123", "seats": 2}`)
		
		req, _ := http.NewRequest("POST", "/booking", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", token)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("POST istegi basarisiz oldu, durum kodu: %v", status)
		}
	})
}

func TestProxyRouting(t *testing.T) {
	// 1. Sahte bir arka uç (backend) servisi oluştur
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Backend Response"))
	}))
	defer backend.Close()

	// 2. Dispatcher'ın yönlendirmesini test et
	req := httptest.NewRequest("GET", "/events", nil)
	rr := httptest.NewRecorder()

	ProxyHandler(rr, req, backend.URL)

	if rr.Body.String() != "Backend Response" {
		t.Errorf("Beklenen yanit gelmedi: %s", rr.Body.String())
	}
}