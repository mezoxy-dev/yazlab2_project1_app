package service_test

import (
	"auth-service/models"
	"auth-service/service"
	"errors"
	"testing"
)

// ── Mock Repository ──────────────────────────────────────────────────────────
// repository.UserStore interface'ini implement eder.
// Gerçek MongoDB'ye ihtiyaç duymadan bellekte çalışır.

type mockRepo struct {
	users map[string]models.User
}

func newMockRepo() *mockRepo {
	return &mockRepo{users: make(map[string]models.User)}
}

func (m *mockRepo) CreateUser(user models.User) error {
	if _, exists := m.users[user.Username]; exists {
		return errors.New("zaten var")
	}
	m.users[user.Username] = user
	return nil
}

func (m *mockRepo) FindByUsername(username string) (*models.User, error) {
	u, exists := m.users[username]
	if !exists {
		return nil, errors.New("bulunamadı")
	}
	return &u, nil
}

// ── Yardımcı ─────────────────────────────────────────────────────────────────

func newService() service.AuthService {
	return service.NewAuthService(newMockRepo(), []byte("test_secret"))
}

// ── Testler ───────────────────────────────────────────────────────────────────

// Service katmanı burada doğrudan çağrılıyor.
// HTTP Request/Response, router, httptest — hiçbiri yok.
// Sadece saf iş mantığı test ediliyor.

func TestRegister_Basarili(t *testing.T) {
	svc := newService()
	err := svc.Register("oguzhan", "142213", "")
	if err != nil {
		t.Fatalf("kayıt başarısız olmamalıydı, hata: %v", err)
	}
}

func TestRegister_MukerrerKayitEngelleniz(t *testing.T) {
	svc := newService()
	_ = svc.Register("oguzhan", "142213", "")

	err := svc.Register("oguzhan", "baska_sifre", "")
	if err == nil {
		t.Fatal("aynı kullanıcı adıyla ikinci kayıt engellenmeliydi")
	}
}

func TestLogin_Basarili(t *testing.T) {
	svc := newService()
	_ = svc.Register("oguzhan", "142213", "")

	token, err := svc.Login("oguzhan", "142213")
	if err != nil {
		t.Fatalf("login başarısız olmamalıydı, hata: %v", err)
	}
	if token == "" {
		t.Fatal("başarılı login'de token dönmeliydi")
	}
}

func TestLogin_HataliSifre(t *testing.T) {
	svc := newService()
	_ = svc.Register("oguzhan", "142213", "")

	_, err := svc.Login("oguzhan", "yanlis_sifre")
	if err == nil {
		t.Fatal("hatalı şifrede hata dönmeliydi")
	}
}

func TestLogin_YoksuKullanici(t *testing.T) {
	svc := newService()

	_, err := svc.Login("olmayan_kullanici", "sifre")
	if err == nil {
		t.Fatal("var olmayan kullanıcıda hata dönmeliydi")
	}
}

func TestRegister_AdminRolu(t *testing.T) {
	svc := newService()
	t.Setenv("ADMIN_REGISTRATION_SECRET", "gizli_anahtar")

	err := svc.Register("admin_user", "sifre", "gizli_anahtar")
	if err != nil {
		t.Fatalf("admin kayıt başarısız: %v", err)
	}
	// Token içindeki role claim'i doğrulamak için login yapılabilir;
	// burada sadece kayıt hatasız tamamlandı mı kontrol ediyoruz.
}