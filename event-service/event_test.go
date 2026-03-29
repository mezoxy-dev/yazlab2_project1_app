package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Testler için bellekte çalışan sahte veritabanı
type MockEventRepo struct {
	events []Event
}

func (m *MockEventRepo) CreateEvent(e Event) error {
	m.events = append(m.events, e)
	return nil
}

func (m *MockEventRepo) GetAllEvents() ([]Event, error) {
	return m.events, nil
}

func (m *MockEventRepo) GetEventByID(id string) (*Event, error) {
	return nil, nil
}

func TestEventService(t *testing.T) {
	// Test ortamı kuruyoruz
	mockRepo := &MockEventRepo{}
	service := &EventService{Repo: mockRepo}
	router := setupRouter(service)

	t.Run("Herkes etkinlikleri listeleyebilmeli (GET /events)", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/events", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("200 bekleniyordu, alınan: %v", rr.Code)
		}
	})

	t.Run("Admin olmayan kullanıcı etkinlik ekleyememeli (403)", func(t *testing.T) {
		eventData := map[string]interface{}{"name": "Konser", "capacity": 100}
		body, _ := json.Marshal(eventData)
		req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(body))

		// Rol 'user' olarak set ediliyor
		req.Header.Set("X-User-Role", "user") // amacı admin olmayan bir kullanıcı simüle etmek

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("Admin olmayan için 403 bekleniyordu, alınan: %v", rr.Code)
		}
	})

	t.Run("Admin kullanıcı etkinlik EKLEYEBİLMELİ (201)", func(t *testing.T) {
		eventData := map[string]interface{}{
			"name": "Büyük Festival", 
			"capacity": 500,
			"location": "Kocaeli",
		}
		body, _ := json.Marshal(eventData)
		req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(body))
		
		// Rol 'admin' olarak set ediliyor
		req.Header.Set("X-User-Role", "admin")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("Admin için 201 bekleniyordu, alınan: %v", rr.Code)
		}
	})
}