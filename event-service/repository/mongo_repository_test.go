package repository_test

import (
	"event-service/models"
	"event-service/repository"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ── In-Memory Repository ──────────────────────────────────────────────────────
// EventStore interface'ini implement eden saf bellekte çalışan versiyon.
//
// Bu dosyanın amacı: EventStore interface'inin sözleşmesini doğrulamak.
// "Interface'i implement eden her struct doğru davranıyor mu?"
//
// NOT: MongoRepository'nin gerçek MongoDB ile testi integration test sayılır
// ve CI ortamında çalışan bir MongoDB gerektirir. Bu testler ise hiçbir
// dış bağımlılık olmadan çalışır — Dockerfile'da `go test ./...` güvenle geçer.

type inMemoryRepo struct {
	events []models.Event
}

func newInMemoryRepo() repository.EventStore {
	return &inMemoryRepo{}
}

func (r *inMemoryRepo) CreateEvent(e models.Event) error {
	if e.ID.IsZero() {
		e.ID = primitive.NewObjectID()
	}
	r.events = append(r.events, e)
	return nil
}

func (r *inMemoryRepo) GetAllEvents() ([]models.Event, error) {
	return r.events, nil
}

func (r *inMemoryRepo) GetEventByID(id string) (*models.Event, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("geçersiz ID formatı: %w", err)
	}
	for _, e := range r.events {
		if e.ID == objID {
			return &e, nil
		}
	}
	return nil, fmt.Errorf("etkinlik bulunamadı")
}

func (r *inMemoryRepo) UpdateEvent(id string, updated models.Event) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("geçersiz ID formatı: %w", err)
	}
	for i, e := range r.events {
		if e.ID == objID {
			r.events[i].Name = updated.Name
			r.events[i].Location = updated.Location
			r.events[i].Capacity = updated.Capacity
			r.events[i].Date = updated.Date
			return nil
		}
	}
	return fmt.Errorf("etkinlik bulunamadı")
}

