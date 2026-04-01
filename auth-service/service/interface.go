package service

// AuthService: İş mantığını içeren servis katmanı
// Handler'lar bu interface'e bağımlıdır, somut implementasyona değil.
// Bu sayede handler testlerinde gerçek bcrypt/JWT çalıştırmak yerine
// MockAuthService kullanılabilir — testler anında çalışır.
type AuthService interface {
	Register(username, password, adminSecret string) error
	Login(username, password string) (string, error)
}
