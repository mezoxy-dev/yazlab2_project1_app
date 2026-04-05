package proxy

import (
	"dispatcher/service"
	"encoding/json"
	"net/http"
)

// SetupRouter: URL rotalarını ilgili downstream servislere bağlar. (SRP)
// Router → ProxyService (Forward) → downstream servisler
// Router → LogService  (GetRecent) → MongoDB (admin/stats için)
func SetupRouter(proxySvc service.ProxyService, logSvc service.LogService) *http.ServeMux {
	mux := http.NewServeMux()

	// Admin
	mux.HandleFunc("/admin/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-User-Role") != "admin" {
			http.Error(w, `{"error":"yetkisiz erişim"}`, http.StatusForbidden)
			return
		}
		logs, err := logSvc.GetRecent(100)
		if err != nil {
			http.Error(w, `{"error":"log alınamadı"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(logs)
	})

	// Auth Service
	authTarget := "http://auth-service:8081"
	mux.HandleFunc("/login",    func(w http.ResponseWriter, r *http.Request) { proxySvc.Forward(w, r, authTarget) })
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) { proxySvc.Forward(w, r, authTarget) })
	mux.HandleFunc("/users",    func(w http.ResponseWriter, r *http.Request) { proxySvc.Forward(w, r, authTarget) })

	// Event Service
	eventTarget := "http://event-service:8082"
	mux.HandleFunc("/events",  func(w http.ResponseWriter, r *http.Request) { proxySvc.Forward(w, r, eventTarget) })
	mux.HandleFunc("/events/", func(w http.ResponseWriter, r *http.Request) { proxySvc.Forward(w, r, eventTarget) })

	// Booking Service
	bookingTarget := "http://booking-service:8083"
	mux.HandleFunc("/bookings",  func(w http.ResponseWriter, r *http.Request) { proxySvc.Forward(w, r, bookingTarget) })
	mux.HandleFunc("/bookings/", func(w http.ResponseWriter, r *http.Request) { proxySvc.Forward(w, r, bookingTarget) })

	return mux
}