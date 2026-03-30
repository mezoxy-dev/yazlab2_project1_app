package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)


// Etkinlik bilgilerini tutan yapı
type Event struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"`
	Location  string             `json:"location" bson:"location"`
	Capacity  int                `json:"capacity" bson:"capacity"`
	Available int                `json:"available" bson:"available"` 
	Date      string             `json:"date" bson:"date"`
}

// EventStore: Veritabanı işlemlerini soyutlayan arayüz (Interface)
// Bu arayüz sayesinde test dosyasında MockEventRepo kullanabiliyoruz.
type EventStore interface {
	CreateEvent(event Event) error
	GetAllEvents() ([]Event, error)
	GetEventByID(id string) (*Event, error)
	UpdateEvent(id string, event Event) error
	DeleteEvent(id string) error
	UpdateAvailableTickets(id string, amount int) error
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

func (r *EventRepository) UpdateEvent(id string, event Event) error {
	objID, err := primitive.ObjectIDFromHex(id) 
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"name":     event.Name,
			"location": event.Location,
			"capacity": event.Capacity,
			"date":     event.Date,
		},
	}
	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}



func (r *EventRepository) UpdateAvailableTickets(id string, amount int) error {
    objID, _ := primitive.ObjectIDFromHex(id)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // $inc operatörü ile bilet sayısını azaltıyoruz (-1)
    filter := bson.M{"_id": objID, "available": bson.M{"$gt": 0}} // Kontenjan 0'dan büyükse
    update := bson.M{"$inc": bson.M{"available": amount}}
    
    result, err := r.collection.UpdateOne(ctx, filter, update)
    if err != nil {
        return err
    }
    if result.MatchedCount == 0 {
        return errors.New("etkinlik bulunamadı veya kontenjan yetersiz")
    }
    return nil
}

func (r *EventRepository) DeleteEvent(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

func getEventID(r *http.Request) string {
	// Önce path e bakılır: /events/65f... 
	id := strings.TrimPrefix(r.URL.Path, "/events/")
	if id != "" && id != "/events" {
		return id
	}
	// Eğer Path boşsa Query Parameter'a bakıyoruz: ?id=65f...	
	return r.URL.Query().Get("id")
}

// EventService: Testlerdeki 'service := &EventService{Repo: ...}' yapısına uygun
type EventService struct {
	Repo EventStore
}

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


func setupRouter(service *EventService) *http.ServeMux {
	mux := http.NewServeMux()

	handler := func(w http.ResponseWriter, r *http.Request) {
		id := getEventID(r)

		if id == "" {
			if r.Method == http.MethodGet {
				events, err := service.Repo.GetAllEvents()
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(events)
				return
			}
			if r.Method == http.MethodPost {
				if r.Header.Get("X-User-Role") != "admin" {
					http.Error(w, "Admin yetkisi gerekli", 403)
					return
				}
				var e Event
				json.NewDecoder(r.Body).Decode(&e)
				e.Available = e.Capacity
				service.Repo.CreateEvent(e)
				w.WriteHeader(http.StatusCreated)
				return
			}
		}

		if id != "" {
			switch r.Method {
			case http.MethodGet:
				event, err := service.Repo.GetEventByID(id)
				if err != nil {
					http.Error(w, "Bulunamadi", 404)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(event)
			case http.MethodPut:
				if r.Header.Get("X-User-Role") != "admin" {
					http.Error(w, "Yetkisiz", 403)
					return
				}
				var e Event
				json.NewDecoder(r.Body).Decode(&e)
				service.Repo.UpdateEvent(id, e)
				w.Write([]byte(`{"message": "Guncellendi"}`))
			case http.MethodDelete:
				if r.Header.Get("X-User-Role") != "admin" {
					http.Error(w, "Yetkisiz", 403)
					return
				}
				service.Repo.DeleteEvent(id)
				w.WriteHeader(http.StatusNoContent)
			case http.MethodPatch:
				var body struct{ Amount int `json:"amount"` }
				json.NewDecoder(r.Body).Decode(&body)
				if err := service.Repo.UpdateAvailableTickets(id, body.Amount); err != nil {
					http.Error(w, err.Error(), 400)
					return
				}
				w.Write([]byte(`{"message": "Kontenjan guncellendi"}`))
			default:
				http.Error(w, "Method Not Allowed", 405)
			}
			return
		}

		http.Error(w, "Gecersiz istek veya ID eksik", 400)
	}

	mux.HandleFunc("/events", handler)
	mux.HandleFunc("/events/", handler)

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