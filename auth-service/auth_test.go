package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockRepository: Gerçek MongoDB yerine bellekte (RAM) çalışan sahte veritabanı.
// Testlerin hızlı çalışmasını ve dış bağımlılığa (DB) ihtiyaç duymamasını sağlar.
type MockRepository struct {
	users map[string]User
}

func (m *MockRepository) CreateUser(user User) error {
	if _, exists := m.users[user.Username]; exists {
		return errors.New("bu kullanıcı adı zaten alınmış")
	}
	m.users[user.Username] = user
	return nil
}

func (m *MockRepository) FindByUsername(username string) (*User, error) {
	user, exists := m.users[username]
	if !exists {
		return nil, errors.New("kullanıcı bulunamadı")
	}
	return &user, nil
}

// TestAuthService: Go test aracının bulup çalıştıracağı ana fonksiyon.
func TestAuthService(t *testing.T) {
	// Test ortamını hazırla
	mockRepo := &MockRepository{users: make(map[string]User)}
	service := &AuthService{repo: mockRepo, key: []byte("test_secret")}
	router := SetupRouter(service)



	// t.Run kullanarak testleri alt senaryolara bölüyoruz
	t.Run("Başarılı Kayıt ve Login Senaryosu", func(t *testing.T) {
		regData := map[string]string{"username": "oguzhan", "password": "142213"}
		body, _ := json.Marshal(regData)
		req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("Kayıt başarısız, beklenen 201, alınan: %v", rr.Code)
		}

		loginReq, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
		rrLogin := httptest.NewRecorder()
		router.ServeHTTP(rrLogin, loginReq)

		if rrLogin.Code != http.StatusOK {
			t.Errorf("Login başarısız, beklenen 200, alınan: %v", rrLogin.Code)
		}

		var resp map[string]string
		json.NewDecoder(rrLogin.Body).Decode(&resp)
		if resp["token"] == "" {
			t.Error("Login başarılı olmasına rağmen token dönmedi")
		}
	})

	t.Run("Mükerrer Kayıt Engellenmeli (400)", func(t *testing.T) { 
		// Daha önce kaydettiğimiz "oguzhan" ismiyle tekrar deniyoruz
		regData := map[string]string{"username": "oguzhan", "password": "baska_sifre"}
		body, _ := json.Marshal(regData)
		req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Aynı isimle kayıtta 400 bekleniyordu, alınan: %v", rr.Code)
		}
	})

	t.Run("Hatalı Şifre Denemesi (401)", func(t *testing.T) {
		loginData := map[string]string{"username": "oguzhan", "password": "yanlis_sifre"}
		body, _ := json.Marshal(loginData)
		req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Hatalı şifrede 401 bekleniyordu, alınan: %v", rr.Code)
		}
	})

	t.Run("Eksik JSON Verisi Kontrolü (400)", func(t *testing.T) {
		regData := map[string]string{"username": "sadece_isim"} // password eksik
		body, _ := json.Marshal(regData)
		req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Eksik bilgide 400 bekleniyordu, alınan: %v", rr.Code)
		}
	})
}