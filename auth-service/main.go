package main

import (
	"context"
	"encoding/json"
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

// Modeller
type User struct {
	Username string `json:"username" bson:"username"`
	Password string `json:"password" bson:"password"`
}

// Veritabanı katmanı
type UserRepository struct {
	collection *mongo.Collection
}

func (r *UserRepository) CreateUser(user User) error {
	_, err := r.collection.InsertOne(context.TODO(), user)
	return err
}

func (r *UserRepository) FindByUsername(username string) (*User, error) {
	var user User
	err := r.collection.FindOne(context.TODO(), bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Servis katmanı
type AuthService struct {
	repo *UserRepository
	key  []byte
}

func (s *AuthService) Register(username, password string) error {
	// Şifreyi Hash'le
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return s.repo.CreateUser(User{Username: username, Password: string(hashedPassword)})
}

func (s *AuthService) Login(username, password string) (string, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return "", err
	}

	// Hash'li şifre kontrolü
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", err
	}

	// JWT Oluşturma
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.Username,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	return token.SignedString(s.key)
}



// Ana uygulama
func main() {
	// Ayarları Docker'dan oku
	mongoURI := os.Getenv("MONGO_URI")
	jwtSecret := os.Getenv("JWT_SECRET")

	client, _ := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	repo := &UserRepository{collection: client.Database("authdb").Collection("users")}
	service := &AuthService{repo: repo, key: []byte(jwtSecret)}

	// Router
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
        var u User
		// r.body
		log.Printf("DEBUG: r.Body: %v", r.Body)
        err := json.NewDecoder(r.Body).Decode(&u)
        
        if err != nil {
            log.Printf("HATA: JSON çözülemedi: %v", err)
            http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
            return
        }

        // Log: Decode sonrası veri durumu
        log.Printf("DEBUG: Decode sonrası -> Kullanıcı: [%s], Şifre: [%s]", u.Username, u.Password)
        if u.Username == "" || u.Password == "" {
            log.Println("HATA: Kullanıcı adı veya şifre boş geldi!")
            http.Error(w, "Kullanıcı adı veya şifre boş olamaz", http.StatusBadRequest)
            return
        }
        if err := service.Register(u.Username, u.Password); err != nil {
            log.Printf("HATA: Servis kayıt hatası: %v", err)
            http.Error(w, "Kayit hatasi", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"success"}`))
    })

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		var u User
		json.NewDecoder(r.Body).Decode(&u)
		token, err := service.Login(u.Username, u.Password)
		if err != nil {
			http.Error(w, "Giris hatasi", 401)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	})

	log.Println("Auth Service 8081 aktif...")
	http.ListenAndServe(":8081", nil)
}