package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// 1. Mock Repository (Değişmedi)
type mockBookingRepo struct {
	bookings   map[string][]Booking
	shouldFail bool
}

func (m *mockBookingRepo) CreateBooking(booking Booking) (string, error) {
	if m.shouldFail {
		return "", errors.New("veri tabanına yazılamadı")
	}
	booking.ID = "mock-id-123"
	if m.bookings == nil {
		m.bookings = make(map[string][]Booking)
	}
	m.bookings[booking.UserID] = append(m.bookings[booking.UserID], booking)
	return booking.ID, nil
}

func (m *mockBookingRepo) GetBookingsByUser(userID string) ([]Booking, error) {
	if m.shouldFail {
		return nil, errors.New("veri tabanı hatası")
	}
	return m.bookings[userID], nil
}

// 2. Test Senaryoları
func TestCreateBooking(t *testing.T) {
	// SAHTE EVENT SERVİSİ (Mock Server) OLUŞTURMA
	mockEventServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// İç güvenlik anahtarını kontrol et
		if r.Header.Get("X-Internal-Secret") != "test_gizli_anahtar" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		// GET: Etkinlik var mı ve yer var mı kontrolü
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK) // 200 dönerek "yer var" diyoruz
			return
		}

		// PATCH: Bilet satıldıktan sonra kapasite düşürme işlemi
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer mockEventServer.Close()

	// Ortam değişkenlerini test için ayarla
	os.Setenv("EVENT_SERVICE_URL", mockEventServer.URL)
	os.Setenv("INTERNAL_GATEWAY_KEY", "test_gizli_anahtar")

	t.Run("Gecerli Veri ve Event Onayi Ile Bilet Alinmali (201)", func(t *testing.T) {
		mockRepo := &mockBookingRepo{}
		handler := NewBookingHandler(mockRepo)

		body := []byte(`{"event_id": "konser123", "user_id": "oguzhan", "seats": 2}`)
		req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("Beklenen durum kodu 201, alınan: %d", rr.Code)
		}
	})

	t.Run("Eksik Veri İle İstek Atilirsa 400 Donmeli", func(t *testing.T) {
		mockRepo := &mockBookingRepo{}
		handler := NewBookingHandler(mockRepo)

		body := []byte(`{"event_id": "konser123", "user_id": "oguzhan", "seats": 0}`)
		req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Beklenen durum kodu 400, alınan: %d", rr.Code)
		}
	})

	t.Run("Event Bulunamazsa veya Yer Yoksa 400 Donmeli", func(t *testing.T) {
		// Sadece bu test için hata dönen bir sahte sunucu
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound) // Etkinlik yok hatası
		}))
		defer failServer.Close()
		os.Setenv("EVENT_SERVICE_URL", failServer.URL)

		mockRepo := &mockBookingRepo{}
		handler := NewBookingHandler(mockRepo)

		body := []byte(`{"event_id": "olmayan_konser", "user_id": "oguzhan", "seats": 2}`)
		req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Etkinlik yokken 400 bekleniyordu, alınan: %d", rr.Code)
		}

		// Test bitince orijinal URL'i geri koyalım
		os.Setenv("EVENT_SERVICE_URL", mockEventServer.URL)
	})
}
