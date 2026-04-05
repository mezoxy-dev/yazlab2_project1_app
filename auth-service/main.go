package main
 
import (
	"auth-service/middleware"
	"auth-service/repository"
	"auth-service/service"
	"context"
	"log"
	"net/http"
	"os"
	"time"
 
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//	MongoDB  →  UserRepository  →  AuthService  →  AuthHandler  →  Router  →  Server
// Main bağımlılıkları birbirine bağlayacak
func main() {
	mongoURI := os.Getenv("MONGO_URI")
	jwtSecret := os.Getenv("JWT_SECRET") // JWT Key üretmek için gerekli

	if mongoURI == "" || jwtSecret == "" {
		log.Fatal("MONGO_URI veya JWT_SECRET eksik")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// MongoDB Bağlantısı
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("MongoDB bağlantı hatası:", err)
	}

	// Katmanları birbirine bağlama

	// repo: authdb içerisinde users tablosunu işaret ediyoruz, 
	// NewUserRepository fonksiyonu ile UserRepository oluşturuyoruz, 
	// daha sonra uygulama "yeni kullanıcı kaydet (yazma)" ve "kullanıcı bul (okuma)" işlemleri geldiğinde 
	// doğru tabloda işlemleri gerçekleştirir.
	userColl := client.Database("authdb").Collection("users")
	
	// Username için benzersiz index oluşturma (Hem performans hem de mükerrer kayıt engelleme)
	_, _ = userColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	repo := repository.NewUserRepository(userColl) 
	// Uygulama içi kuralları yönetir, veritabanı işlemleri yapabilmesi için repo verilir, 
	// başarılı giriş yapan kullanıcıların jwt keyi de verilir diğer servislerin doğrulama yapabilmesi için
	service := service.NewAuthService(repo, []byte(jwtSecret))
	// Web üzerinden gelen istekleri yönlendirir, /login ve /register adresine yaptığı HTTP isteklerini ilgili servislere bağlar
	router := SetupRouter(service)

	// Middleware ile sar ve HTTP sunucusunu başlat
	// Middleware gelen isteklere JWT doğrulaması yapar, geçerli token yoksa 401 döner

	log.Println("Auth Service 8081 portunda başlatılıyor")
	// Web sunucudan gelen istekleri yönlendirmeden önce InternalOnly güvenliği uygular
	// InternalOnly servisler arası güvenliği sağlamak için var, dışarıdan gelen istekleri engeller
	// Internal Key dispatcher dan gelir
	if err := http.ListenAndServe(":8081",middleware.InternalOnly(router)); err != nil {
		log.Fatalf("Sunucu hatası: %v", err)
	}
}