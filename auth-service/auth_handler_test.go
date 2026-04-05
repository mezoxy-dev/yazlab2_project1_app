package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── Mock Service ──────────────────────────────────────────────────────────────
// service.AuthService interface'ini implement eder.
// Gerçek bcrypt/JWT çalıştırmaz — anında sonuç döner.
// Handler testleri sadece "HTTP adaptörü doğru çalışıyor mu?" sorusunu sorar.

type mockAuthService struct {
	registerErr error
	loginToken  string
	loginErr    error
}

func (m *mockAuthService) Register(_, _, _ string) error {
	return m.registerErr
}

func (m *mockAuthService) Login(_, _ string) (string, error) {
	return m.loginToken, m.loginErr
}

// Yardımcı

func makeRequest(router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// Register Handler Testleri

func TestRegisterHandler_Basarili_201(t *testing.T) {
	svc := &mockAuthService{}
	router := SetupRouter(svc)

	rr := makeRequest(router, "POST", "/register",
		map[string]string{"username": "oguzhan", "password": "142213"})

	if rr.Code != http.StatusCreated {
		t.Errorf("beklenen 201, alınan %d", rr.Code)
	}
}

func TestRegisterHandler_MukerrerKayit_400(t *testing.T) {
	svc := &mockAuthService{registerErr: errors.New("bu kullanıcı adı zaten alınmış")}
	router := SetupRouter(svc)

	rr := makeRequest(router, "POST", "/register",
		map[string]string{"username": "oguzhan", "password": "142213"})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestRegisterHandler_EksikAlan_400(t *testing.T) {
	svc := &mockAuthService{}
	router := SetupRouter(svc)

	// password alanı eksik → handler validasyonda 400 döner, service hiç çağrılmaz
	rr := makeRequest(router, "POST", "/register",
		map[string]string{"username": "oguzhan"})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("eksik alanda beklenen 400, alınan %d", rr.Code)
	}
}

// Login Handler Testleri

func TestLoginHandler_Basarili_200(t *testing.T) {
	svc := &mockAuthService{loginToken: "mock.jwt.token"}
	router := SetupRouter(svc)

	rr := makeRequest(router, "POST", "/login",
		map[string]string{"username": "oguzhan", "password": "142213"})

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Error("başarılı login'de token dönmeliydi")
	}
}

func TestLoginHandler_HataliSifre_401(t *testing.T) {
	svc := &mockAuthService{loginErr: errors.New("hatalı şifre")}
	router := SetupRouter(svc)

	rr := makeRequest(router, "POST", "/login",
		map[string]string{"username": "oguzhan", "password": "yanlis"})

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("beklenen 401, alınan %d", rr.Code)
	}
}