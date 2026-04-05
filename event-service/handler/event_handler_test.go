package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"event-service/handler"
	"event-service/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Mock Service

type mockEventService struct {
	createErr         error
	getAllResult       []models.Event
	getAllErr          error
	getByIDResult     *models.Event
	getByIDErr        error
	updateErr         error
	deleteErr         error
	patchTicketsErr   error
}

func (m *mockEventService) CreateEvent(_ models.Event) error {
	return m.createErr
}
func (m *mockEventService) GetAllEvents() ([]models.Event, error) {
	return m.getAllResult, m.getAllErr
}
func (m *mockEventService) GetEventByID(_ string) (*models.Event, error) {
	return m.getByIDResult, m.getByIDErr
}
func (m *mockEventService) UpdateEvent(_ string, _ models.Event) error {
	return m.updateErr
}
func (m *mockEventService) DeleteEvent(_ string) error {
	return m.deleteErr
}
func (m *mockEventService) UpdateAvailableTickets(_ string, _ int) error {
	return m.patchTicketsErr
}

// Yardımcı 

func makeRequest(router http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// GET /events

func TestGetAll_Basarili_200(t *testing.T) {
	svc := &mockEventService{getAllResult: []models.Event{{Name: "Fest"}}}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "GET", "/events", nil, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

// POST /events

func TestCreate_AdminBasarili_201(t *testing.T) {
	svc := &mockEventService{}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "POST", "/events",
		models.Event{Name: "Konser", Location: "İzmir", Capacity: 200},
		map[string]string{"X-User-Role": "admin"})

	if rr.Code != http.StatusCreated {
		t.Errorf("beklenen 201, alınan %d", rr.Code)
	}
}

func TestCreate_AdminDegilse_403(t *testing.T) {
	svc := &mockEventService{}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "POST", "/events",
		models.Event{Name: "Yasak", Location: "Ankara", Capacity: 50},
		map[string]string{"X-User-Role": "user"})

	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}

func TestCreate_ServiceHataVerirse_400(t *testing.T) {
	// Service validasyon hatası döndürürse handler 400 yazmalı
	svc := &mockEventService{createErr: errors.New("isim boş olamaz")}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "POST", "/events",
		models.Event{Name: "X", Location: "Y", Capacity: 10},
		map[string]string{"X-User-Role": "admin"})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

// GET /events/{id}

func TestGetByID_Basarili_200(t *testing.T) {
	id := primitive.NewObjectID()
	svc := &mockEventService{getByIDResult: &models.Event{ID: id, Name: "Festival"}}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "GET", "/events/"+id.Hex(), nil, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestGetByID_Bulunamadiysa_404(t *testing.T) {
	svc := &mockEventService{getByIDErr: errors.New("bulunamadı")}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "GET", "/events/"+primitive.NewObjectID().Hex(), nil, nil)
	if rr.Code != http.StatusNotFound {
		t.Errorf("beklenen 404, alınan %d", rr.Code)
	}
}

// PATCH /events/{id}

func TestPatchTickets_Basarili_200(t *testing.T) {
	svc := &mockEventService{}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "PATCH", "/events/"+primitive.NewObjectID().Hex(),
		map[string]int{"amount": -1}, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("beklenen 200, alınan %d", rr.Code)
	}
}

func TestPatchTickets_KontenjanYetersiz_400(t *testing.T) {
	svc := &mockEventService{patchTicketsErr: errors.New("kontenjan yetersiz")}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "PATCH", "/events/"+primitive.NewObjectID().Hex(),
		map[string]int{"amount": -1}, nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("beklenen 400, alınan %d", rr.Code)
	}
}

// DELETE /events/{id}

func TestDelete_AdminBasarili_204(t *testing.T) {
	svc := &mockEventService{}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "DELETE", "/events/"+primitive.NewObjectID().Hex(),
		nil, map[string]string{"X-User-Role": "admin"})

	if rr.Code != http.StatusNoContent {
		t.Errorf("beklenen 204, alınan %d", rr.Code)
	}
}

func TestDelete_AdminDegilse_403(t *testing.T) {
	svc := &mockEventService{}
	router := handler.SetupRouter(svc)

	rr := makeRequest(router, "DELETE", "/events/"+primitive.NewObjectID().Hex(),
		nil, map[string]string{"X-User-Role": "user"})

	if rr.Code != http.StatusForbidden {
		t.Errorf("beklenen 403, alınan %d", rr.Code)
	}
}