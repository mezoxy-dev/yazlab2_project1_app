package handler

import (
	"auth-service/models"
	"auth-service/service"
	"encoding/json"
	"net/http"
)

// AuthHandler: HTTP katmanını service katmanına bağlayan adaptör.
// İçinde hiçbir iş mantığı yoktur — JSON okur, service'e iletir, HTTP yanıtı yazar.
type AuthHandler struct {
	service service.AuthService
}

// NewAuthHandler: Constructor. Handler'ı service bağımlılığıyla oluşturur.
func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{service: svc}
}

// Register: POST /register
// Sorumluluğu: JSON parse → validasyon → service.Register → HTTP yanıtı.
// "Şifre nasıl hashlenir?" bilmez, bilmemeli.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "username ve password boş bırakılamaz",
		})
		return
	}

	if err := h.service.Register(req.Username, req.Password, req.AdminSecret); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "kayıt başarılı"})
}

// Login: POST /login
// Sorumluluğu: JSON parse → service.Login → token'ı HTTP yanıtına yaz.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// SetupRouter: Handler'ları URL rotalarına bağlar.
// Sadece "kablolama" yapar — mantık işlemez.
func SetupRouter(svc service.AuthService) *http.ServeMux {
	h := NewAuthHandler(svc)
	mux := http.NewServeMux()
	mux.HandleFunc("/register", h.Register)
	mux.HandleFunc("/login", h.Login)
	return mux
}