// dispatcher/dispatcher_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDispatcher(t *testing.T) {
	// olusturulan dispatcher'ı test etmek için bir HTTP isteği yapalım
	req, err := http.NewRequest("GET", "/secure-route", nil)
	if err != nil {
		t.Fatal(err)
	}
	
	// Kaydedici oluştur ve dispatcher'ı test etmek için bir HTTP isteği yapalım
	rr := httptest.NewRecorder()
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	// token yoksa 401 Unauthorized döndürülmeli
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("Beklenen durum kodu %v, ancak alınan %v", http.StatusUnauthorized, status)
	}}

func TestProxyRouting(t *testing.T) {
    // 1. Sahte bir arka uç (backend) servisi oluştur
    backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Backend Response"))
    }))
    defer backend.Close()

    // 2. Dispatcher'ın bu sahte servise yönlendirme yapıp yapmadığını test et
    req := httptest.NewRequest("GET", "/events", nil)
    rr := httptest.NewRecorder()

    // ProxyHandler fonksiyonunu henüz yazmadık (RED)
    ProxyHandler(rr, req, backend.URL)

    if rr.Body.String() != "Backend Response" {
        t.Errorf("Beklenen yanıt gelmedi: %s", rr.Body.String())
    }
}