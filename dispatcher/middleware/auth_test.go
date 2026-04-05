package middleware_test

import (
	"dispatcher/middleware"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock AuthService

type mockAuthService struct {
	validateUser string
	validateRole string
	validateErr  error
	publicRoutes map[string]bool
	secret       string
}

func (m *mockAuthService) ValidateToken(token string) (string, string, error) {
	return m.validateUser, m.validateRole, m.validateErr
}
func (m *mockAuthService) IsPublicRoute(path string) bool {
	return m.publicRoutes[path]
}
func (m *mockAuthService) ExtractToken(header string) string {
	if len(header) > 7 && header[:7] == "Bearer " {
		return header[7:]
	}
	return header
}
func (m *mockAuthService) InternalSecret() string { return m.secret }

// Yardımcı

func newMockAuth(user, role string, err error, publicPaths ...string) *mockAuthService {
	routes := map[string]bool{}
	for _, p := range publicPaths {
		routes[p] = true
	}
	return &mockAuthService{
		validateUser: user,
		validateRole: role,
		validateErr:  err,
		publicRoutes: routes,
	}
}

func doRequest(handler http.Handler, method, path, authHeader string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// Testler

func TestAuth_PublicRotaTokensizGecer(t *testing.T) {
	authSvc := newMockAuth("", "", nil, "/login", "/register")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := middleware.Auth(authSvc)(next)

	for _, path := range []string{"/login", "/register"} {
		rr := doRequest(handler, "POST", path, "")
		if rr.Code != 200 {
			t.Errorf("%s tokensız 200 beklendi, alınan %d", path, rr.Code)
		}
	}
}

func TestAuth_KorunanRotaTokensiz401(t *testing.T) {
	authSvc := newMockAuth("", "", nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := middleware.Auth(authSvc)(next)

	rr := doRequest(handler, "GET", "/events", "")
	if rr.Code != 401 {
		t.Errorf("tokensız korumalı rota 401 beklendi, alınan %d", rr.Code)
	}
}

func TestAuth_GecersizToken401(t *testing.T) {
	authSvc := newMockAuth("", "", errors.New("geçersiz"))
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := middleware.Auth(authSvc)(next)

	rr := doRequest(handler, "GET", "/events", "Bearer yanlis_token")
	if rr.Code != 401 {
		t.Errorf("geçersiz token 401 beklendi, alınan %d", rr.Code)
	}
}

func TestAuth_GecerliTokenRolHeaderaEklenir(t *testing.T) {
	authSvc := newMockAuth("oguzhan", "admin", nil)
	var capturedRole, capturedUser string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRole = r.Header.Get("X-User-Role")
		capturedUser = r.Header.Get("X-User-Name")
		w.WriteHeader(200)
	})
	handler := middleware.Auth(authSvc)(next)

	rr := doRequest(handler, "GET", "/events", "Bearer gecerli_token")
	if rr.Code != 200 {
		t.Errorf("geçerli token 200 beklendi, alınan %d", rr.Code)
	}
	if capturedRole != "admin" {
		t.Errorf("X-User-Role: beklenen 'admin', alınan '%s'", capturedRole)
	}
	if capturedUser != "oguzhan" {
		t.Errorf("X-User-Name: beklenen 'oguzhan', alınan '%s'", capturedUser)
	}
}

func TestAuth_GecerliTokenUserRolu(t *testing.T) {
	authSvc := newMockAuth("ali", "user", nil)
	var capturedRole string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRole = r.Header.Get("X-User-Role")
		w.WriteHeader(200)
	})
	handler := middleware.Auth(authSvc)(next)

	doRequest(handler, "GET", "/bookings", "Bearer gecerli_token")
	if capturedRole != "user" {
		t.Errorf("X-User-Role: beklenen 'user', alınan '%s'", capturedRole)
	}
}