package service_test

import (
	"dispatcher/service"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "gizli_anahtar_test"

func generateToken(username, role string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  username,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	})
	ts, _ := token.SignedString([]byte(testSecret))
	return ts
}

func newAuthService(t *testing.T) service.AuthService {
	t.Helper()
	os.Setenv("JWT_SECRET", testSecret)
	return service.NewAuthService()
}

// ValidateToken

func TestValidateToken_GecerliToken(t *testing.T) {
	svc := newAuthService(t)
	token := generateToken("oguzhan", "admin")

	username, role, err := svc.ValidateToken(token)

	if err != nil {
		t.Fatalf("hata olmamalıydı: %v", err)
	}
	if username != "oguzhan" {
		t.Errorf("beklenen 'oguzhan', alınan '%s'", username)
	}
	if role != "admin" {
		t.Errorf("beklenen 'admin', alınan '%s'", role)
	}
}

func TestValidateToken_YanlisSecret(t *testing.T) {
	svc := newAuthService(t)

	// Farklı secret ile üretilmiş token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "test"})
	ts, _ := token.SignedString([]byte("yanlis_secret"))

	_, _, err := svc.ValidateToken(ts)
	if err == nil {
		t.Error("yanlış secret ile token geçmemeli")
	}
}

func TestValidateToken_SuresinDolmus(t *testing.T) {
	svc := newAuthService(t)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "oguzhan",
		"role": "user",
		"exp":  time.Now().Add(-time.Hour).Unix(), // geçmiş zaman
	})
	ts, _ := token.SignedString([]byte(testSecret))

	_, _, err := svc.ValidateToken(ts)
	if err == nil {
		t.Error("süresi dolmuş token geçmemeli")
	}
}

func TestValidateToken_BosToken(t *testing.T) {
	svc := newAuthService(t)
	_, _, err := svc.ValidateToken("")
	if err == nil {
		t.Error("boş token geçmemeli")
	}
}

func TestValidateToken_RolDogru(t *testing.T) {
	svc := newAuthService(t)

	for _, role := range []string{"user", "admin"} {
		token := generateToken("test", role)
		_, gotRole, err := svc.ValidateToken(token)
		if err != nil {
			t.Fatalf("role=%s için hata: %v", role, err)
		}
		if gotRole != role {
			t.Errorf("beklenen role '%s', alınan '%s'", role, gotRole)
		}
	}
}

// IsPublicRoute

func TestIsPublicRoute_PublicRotalar(t *testing.T) {
	svc := newAuthService(t)

	for _, path := range []string{"/login", "/register"} {
		if !svc.IsPublicRoute(path) {
			t.Errorf("%s public olmalıydı", path)
		}
	}
}

func TestIsPublicRoute_KorunanRotalar(t *testing.T) {
	svc := newAuthService(t)

	for _, path := range []string{"/events", "/bookings", "/admin/stats", "/users"} {
		if svc.IsPublicRoute(path) {
			t.Errorf("%s korumalı olmalıydı", path)
		}
	}
}

// ExtractToken

func TestExtractToken_BearerPrefix(t *testing.T) {
	svc := newAuthService(t)

	cases := []struct{ input, want string }{
		{"Bearer abc123", "abc123"},
		{"abc123", "abc123"},       // prefix yoksa olduğu gibi döner
		{"Bearer ", ""},            // sadece prefix
	}
	for _, c := range cases {
		got := svc.ExtractToken(c.input)
		if got != c.want {
			t.Errorf("input=%q: beklenen %q, alınan %q", c.input, c.want, got)
		}
	}
}

// InternalSecret

func TestInternalSecret_EnvdenOkunmali(t *testing.T) {
	os.Setenv("INTERNAL_GATEWAY_KEY", "test_key_123")
	svc := service.NewAuthService()

	if svc.InternalSecret() != "test_key_123" {
		t.Errorf("beklenen 'test_key_123', alınan '%s'", svc.InternalSecret())
	}
}

// Entegrasyon: ValidateToken + ExtractToken birlikte

func TestExtractAndValidate_BearerFlow(t *testing.T) {
	svc := newAuthService(t)
	rawToken := generateToken("oguzhan", "user")
	bearerHeader := fmt.Sprintf("Bearer %s", rawToken)

	extracted := svc.ExtractToken(bearerHeader)
	username, role, err := svc.ValidateToken(extracted)

	if err != nil {
		t.Fatalf("hata olmamalıydı: %v", err)
	}
	if username != "oguzhan" || role != "user" {
		t.Errorf("beklenen oguzhan/user, alınan %s/%s", username, role)
	}
}