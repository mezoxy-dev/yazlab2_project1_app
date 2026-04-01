package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- 1. MODELLER VE ARAYÜZLER ---

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
	return nil, nil // Şimdilik listeleme boş
}

// --- 3. HTTP YÖNLENDİRİCİ VE SERVİSLER ARASI İLETİŞİM ---

// checkEventCapacity: Event servisine gidip etkinlik durumunu sorar
func checkEventCapacity(eventID string) error {
	eventURL := os.Getenv("EVENT_SERVICE_URL")
	internalKey := os.Getenv("INTERNAL_GATEWAY_KEY")

	if eventURL == "" {
		return errors.New("event servisi URL'i bulunamadı")
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/events/%s", eventURL, eventID), nil)
	if err != nil {
		return err
	}

	// Güvenlik duvarını geçmek için anahtarı ekliyoruz
	req.Header.Set("X-Internal-Secret", internalKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("etkinlik bulunamadı veya yer yok")
	}
	return nil
}

// updateEventCapacity: Bilet kesildikten sonra Event servisindeki kapasiteyi düşürür
func updateEventCapacity(eventID string, seats int) error {
	eventURL := os.Getenv("EVENT_SERVICE_URL")
	internalKey := os.Getenv("INTERNAL_GATEWAY_KEY")

	// Kapasiteyi düşüreceğimiz için eksi değer gönderiyoruz
	patchData := map[string]int{"amount": -seats}
	body, _ := json.Marshal(patchData)

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/events/%s", eventURL, eventID), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	// Güvenlik duvarını geçmek için anahtarı ekliyoruz
	req.Header.Set("X-Internal-Secret", internalKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("event kapasitesi güncellenemedi")
	}
	return nil
}

func NewBookingHandler(repo BookingRepository) *http.ServeMux {
	mux := http.NewServeMux()

	// Ortak işlem fonksiyonunu bir değişkene atıyoruz
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
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

			err := checkEventCapacity(b.EventID)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "` + err.Error() + `"}`))
				return
			}

			_, err = repo.CreateBooking(b)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			_ = updateEventCapacity(b.EventID, b.Seats)

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status": "bilet oluşturuldu"}`))
			return
		}
	}

	// Her iki URL varyasyonunu da aynı fonksiyona bağlıyoruz
	mux.HandleFunc("/bookings", handlerFunc)
	mux.HandleFunc("/bookings/", handlerFunc)

	return mux
}

// --- 4. ANA ÇALIŞTIRICI ---

// InternalOnlyMiddleware: Sadece Dispatcher'dan gelenlere izin ver
func InternalOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedSecret := os.Getenv("INTERNAL_GATEWAY_KEY")
		providedSecret := r.Header.Get("X-Internal-Secret")

		if expectedSecret == "" || providedSecret != expectedSecret {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "Doğrudan erişim yasaktır. Gateway üzerinden erişiniz."}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

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

	router := NewBookingHandler(repo)

	port := ":8083"
	fmt.Printf("Booking Service %s portunda aktif...\n", port)

	// Booking servisini de güvene alıyoruz
	log.Fatal(http.ListenAndServe(port, InternalOnlyMiddleware(router)))
}
