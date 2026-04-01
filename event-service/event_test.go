package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockEventRepo: Veritabanı işlemlerini simüle eden sahte depo
type MockEventRepo struct {
	events []Event
}

func (m *MockEventRepo) CreateEvent(e Event) error {
	// ID atanmadıysa yeni bir ID oluştur (İleriki testlerde ID ile arama yapabilmek için kritik)
	if e.ID.IsZero() {
		e.ID = primitive.NewObjectID()
	}
	m.events = append(m.events, e)
	return nil
}

func (m *MockEventRepo) GetAllEvents() ([]Event, error) {
	return m.events, nil
}

func (m *MockEventRepo) GetEventByID(id string) (*Event, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	for _, e := range m.events {
		if e.ID == objID {
			return &e, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *MockEventRepo) UpdateEvent(id string, event Event) error {
	objID, _ := primitive.ObjectIDFromHex(id)
	for i, e := range m.events {
		if e.ID == objID {
			// Mevcut etkinliğin alanlarını güncelle
			m.events[i].Name = event.Name
			m.events[i].Location = event.Location
			m.events[i].Capacity = event.Capacity
			m.events[i].Date = event.Date
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func (m *MockEventRepo) DeleteEvent(id string) error {
	objID, _ := primitive.ObjectIDFromHex(id)
	for i, e := range m.events {
		if e.ID == objID {
			// Elemanı dilimden (slice) çıkar
			m.events = append(m.events[:i], m.events[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func (m *MockEventRepo) UpdateAvailableTickets(id string, amount int) error {
	objID, _ := primitive.ObjectIDFromHex(id)
	for i, e := range m.events {
		if e.ID == objID {
			m.events[i].Available += amount
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func TestEventService(t *testing.T) {
	// 1. Güvenlik ve Ortam Kurulumu
	const internalKey = "test_gizli_anahtar"
	os.Setenv("INTERNAL_GATEWAY_KEY", internalKey)

	mockRepo := &MockEventRepo{}
	service := &EventService{Repo: mockRepo}

	// Router'ı middleware ile sarmalıyoruz
	router := InternalOnlyMiddleware(setupRouter(service))

	// Yardımcı Fonksiyon: Tüm isteklere otomatik olarak gizli anahtar ekler
	newRequest := func(method, url string, body []byte) *http.Request {
		req, _ := http.NewRequest(method, url, bytes.NewBuffer(body))
		req.Header.Set("X-Internal-Secret", internalKey)
		return req
	}

	t.Run("Gizli Anahtar Yoksa 403 Forbidden Vermeli", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/events", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("Anahtar yokken 403 bekleniyordu, %v alındı", rr.Code)
		}
	})

	t.Run("Herkes etkinlikleri listeleyebilmeli (GET /events)", func(t *testing.T) {
		req := newRequest("GET", "/events", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("200 bekleniyordu, %v alındı", rr.Code)
		}
	})

	t.Run("Admin Tüm Bilgilerle Etkinlik Ekleyebilmeli (POST /events)", func(t *testing.T) {
		eventData := Event{
			Name:     "Büyük Festival",
			Capacity: 500,
			Location: "Kocaeli",
			Date:     "2026-05-15",
		}
		body, _ := json.Marshal(eventData)
		req := newRequest("POST", "/events", body)
		req.Header.Set("X-User-Role", "admin")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("201 bekleniyordu, %v alındı", rr.Code)
		}

		// Kaydedilen veriyi doğrula (Listenin son elemanına bakıyoruz)
		if len(mockRepo.events) == 0 {
			t.Fatal("Etkinlik kaydedilmedi!")
		}

		saved := mockRepo.events[len(mockRepo.events)-1]
		if saved.Location != "Kocaeli" || saved.Date != "2026-05-15" {
			t.Errorf("Eksik veri kaydedildi: %+v", saved)
		}
		if saved.Available != 500 {
			t.Errorf("Available capacity'e eşit olmalıydı, alınan: %v", saved.Available)
		}
	})

	t.Run("Normal Kullanıcı Etkinlik Ekleyememeli (403)", func(t *testing.T) {
		eventData := Event{Name: "Yasak Etkinlik", Capacity: 100}
		body, _ := json.Marshal(eventData)
		req := newRequest("POST", "/events", body)
		req.Header.Set("X-User-Role", "user")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("Admin olmayan için 403 bekleniyordu, alınan: %v", rr.Code)
		}
	})

	t.Run("Tekil Etkinlik Getirme (GET /events/{id})", func(t *testing.T) {
		// Kaydedilen ilk etkinliği kullanıyoruz
		id := mockRepo.events[0].ID.Hex()
		req := newRequest("GET", "/events/"+id, nil)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("200 bekleniyordu, %v alındı", rr.Code)
		}
	})

	t.Run("Bilet Sayısını Düşürme (PATCH /events/{id})", func(t *testing.T) {
		id := mockRepo.events[0].ID.Hex()
		// Başlangıç değerini alalım (500 idi)
		initialAvailable := mockRepo.events[0].Available

		patchData := map[string]int{"amount": -1}
		body, _ := json.Marshal(patchData)

		req := newRequest("PATCH", "/events/"+id, body)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("PATCH 200 bekleniyordu, %v alındı", rr.Code)
		}

		if mockRepo.events[0].Available != initialAvailable-1 {
			t.Errorf("Bilet sayısı düşmedi. Beklenen: %v, Alınan: %v", initialAvailable-1, mockRepo.events[0].Available)
		}
	})

	t.Run("Admin Etkinlik Silebilmeli (DELETE)", func(t *testing.T) {
		id := mockRepo.events[0].ID.Hex()
		req := newRequest("DELETE", "/events/"+id, nil)
		req.Header.Set("X-User-Role", "admin")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Errorf("204 bekleniyordu, %v alındı", rr.Code)
		}
	})
	t.Run("Eksik veya Hatali Veriyle Etkinlik Eklenememeli (400)", func(t *testing.T) {
		// İsim boş, kapasite 0 ve lokasyon boş olarak ayarlanıyor
		eventData := Event{
			Name:     "",
			Capacity: 0,
			Location: "",
			Date:     "2026-06-01",
		}
		body, _ := json.Marshal(eventData)
		req := newRequest("POST", "/events", body)
		req.Header.Set("X-User-Role", "admin") // Admin bile olsa eksik veriyi geçirememeli

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Eksik veride 400 Bad Request bekleniyordu, alınan: %v", rr.Code)
		}
	})
}
