package service

import (
	"dispatcher/models"
	"net/http"
)

type LogService interface {
	RecordAsync(method, path, ip string, status int, durationMs int64)
	GetRecent(limit int64) ([]models.TrafficLog, error)
}

type AuthService interface {
	ValidateToken(tokenString string) (username, role string, err error)
	IsPublicRoute(path string) bool
	// ExtractToken: "Bearer xxx" formatından token string'i çıkarır.
	ExtractToken(authHeader string) string
	// InternalSecret: Downstream servislere gönderilecek secret'ı döndürür.
	InternalSecret() string
}


type ProxyService interface {
	// Gelen isteği hedef URL'e iletir.
	Forward(w http.ResponseWriter, r *http.Request, targetURL string)
}