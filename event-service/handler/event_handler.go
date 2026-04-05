package handler
 
import (
	"encoding/json"
	"event-service/models"
	"event-service/service"
	"net/http"
	"strings"
)

type EventHandler struct {
	service service.EventService
}
 

func NewEventHandler(svc service.EventService) *EventHandler {
	return &EventHandler{service: svc}
}

func (h *EventHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	events, err := h.service.GetAllEvents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
 
// Create: POST /events
func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-User-Role") != "admin" {
		http.Error(w, "Admin yetkisi gerekli", http.StatusForbidden)
		return
	}
 
	var e models.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}
 
	if err := h.service.CreateEvent(e); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
 
	w.WriteHeader(http.StatusCreated)
}
 
// GetByID: GET /events/{id}
func (h *EventHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := extractID(r)
	event, err := h.service.GetEventByID(id)
	if err != nil {
		http.Error(w, "Etkinlik bulunamadı", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}
 
// Update: PUT /events/{id}
func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-User-Role") != "admin" {
		http.Error(w, "Admin yetkisi gerekli", http.StatusForbidden)
		return
	}
 
	id := extractID(r)
	var e models.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}
 
	if err := h.service.UpdateEvent(id, e); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "güncellendi"})
}
 
// Delete: DELETE /events/{id}
func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-User-Role") != "admin" {
		http.Error(w, "Admin yetkisi gerekli", http.StatusForbidden)
		return
	}
 
	id := extractID(r)
	if err := h.service.DeleteEvent(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
 
// PatchTickets: PATCH /events/{id}
func (h *EventHandler) PatchTickets(w http.ResponseWriter, r *http.Request) {
	id := extractID(r)
 
	var req models.TicketPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}
 
	if err := h.service.UpdateAvailableTickets(id, req.Amount); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "kontenjan güncellendi"})
}
 
// SetupRouter: Sadece kablolama yapar.
func SetupRouter(svc service.EventService) *http.ServeMux {
	h := NewEventHandler(svc)
	mux := http.NewServeMux()
 
	// /events → koleksiyon işlemleri
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetAll(w, r)
		case http.MethodPost:
			h.Create(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
 
	// /events/{id} → tekil kayıt işlemleri
	mux.HandleFunc("/events/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetByID(w, r)
		case http.MethodPut:
			h.Update(w, r)
		case http.MethodDelete:
			h.Delete(w, r)
		case http.MethodPatch:
			h.PatchTickets(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
 
	return mux
}
 
// extractID: "/events/abc123" → "abc123"
func extractID(r *http.Request) string {
return strings.TrimPrefix(r.URL.Path, "/events/")
}
 