package main

import (
	"booking-service/handler"
	"booking-service/middleware"
	"booking-service/repository"
	"booking-service/service"
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB bağlantı hatası: %v", err)
	}

	repo        := repository.NewMongoBookingRepository(client.Database("booking_db"))
	eventClient := service.NewEventClient()          
	svc         := service.NewBookingService(repo, eventClient)
	router      := handler.SetupRouter(svc)

	log.Println("Booking Service :8083 portunda başlatılıyor...")
	if err := http.ListenAndServe(":8083", middleware.InternalOnly(router)); err != nil {
		log.Fatalf("Sunucu hatası: %v", err)
	}
}