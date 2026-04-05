package main

import (
	"context"
	"event-service/handler"
	"event-service/middleware"
	"event-service/repository"
	"event-service/service"
	"log"
	"net/http"
	"os"
	"time"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB → EventRepository → EventService → EventHandler → Router → Server
// Her katman bir interface üzerinden üst katmana bağımlıdır, somut implementasyonlara değil.
// EventHandler sadece EventService interface'ini görür, gerçek EventService implementasyonunu değil.
func main() {
	mongoURI := os.Getenv("MONGO_URI")
 
	if mongoURI == "" {
		log.Fatal("MONGO_URI eksik")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // MongoDB bağlantısı için zaman aşımı süresi
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("MongoDB bağlantı hatası:", err)
	}

	repo   := repository.NewEventRepository(client.Database("eventdb").Collection("events"))
	svc    := service.NewEventService(repo)
	router := handler.SetupRouter(svc)

	log.Println("Event Service :8082 portunda başlatılıyor...")
	if err := http.ListenAndServe(":8082", middleware.InternalOnly(router)); err != nil {
		log.Fatalf("Sunucu hatası: %v", err)
	}
}
 