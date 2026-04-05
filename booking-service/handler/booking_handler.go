package handler

import (
	"booking-service/models"
	"booking-service/service"
	"encoding/json"
	"net/http"
	"strings"
)

type BookingHandler struct {
	service service.BookingService
}

func NewBookingHandler(svc service.BookingService) *BookingHandler {
	return &BookingHandler{service: svc}
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var b models.Booking
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"geçersiz JSON formatı"}`))
		return
	}

	id, err := h.service.CreateBooking(b)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		status := http.StatusBadRequest
		w.WriteHeader(status)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "bilet oluşturuldu"})
}

func (h *BookingHandler) GetByUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"user_id parametresi gerekli"}`))
		return
	}

	bookings, err := h.service.GetBookingsByUser(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookings)
}

func SetupRouter(svc service.BookingService) *http.ServeMux {
	h := NewBookingHandler(svc)
	mux := http.NewServeMux()

	mux.HandleFunc("/bookings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.Create(w, r)
		case http.MethodGet:
			h.GetByUser(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/bookings/", func(w http.ResponseWriter, r *http.Request) {
		_ = strings.TrimPrefix(r.URL.Path, "/bookings/")
		http.Error(w, "not implemented", http.StatusNotImplemented)
	})

	return mux
}