package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	jwtKey = []byte("gizli_anahtar_oguzhan")
	client *mongo.Client
)

// MongoDB Bağlantısı (İsterlerde her servisin kendi DB'si olacak denmişti)
func initDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Docker Compose'daki servis ismini kullanıyoruz: db-auth
	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://db-auth:27017"))
	if err != nil {
		log.Fatal("MongoDB bağlantı hatası:", err)
	}
	log.Println("Auth-Service MongoDB'ye bağlandı.")
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// TDD ve Hızlı Test için: Şimdilik kullanıcıyı "admin" varsayıyoruz
	// İleride burası MongoDB'den kullanıcıyı çekecek şekilde güncellenebilir.
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		Subject:   "oguzhan_erbil",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
		"status": "success",
	})
}

func main() {
	initDB()
	http.HandleFunc("/login", LoginHandler)
	log.Println("Auth Service 8081 portunda aktif...")
	log.Fatal(http.ListenAndServe(":8081", nil))
}