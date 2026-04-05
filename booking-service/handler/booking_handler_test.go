package handler_test

import (
	"booking-service/handler"
	"booking-service/models"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockBookingService struct {
	createID  string
	createErr error
	bookings  []models.Booking
	getErr    error
}

func (m *mockBookingService) CreateBooking(_ models.Booking) (string, error) {
	return m.createID, m.createErr
}

func (m *mockBookingService) GetBookingsByUser(_ string) ([]models.Booking, error) {
	return m.bookings, m.getErr
}

// Yardımcı

func makeRequest(router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// POST /bookings

func TestCreate_Basarili_201(t *testing.T) {
	svc := &mockBookingService{createID: "mock-id-123"}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "POST", "/bookings",
		map[string]any{"event_id": "konser123", "user_id": "oguzhan", "seats": 2})

	if rr.Code != http.StatusCreated {
		t.Errorf("beklenen 201, alınan %d", rr.Code)
	}
}

func TestCreate_GeçersizJSON_400(t *testing.T) {
	svc := &mockBookingService{}
	router := handler.SetupRouter(svc)

	req, _ := http.NewRequest("POST", "/bookings", bytes.NewBufferString("bu json değil"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestCreate_ServiceHataVerirse_400(t *testing.T) {
	svc := &mockBookingService{createErr: errors.New("etkinlik bulunamadı")}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "POST", "/bookings",
		map[string]any{"event_id": "yok", "user_id": "oguzhan", "seats": 2})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestCreate_YanitBodyIDIceriyor(t *testing.T) {
	svc := &mockBookingService{createID: "abc-123"}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "POST", "/bookings",
		map[string]any{"event_id": "konser123", "user_id": "oguzhan", "seats": 2})

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["id"] != "abc-123" {
		t.Errorf("yanıtta ID beklendi 'abc-123', alınan '%s'", resp["id"])
	}
}

// GET /bookings

func TestGetByUser_Basarili_200(t *testing.T) {
	svc := &mockBookingService{bookings: []models.Booking{
		{EventID: "konser1", UserID: "oguzhan", Seats: 2},
	}}
	router := handler.SetupRouter(svc)

	req, _ := http.NewRequest("GET", "/bookings?user_id=oguzhan", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}

	var bookings []models.Booking
	json.NewDecoder(rr.Body).Decode(&bookings)
	if len(bookings) != 1 {
		t.Errorf("1 rezervasyon bekleniyordu, %d alındı", len(bookings))
	}
}

func TestGetByUser_UserIDEksik_400(t *testing.T) {
	svc := &mockBookingService{}
	router := handler.SetupRouter(svc)

	req, _ := http.NewRequest("GET", "/bookings", nil) // user_id yok
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

func TestGetByUser_ServiceHataVerirse_500(t *testing.T) {
	svc := &mockBookingService{getErr: errors.New("db hatası")}
	router := handler.SetupRouter(svc)

	req, _ := http.NewRequest("GET", "/bookings?user_id=oguzhan", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("beklenen 500, alınan %d", rr.Code)
	}
}