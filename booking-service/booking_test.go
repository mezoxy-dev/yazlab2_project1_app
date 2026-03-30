package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 2. Test İçin Sahte (Mock) Veri Tabanı
type mockBookingRepo struct {
	bookings   map[string][]Booking
	shouldFail bool // Veri tabanı hatası simüle etmek için
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

// 4. Test Senaryoları (TDD: Red-Green-Refactor)
func TestCreateBooking(t *testing.T) {
	t.Run("Gecerli Veri Ile Bilet Alinmali (201 Created)", func(t *testing.T) {
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

	t.Run("Eksik Veri İle İstek Atilirsa 400 Bad Request Donmeli", func(t *testing.T) {
		mockRepo := &mockBookingRepo{}
		handler := NewBookingHandler(mockRepo)

		// seats bilgisi eksik veya 0
		body := []byte(`{"event_id": "konser123", "user_id": "oguzhan", "seats": 0}`)
		req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Beklenen durum kodu 400, alınan: %d", rr.Code)
		}
	})

	t.Run("Veritabani Hatasi Durumunda 500 Donmeli", func(t *testing.T) {
		// Mock DB'yi bilerek bozuyoruz
		mockRepo := &mockBookingRepo{shouldFail: true}
		handler := NewBookingHandler(mockRepo)

		body := []byte(`{"event_id": "konser123", "user_id": "oguzhan", "seats": 2}`)
		req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Beklenen durum kodu 500, alınan: %d", rr.Code)
		}
	})
}
