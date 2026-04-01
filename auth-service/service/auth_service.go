package service

import (
	"auth-service/models"
	"auth-service/repository"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// authService: AuthService interface'inin gerçek implementasyonu.
// Küçük harf — dışarıya constructor (NewAuthService) üzerinden açılır.
type authService struct {
	repo repository.UserStore
	key  []byte
}

// NewAuthService: Dependency Injection constructor'ı.
// Dışarıya AuthService interface'i döner; somut tipi gizler.
func NewAuthService(repo repository.UserStore, jwtSecret []byte) AuthService {
	return &authService{repo: repo, key: jwtSecret}
}

// Register: Kullanıcı kayıt iş mantığı.
// - Mükerrer kayıt kontrolü
// - Şifre hashleme (bcrypt)
// - Rol belirleme (admin_secret eşleşmesi)
// HTTP Request/Response bilmez — sadece saf Go değerleri alır/döner.
func (s *authService) Register(username, password, providedSecret string) error {
	existing, _ := s.repo.FindByUsername(username)
	if existing != nil {
		return errors.New("bu kullanıcı adı zaten alınmış")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	role := "user"
	expectedSecret := os.Getenv("ADMIN_REGISTRATION_SECRET")
	if providedSecret != "" && expectedSecret != "" && providedSecret == expectedSecret {
		role = "admin"
	}

	return s.repo.CreateUser(models.User{
		Username: username,
		Password: string(hashed),
		Role:     role,
	})
}

// Login: Kullanıcı giriş iş mantığı.
// - Kullanıcı arama
// - Şifre doğrulama (bcrypt compare)
// - JWT token üretme
// Başarılı olursa imzalı token string'i döner.
func (s *authService) Login(username, password string) (string, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return "", errors.New("kullanıcı bulunamadı")
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("hatalı şifre")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.Username,
		"role": user.Role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString(s.key)
}