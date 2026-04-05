package service

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// ProxyService interface'inin gerçek implementasyonu.
// Gelen isteği hedef servise iletir ve X-Internal-Secret header'ını ekler.
type proxyService struct {
	authSvc AuthService
}

// NewProxyService: Constructor. InternalSecret'ı AuthService'ten alır.
func NewProxyService(authSvc AuthService) ProxyService {
	return &proxyService{authSvc: authSvc}
}

// Gelen HTTP isteğini targetURL'e reverse proxy ile iletir.
// Her isteğe X-Internal-Secret header'ı eklenir; downstream servisler
// bu header olmadan isteği reddeder (InternalOnly middleware).
func (s *proxyService) Forward(w http.ResponseWriter, r *http.Request, targetURL string) {
	dest, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, "geçersiz hedef URL", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(dest)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Internal-Secret", s.authSvc.InternalSecret())
	}

	proxy.ServeHTTP(w, r)
}