package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// Sadece Dispatcher'dan gelen isteklere izin veren middleware
func InternalOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := os.Getenv("INTERNAL_GATEWAY_KEY")
		provided := r.Header.Get("X-Internal-Secret")

		if expected == "" || provided != expected {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "Doğrudan erişim yasaktır. Lütfen Gateway üzerinden erişiniz."}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Modeller
type User struct {
	Username string `json:"username" bson:"username"`
	Password string `json:"password" bson:"password"`
	Role     string `json:"role" bson:"role"`
}

// RegisterRequest: Kayıt isteği için özel yapı. 
// JSON'daki "admin_secret" alanını yakalamak için şarttır.
type RegisterRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	AdminSecret string `json:"admin_secret"` 
}

// UserStore: Test edilebilirliği sağlayan arayüz (Interface)
type UserStore interface {
	CreateUser(user User) error
	FindByUsername(username string) (*User, error)
}

// UserRepository: Gerçek MongoDB implementasyonu
type UserRepository struct {
	collection *mongo.Collection
}

func (r *UserRepository) CreateUser(user User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.collection.InsertOne(ctx, user)
	return err
}

func (r *UserRepository) FindByUsername(username string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var user User
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// AuthService
type AuthService struct {
	repo UserStore
	key  []byte
}

func (s *AuthService) Register(username, password, providedSecret string) error {
	existingUser, _ := s.repo.FindByUsername(username)
	if existingUser != nil {
		return errors.New("bu kullanıcı adı zaten alınmış")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Rol Belirleme Mantığı
	role := "user"
	expectedAdminSecret := os.Getenv("ADMIN_REGISTRATION_SECRET")

	// Eşleşme kontrolü
	if providedSecret != "" && expectedAdminSecret != "" && providedSecret == expectedAdminSecret {
		role = "admin"
	}

	return s.repo.CreateUser(User{
		Username: username,
		Password: string(hashedPassword),
		Role:     role,
	})
}

func (s *AuthService) Login(username, password string) (string, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return "", errors.New("kullanıcı bulunamadı")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("hatalı şifre")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.Username,
		"role": user.Role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})
	return token.SignedString(s.key)
}

// SetupRouter: Testlerin ve ana uygulamanın ortak kullandığı router yapısı
func SetupRouter(service *AuthService) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// --- DÜZELTME: Veriyi RegisterRequest ile karşılıyoruz ---
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Password == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Username ve password alanları boş bırakılamaz"}`))
			return
		}

		// --- DÜZELTME: Servise req.AdminSecret gönderiyoruz ---
		if err := service.Register(req.Username, req.Password, req.AdminSecret); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "success"})
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		var u User
		json.NewDecoder(r.Body).Decode(&u)
		token, err := service.Login(u.Username, u.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	})

	return mux
}

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	jwtSecret := os.Getenv("JWT_SECRET")

	if mongoURI == "" || jwtSecret == "" {
		log.Fatal("MONGO_URI veya JWT_SECRET eksik!")
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	repo := &UserRepository{collection: client.Database("authdb").Collection("users")}
	service := &AuthService{repo: repo, key: []byte(jwtSecret)}

	router := SetupRouter(service)

	log.Println("Auth Service 8081 aktif...")
	http.ListenAndServe(":8081", InternalOnlyMiddleware(router))
}