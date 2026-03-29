package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InternalOnlyMiddleware: İsteğin sadece Dispatcher (Gateway) üzerinden geldiğini doğrular.
func InternalOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedSecret := os.Getenv("INTERNAL_GATEWAY_KEY")
		providedSecret := r.Header.Get("X-Internal-Secret")

		// Eğer anahtar boşsa veya eşleşmiyorsa erişimi engelle
		if expectedSecret == "" || providedSecret != expectedSecret {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "Doğrudan erişim yasaktır. Lütfen Gateway üzerinden erişiniz."}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Etkinlik bilgilerini tutan yapı
type Event struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"`
	Location  string             `json:"location" bson:"location"`
	Capacity  int                `json:"capacity" bson:"capacity"`
	Available int                `json:"available" bson:"available"` // Satılabilir bilet sayısı
	Date      string             `json:"date" bson:"date"`
}

// EventStore: Veritabanı işlemlerini soyutlayan arayüz (Interface)
// Bu arayüz sayesinde test dosyasında MockEventRepo kullanabiliyoruz.
type EventStore interface {
	CreateEvent(event Event) error
	GetAllEvents() ([]Event, error)
	GetEventByID(id string) (*Event, error)
}

// EventRepository: MongoDB implementasyonu
type EventRepository struct {
    collection *mongo.Collection
}

func (r *EventRepository) CreateEvent(event Event) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _, err := r.collection.InsertOne(ctx, event)
    return err
}

func (r *EventRepository) GetAllEvents() ([]Event, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    cursor, err := r.collection.Find(ctx, bson.M{})
    if err != nil {
        return nil, err
    }
    var events []Event
    err = cursor.All(ctx, &events)
    return events, err
}

func (r *EventRepository) GetEventByID(id string) (*Event, error) {
    objID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return nil, err
    }
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    var event Event
    err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&event)
    if err != nil {
        return nil, err
    }
    return &event, nil
}

// EventService: Testlerdeki 'service := &EventService{Repo: ...}' yapısına uygun
type EventService struct {
	Repo EventStore
}

func setupRouter(service *EventService) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		// GET: Etkinlikleri Listele
		if r.Method == http.MethodGet {
			events, err := service.Repo.GetAllEvents()
			if err != nil {
				http.Error(w, "Veriler okunamadı", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(events)
			return
		}

		// POST: Etkinlik Ekle (Sadece Admin)
		if r.Method == http.MethodPost {
			// Dispatcher'dan gelen Header kontrolü (RBAC)
			role := r.Header.Get("X-User-Role")
			if role != "admin" {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error": "Yetkisiz işlem: Admin yetkisi gerekli"}`))
				return
			}

			var e Event
			if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
				http.Error(w, "Geçersiz veri", http.StatusBadRequest)
				return
			}

			// İş Mantığı: Yeni bir etkinlikte Available başlangıçta Capacity'e eşittir.
			if e.Available == 0 {
				e.Available = e.Capacity
			}

			if err := service.Repo.CreateEvent(e); err != nil {
				http.Error(w, "Kaydetme hatası", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"message": "Etkinlik başarıyla oluşturuldu"})
			return
		}

		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	})

	return mux
}

func main() {
	// Ortam değişkenlerinden ayarları oku
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://db-event:27017"
	}

	// MongoDB Bağlantısı
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("MongoDB bağlantı hatası:", err)
	}

	// Katmanları başlat
	repo := &EventRepository{collection: client.Database("eventdb").Collection("events")}
	service := &EventService{Repo: repo}
	router := setupRouter(service)

	log.Println("Event Service 8082 portunda aktif...")
	if err := http.ListenAndServe(":8082", InternalOnlyMiddleware(router)); err != nil {
		log.Fatal(err)
	}
}