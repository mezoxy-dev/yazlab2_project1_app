package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- 1. MODELLER VE ARAYÜZLER (OOP İsteri) ---

type Booking struct {
	ID      string `json:"id,omitempty" bson:"_id,omitempty"`
	EventID string `json:"event_id" bson:"event_id"`
	UserID  string `json:"user_id" bson:"user_id"`
	Seats   int    `json:"seats" bson:"seats"`
}

type BookingRepository interface {
	CreateBooking(booking Booking) (string, error)
	GetBookingsByUser(userID string) ([]Booking, error)
}

// --- 2. MONGODB IMPLEMENTASYONU ---

type mongoBookingRepo struct {
	db *mongo.Database
}

func (m *mongoBookingRepo) CreateBooking(booking Booking) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := m.db.Collection("bookings")
	res, err := collection.InsertOne(ctx, booking)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", res.InsertedID), nil
}

func (m *mongoBookingRepo) GetBookingsByUser(userID string) ([]Booking, error) {
	return nil, nil // Şimdilik boş bırakıyoruz
}

// --- 3. HTTP YÖNLENDİRİCİ (RMM Seviye 2 Uyumlu) ---

func NewBookingHandler(repo BookingRepository) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/bookings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodPost {
			var b Booking
			if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "geçersiz JSON formatı"}`))
				return
			}

			if b.EventID == "" || b.UserID == "" || b.Seats <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "eksik veya hatalı bilgi"}`))
				return
			}

			// Proje İsteri: Mikroservislerin birbiriyle haberleşmesi (Şimdilik yoruma kapalı)
			/*
				eventURL := os.Getenv("EVENT_SERVICE_URL")
				if eventURL != "" {
					resp, err := http.Get(fmt.Sprintf("%s/events/%s", eventURL, b.EventID))
					if err != nil || resp.StatusCode != http.StatusOK {
						w.WriteHeader(http.StatusBadRequest)
						w.Write([]byte(`{"error": "etkinlik bulunamadı veya ulaşılamıyor"}`))
						return
					}
				}
			*/

			_, err := repo.CreateBooking(b)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated) // 201 Created (Kural)
			w.Write([]byte(`{"status": "bilet oluşturuldu"}`))
			return
		}
	})

	return mux
}

// --- 4. ANA ÇALIŞTIRICI ---

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://db-booking:27017"
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	db := client.Database("booking_db")
	repo := &mongoBookingRepo{db: db}

	mux := NewBookingHandler(repo)

	port := ":8083"
	fmt.Printf("Booking Service %s portunda çalışıyor...\n", port)
	log.Fatal(http.ListenAndServe(port, mux))
}
