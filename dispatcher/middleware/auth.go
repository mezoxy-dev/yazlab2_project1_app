package middleware

import (
	"dispatcher/service"
	"net/http"
)

// Middleware sadece şunu yapar:
//   1. Public route mu? → geç
//   2. Header var mı? → yoksa 401
//   3. Token geçerli mi? → AuthService sorar; geçersizse 401
//   4. Geçerliyse username ve role'ü downstream'e header olarak ilet

func Auth(authSvc service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// /login ve /register JWT istemez
			if authSvc.IsPublicRoute(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.WriteHeader(http.StatusUnauthorized) // 401
				return
			}

			tokenString := authSvc.ExtractToken(authHeader)
			username, role, err := authSvc.ValidateToken(tokenString)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized) // 401
				return
			}

			r.Header.Set("X-User-Name", username)
			r.Header.Set("X-User-Role", role)

			next.ServeHTTP(w, r)
		})
	}
}