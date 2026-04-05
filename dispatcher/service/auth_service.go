package service

import (
	"errors"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type authService struct{}

func NewAuthService() AuthService {
	return &authService{}
}

func (s *authService) ValidateToken(tokenString string) (string, string, error) {
	secret := os.Getenv("JWT_SECRET")

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("beklenmeyen imzalama yöntemi")
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return "", "", errors.New("geçersiz token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("claims okunamadı")
	}

	username, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)
	return username, role, nil
}

func (s *authService) IsPublicRoute(path string) bool {
	publicRoutes := map[string]bool{
		"/login":    true,
		"/register": true,
	}
	return publicRoutes[path]
}

func (s *authService) ExtractToken(authHeader string) string {
	return strings.TrimPrefix(authHeader, "Bearer ")
}

// Downstream servislere gönderilecek gateway secret'ını döndürür.
func (s *authService) InternalSecret() string {
	return os.Getenv("INTERNAL_GATEWAY_KEY")
}