func (r *inMemoryRepo) DeleteEvent(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("geçersiz ID formatı: %w", err)
	}
	for i, e := range r.events {
		if e.ID == objID {
			r.events = append(r.events[:i], r.events[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("etkinlik bulunamadı")
}

func (r *inMemoryRepo) UpdateAvailableTickets(id string, amount int) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("geçersiz ID formatı: %w", err)
	}
	for i, e := range r.events {
		if e.ID == objID {
			if e.Available+amount < 0 {
				return fmt.Errorf("kontenjan yetersiz")
			}
			r.events[i].Available += amount
			return nil
		}
	}
	return fmt.Errorf("etkinlik bulunamadı")
}

// ── Yardımcı ─────────────────────────────────────────────────────────────────

// seedEvent: Teste hazır bir etkinlik ekler ve ID'sini döndürür.
func seedEvent(t *testing.T, repo repository.EventStore, name string, capacity int) string {
	t.Helper()
	id := primitive.NewObjectID()
	err := repo.CreateEvent(models.Event{
		ID:        id,
		Name:      name,
		Location:  "Test Lokasyonu",
		Capacity:  capacity,
		Available: capacity,
		Date:      "2026-01-01",
	})
	if err != nil {
		t.Fatalf("seedEvent başarısız: %v", err)
	}
	return id.Hex()
}

// ── CreateEvent ───────────────────────────────────────────────────────────────

func TestCreateEvent_KayitYapilmali(t *testing.T) {
	repo := newInMemoryRepo()
	id := primitive.NewObjectID()

	err := repo.CreateEvent(models.Event{
		ID:        id,
		Name:      "Büyük Festival",
		Location:  "Kocaeli",
		Capacity:  500,
		Available: 500,
		Date:      "2026-05-15",
	})

	if err != nil {
		t.Fatalf("kayıt başarısız olmamalıydı: %v", err)
	}

	// Kaydedilen veriyi doğrula
	saved, err := repo.GetEventByID(id.Hex())
	if err != nil {
		t.Fatalf("kaydedilen etkinlik getirilemedi: %v", err)
	}
	if saved.Name != "Büyük Festival" {
		t.Errorf("beklenen 'Büyük Festival', alınan '%s'", saved.Name)
	}
	if saved.Available != 500 {
		t.Errorf("beklenen available=500, alınan %d", saved.Available)
	}
}

// ── GetAllEvents ──────────────────────────────────────────────────────────────

func TestGetAllEvents_BosListeDonmeli(t *testing.T) {
	repo := newInMemoryRepo()

	events, err := repo.GetAllEvents()
	if err != nil {
		t.Fatalf("hata oluşmamalıydı: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("boş liste bekleniyordu, %d eleman geldi", len(events))
	}
}

func TestGetAllEvents_TumKayitlariDonmeli(t *testing.T) {
	repo := newInMemoryRepo()
	seedEvent(t, repo, "Konser", 100)
	seedEvent(t, repo, "Sergi", 200)
	seedEvent(t, repo, "Tiyatro", 80)

	events, err := repo.GetAllEvents()
	if err != nil {
		t.Fatalf("hata oluşmamalıydı: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("3 etkinlik bekleniyordu, %d alındı", len(events))
	}
}

// ── GetEventByID ──────────────────────────────────────────────────────────────

func TestGetEventByID_DogruEtkinligiBulmali(t *testing.T) {
	repo := newInMemoryRepo()
	id := seedEvent(t, repo, "Hedef Etkinlik", 150)

	event, err := repo.GetEventByID(id)
	if err != nil {
		t.Fatalf("etkinlik bulunmalıydı: %v", err)
	}
	if event.Name != "Hedef Etkinlik" {
		t.Errorf("beklenen 'Hedef Etkinlik', alınan '%s'", event.Name)
	}
}

func TestGetEventByID_YanlisIDHataDonmeli(t *testing.T) {
	repo := newInMemoryRepo()

	_, err := repo.GetEventByID("gecersiz_hex_id")
	if err == nil {
		t.Error("geçersiz ID'de hata dönmeliydi")
	}
}

func TestGetEventByID_OlmayanIDHataDonmeli(t *testing.T) {
	repo := newInMemoryRepo()
	olmayan := primitive.NewObjectID().Hex()

	_, err := repo.GetEventByID(olmayan)
	if err == nil {
		t.Error("olmayan ID'de hata dönmeliydi")
	}
}

// ── UpdateEvent ───────────────────────────────────────────────────────────────

func TestUpdateEvent_AlanlariGuncellenmeli(t *testing.T) {
	repo := newInMemoryRepo()
	id := seedEvent(t, repo, "Eski İsim", 100)

	err := repo.UpdateEvent(id, models.Event{
		Name:     "Yeni İsim",
		Location: "Yeni Lokasyon",
		Capacity: 250,
		Date:     "2026-12-31",
	})
	if err != nil {
		t.Fatalf("güncelleme başarısız: %v", err)
	}

	updated, _ := repo.GetEventByID(id)
	if updated.Name != "Yeni İsim" {
		t.Errorf("isim güncellenmedi, alınan: '%s'", updated.Name)
	}
	if updated.Location != "Yeni Lokasyon" {
		t.Errorf("lokasyon güncellenmedi, alınan: '%s'", updated.Location)
	}
	if updated.Capacity != 250 {
		t.Errorf("kapasite güncellenmedi, alınan: %d", updated.Capacity)
	}
}

func TestUpdateEvent_OlmayanIDHataDonmeli(t *testing.T) {
	repo := newInMemoryRepo()
	olmayan := primitive.NewObjectID().Hex()

	err := repo.UpdateEvent(olmayan, models.Event{Name: "Test"})
	if err == nil {
		t.Error("olmayan ID'de hata dönmeliydi")
	}
}

// ── DeleteEvent ───────────────────────────────────────────────────────────────

func TestDeleteEvent_KayitSilinmeli(t *testing.T) {
	repo := newInMemoryRepo()
	id := seedEvent(t, repo, "Silinecek", 50)

	err := repo.DeleteEvent(id)
	if err != nil {
		t.Fatalf("silme başarısız: %v", err)
	}

	_, err = repo.GetEventByID(id)
	if err == nil {
		t.Error("silinen etkinlik hâlâ getirilebiliyorsa hata var")
	}
}

func TestDeleteEvent_CokluKayittaSadeceBirindenSilmeli(t *testing.T) {
	repo := newInMemoryRepo()
	id1 := seedEvent(t, repo, "Kalacak 1", 100)
	id2 := seedEvent(t, repo, "Silinecek", 200)
	id3 := seedEvent(t, repo, "Kalacak 2", 300)

	_ = repo.DeleteEvent(id2)

	events, _ := repo.GetAllEvents()
	if len(events) != 2 {
		t.Errorf("silme sonrası 2 etkinlik bekleniyordu, %d kaldı", len(events))
	}

	// Diğerleri hâlâ erişilebilir olmalı
	if _, err := repo.GetEventByID(id1); err != nil {
		t.Error("id1 silinmemeli ama erişilemiyor")
	}
	if _, err := repo.GetEventByID(id3); err != nil {
		t.Error("id3 silinmemeli ama erişilemiyor")
	}
}

func TestDeleteEvent_OlmayanIDHataDonmeli(t *testing.T) {
	repo := newInMemoryRepo()
	olmayan := primitive.NewObjectID().Hex()

	err := repo.DeleteEvent(olmayan)
	if err == nil {
		t.Error("olmayan ID'de hata dönmeliydi")
	}
}

// ── UpdateAvailableTickets ────────────────────────────────────────────────────

func TestUpdateAvailableTickets_AzaltmaBasarili(t *testing.T) {
	repo := newInMemoryRepo()
	id := seedEvent(t, repo, "Konser", 100)

	err := repo.UpdateAvailableTickets(id, -1)
	if err != nil {
		t.Fatalf("bilet azaltma başarısız: %v", err)
	}

	event, _ := repo.GetEventByID(id)
	if event.Available != 99 {
		t.Errorf("beklenen available=99, alınan %d", event.Available)
	}
}

func TestUpdateAvailableTickets_ArtirmaBasarili(t *testing.T) {
	repo := newInMemoryRepo()
	id := seedEvent(t, repo, "Konser", 100)
	_ = repo.UpdateAvailableTickets(id, -5) // 95'e düşür

	err := repo.UpdateAvailableTickets(id, 3) // 98'e çıkar (iptal)
	if err != nil {
		t.Fatalf("bilet artırma başarısız: %v", err)
	}

	event, _ := repo.GetEventByID(id)
	if event.Available != 98 {
		t.Errorf("beklenen available=98, alınan %d", event.Available)
	}
}

func TestUpdateAvailableTickets_KontenjanYetersizHataDonmeli(t *testing.T) {
	repo := newInMemoryRepo()
	id := seedEvent(t, repo, "Küçük Etkinlik", 2)

	_ = repo.UpdateAvailableTickets(id, -2) // 0'a düş

	err := repo.UpdateAvailableTickets(id, -1) // 0'dan aşağı → hata
	if err == nil {
		t.Error("kontenjan yetersizken hata dönmeliydi")
	}
}

func TestUpdateAvailableTickets_OlmayanIDHataDonmeli(t *testing.T) {
	repo := newInMemoryRepo()
	olmayan := primitive.NewObjectID().Hex()

	err := repo.UpdateAvailableTickets(olmayan, -1)
	if err == nil {
		t.Error("olmayan ID'de hata dönmeliydi")
	}
}

func TestUpdateAvailableTickets_YanlisIDFormatuHataDonmeli(t *testing.T) {
	repo := newInMemoryRepo()

	err := repo.UpdateAvailableTickets("gecersiz_id", -1)
	if err == nil {
		t.Error("geçersiz ID formatında hata dönmeliydi")
	}
}