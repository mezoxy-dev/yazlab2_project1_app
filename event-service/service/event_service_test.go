package service_test

import (
	"event-service/models"
	"event-service/service"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Mock Repository

type mockRepo struct {
	events []models.Event
}

func (m *mockRepo) CreateEvent(e models.Event) error {
	if e.ID.IsZero() {
		e.ID = primitive.NewObjectID()
	}
	m.events = append(m.events, e)
	return nil
}

func (m *mockRepo) GetAllEvents() ([]models.Event, error) {
	return m.events, nil
}

func (m *mockRepo) GetEventByID(id string) (*models.Event, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	for _, e := range m.events {
		if e.ID == objID {
			return &e, nil
		}
	}
	return nil, fmt.Errorf("bulunamadı")
}

func (m *mockRepo) UpdateEvent(id string, event models.Event) error {
	objID, _ := primitive.ObjectIDFromHex(id)
	for i, e := range m.events {
		if e.ID == objID {
			m.events[i].Name = event.Name
			m.events[i].Location = event.Location
			m.events[i].Capacity = event.Capacity
			m.events[i].Date = event.Date
			return nil
		}
	}
	return fmt.Errorf("bulunamadı")
}

func (m *mockRepo) DeleteEvent(id string) error {
	objID, _ := primitive.ObjectIDFromHex(id)
	for i, e := range m.events {
		if e.ID == objID {
			m.events = append(m.events[:i], m.events[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("bulunamadı")
}

func (m *mockRepo) UpdateAvailableTickets(id string, amount int) error {
	objID, _ := primitive.ObjectIDFromHex(id)
	for i, e := range m.events {
		if e.ID == objID {
			if e.Available+amount < 0 {
				return fmt.Errorf("kontenjan yetersiz")
			}
			m.events[i].Available += amount
			return nil
		}
	}
	return fmt.Errorf("bulunamadı")
}

// Yardımcı

func newService() (service.EventService, *mockRepo) {
	repo := &mockRepo{}
	return service.NewEventService(repo), repo
}

// Testler

func TestCreateEvent_Basarili(t *testing.T) {
	svc, repo := newService()

	err := svc.CreateEvent(models.Event{
		Name:     "Büyük Festival",
		Location: "Kocaeli",
		Capacity: 500,
		Date:     "2026-05-15",
	})

	if err != nil {
		t.Fatalf("Kayıt başarısız olmamalıydı: %v", err)
	}
	if len(repo.events) != 1 {
		t.Fatalf("Etkinlik kaydedilmedi")
	}
}

func TestCreateEvent_AvailableCapacityeEsitOlmali(t *testing.T) {
	svc, repo := newService()
	_ = svc.CreateEvent(models.Event{Name: "Fest", Location: "İzmir", Capacity: 200})

	if repo.events[0].Available != 200 {
		t.Errorf("Available capacity'e eşit olmalıydı, alınan: %d", repo.events[0].Available)
	}
}

func TestCreateEvent_EksikVeriReddedilmeli(t *testing.T) {
	svc, _ := newService()

	cases := []models.Event{
		{Name: "", Location: "Kocaeli", Capacity: 100},  // isim boş
		{Name: "Fest", Location: "", Capacity: 100},     // konum boş
		{Name: "Fest", Location: "Kocaeli", Capacity: 0}, // kapasite sıfır
	}

	for _, c := range cases {
		if err := svc.CreateEvent(c); err == nil {
			t.Errorf("Eksik veriyle kayıt yapılmamalıydı: %+v", c)
		}
	}
}

func TestGetAllEvents(t *testing.T) {
	svc, _ := newService()
	_ = svc.CreateEvent(models.Event{Name: "A", Location: "Bursa", Capacity: 50})
	_ = svc.CreateEvent(models.Event{Name: "B", Location: "Ankara", Capacity: 80})

	events, err := svc.GetAllEvents()
	if err != nil {
		t.Fatalf("Hata oluşmamalıydı: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("2 etkinlik beklendi, %d alındı", len(events))
	}
}

func TestGetEventByID_Basarili(t *testing.T) {
	svc, repo := newService()
	_ = svc.CreateEvent(models.Event{Name: "Konser", Location: "İstanbul", Capacity: 1000})

	id := repo.events[0].ID.Hex()
	event, err := svc.GetEventByID(id)

	if err != nil {
		t.Fatalf("Hata oluşmamalıydı: %v", err)
	}
	if event.Name != "Konser" {
		t.Errorf("Beklenen 'Konser', alınan '%s'", event.Name)
	}
}

func TestGetEventByID_YanlisID(t *testing.T) {
	svc, _ := newService()
	_, err := svc.GetEventByID("gecersiz_id")
	if err == nil {
		t.Error("Geçersiz ID'de hata dönmeliydi")
	}
}

func TestUpdateAvailableTickets_Azaltma(t *testing.T) {
	svc, repo := newService()
	_ = svc.CreateEvent(models.Event{Name: "Fest", Location: "İzmir", Capacity: 100})

	id := repo.events[0].ID.Hex()
	err := svc.UpdateAvailableTickets(id, -1)

	if err != nil {
		t.Fatalf("Bilet azaltma başarısız: %v", err)
	}
	if repo.events[0].Available != 99 {
		t.Errorf("Available 99 olmalıydı, alınan: %d", repo.events[0].Available)
	}
}

func TestUpdateAvailableTickets_KontenjanYetersiz(t *testing.T) {
	svc, repo := newService()
	_ = svc.CreateEvent(models.Event{Name: "Fest", Location: "İzmir", Capacity: 1})

	id := repo.events[0].ID.Hex()
	_ = svc.UpdateAvailableTickets(id, -1) // 1 → 0

	err := svc.UpdateAvailableTickets(id, -1) // 0 → hata vermeli
	if err == nil {
		t.Error("Kontenjan dolduğunda hata dönmeliydi")
	}
}

func TestDeleteEvent(t *testing.T) {
	svc, repo := newService()
	_ = svc.CreateEvent(models.Event{Name: "Fest", Location: "İzmir", Capacity: 50})

	id := repo.events[0].ID.Hex()
	err := svc.DeleteEvent(id)

	if err != nil {
		t.Fatalf("Silme başarısız: %v", err)
	}
	if len(repo.events) != 0 {
		t.Error("Etkinlik silinmedi")
	}
}