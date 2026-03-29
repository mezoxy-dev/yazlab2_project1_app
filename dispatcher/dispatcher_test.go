package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"github.com/golang-jwt/jwt/v5"
)

// Test için sabit gizli anahtar
const testSecret = "gizli_anahtar_oguzhan"

// Testler için JWT üretir
func generateTestToken(username string, role string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  username,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	})
	tokenString, _ := token.SignedString([]byte(testSecret))
	return fmt.Sprintf("Bearer %s", tokenString)
}

// Dispatcher'ın güvenlik mekanizmasını test eder
func TestDispatcherSecurity(t *testing.T) {
	// Dispatcher'ın okuduğu ortam değişkenini ayarla
	os.Setenv("JWT_SECRET", testSecret)

	// AuthMiddleware'ı test etmek için basit bir handler oluştura
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})
	handlerToTest := AuthMiddleware(mockHandler)

	t.Run("Login/Register Token Gerektirmemeli", func(t *testing.T) {
		publicRoutes := []string{"/login", "/register"}
		for _, route := range publicRoutes {
			req, _ := http.NewRequest("POST", route, nil)
			rr := httptest.NewRecorder()
			handlerToTest.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("%s rotası tokensız geçmeliydi, ancak %v alındı", route, rr.Code)
			}
		}
	})

	t.Run("Korumalı Rotada Token Yoksa 401 Dönmeli", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/events", nil)
		rr := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Korumalı rota tokensız 401 vermeliydi, ancak %v alındı", rr.Code)
		}
	})

	t.Run("Hatalı Token (Yanlış Secret) 401 Dönmeli", func(t *testing.T) {
		// Yanlış bir secret ile token üret
		wrongToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "test"})
		ts, _ := wrongToken.SignedString([]byte("yanlis_secret"))
		
		req, _ := http.NewRequest("GET", "/events", nil)
		req.Header.Set("Authorization", "Bearer "+ts)
		rr := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Hatalı token 401 vermeliydi, ancak %v alındı", rr.Code)
		}
	})

	t.Run("Gecerli Token ile GET Basarılı Olmalı", func(t *testing.T) {
		token := generateTestToken("oguzhan", "user")
		req, _ := http.NewRequest("GET", "/events", nil)
		req.Header.Set("Authorization", token)

		rr := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Geçerli token ile 200 bekleniyordu, %v alındı", rr.Code)
		}
	})

	t.Run("Gecerli Token ile POST + Body Basarılı Olmalı", func(t *testing.T) {
		token := generateTestToken("oguzhan", "user")
		body := []byte(`{"event_id": "1", "seats": 1}`)
		req, _ := http.NewRequest("POST", "/booking", bytes.NewBuffer(body))
		req.Header.Set("Authorization", token)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("POST isteği başarısız, durum kodu: %v", rr.Code)
		}
	})
}

func TestProxyRouting(t *testing.T) {
	// Sahte bir hedef servis (örneğin event-service) simüle et
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Proxied Content"))
	}))
	defer backend.Close()

	req := httptest.NewRequest("GET", "/events", nil)
	rr := httptest.NewRecorder()

	// ProxyHandler'ın gelen isteği backend'e iletip iletmediğini ölç
	ProxyHandler(rr, req, backend.URL)

	if rr.Body.String() != "Proxied Content" {
		t.Errorf("Proxy yönlendirmesi başarısız. Alınan: %s", rr.Body.String())
	}
}