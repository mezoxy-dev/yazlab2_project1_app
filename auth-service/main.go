package main
 
import (
	"auth-service/handler"
	"auth-service/middleware"
	"auth-service/repository"
	"auth-service/service"
	"context"
	"log"
	"net/http"
	"os"
 
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//	MongoDB  →  UserRepository  →  AuthService  →  AuthHandler  →  Router  →  Server
// Main bağımlılıkları birbirine bağlayacak
func main() {
	mongoURI := os.Getenv("MONGO_URI")
	jwtSecret := os.Getenv("JWT_SECRET")

	if mongoURI == "" || jwtSecret == "" {
		log.Fatal("MONGO_URI veya JWT_SECRET eksik")
	}

	// MongoDB Bağlantısı
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("MongoDB bağlantı hatası:", err)
	}

	// Katmanları birbirine bağlama
	repo := repository.NewUserRepository(client.Database("authdb").Collection("users"))
	service := service.NewAuthService(repo, []byte(jwtSecret))
	router := handler.SetupRouter(service)

	// Middleware ile sar ve HTTP sunucusunu başlat
	// Middleware gelen isteklere JWT doğrulaması yapar, geçerli token yoksa 401 döner

	log.Println("Auth Service 8081 portunda başlatılıyor")
	if err := http.ListenAndServe(":8081",middleware.InternalOnly(router)); err != nil {
		log.Fatalf("Sunucu hatası: %v", err)
	}
}