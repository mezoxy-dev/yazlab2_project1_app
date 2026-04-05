package repository_test

import (
	"booking-service/models"
	"booking-service/repository"
	"errors"
	"testing"
)

type inMemoryRepo struct {
	bookings   map[string][]models.Booking
	counter    int
	shouldFail bool
}

func newInMemoryRepo() repository.BookingStore {
	return &inMemoryRepo{bookings: make(map[string][]models.Booking)}
}

func (r *inMemoryRepo) CreateBooking(b models.Booking) (string, error) {
	if r.shouldFail {
		return "", errors.New("db hatası")
	}
	r.counter++
	b.ID = "id-" + string(rune('0'+r.counter))
	r.bookings[b.UserID] = append(r.bookings[b.UserID], b)
	return b.ID, nil
}

func (r *inMemoryRepo) GetBookingsByUser(userID string) ([]models.Booking, error) {
	if r.shouldFail {
		return nil, errors.New("db hatası")
	}
	return r.bookings[userID], nil
}

// CreateBooking

func TestCreateBooking_IDDonmeli(t *testing.T) {
	repo := newInMemoryRepo()

	id, err := repo.CreateBooking(models.Booking{EventID: "e1", UserID: "u1", Seats: 2})
	if err != nil {
		t.Fatalf("hata olmamalıydı: %v", err)
	}
	if id == "" {
		t.Error("ID dönmeliydi")
	}
}

func TestCreateBooking_TumAlanlariKorumali(t *testing.T) {
	repo := newInMemoryRepo()

	original := models.Booking{EventID: "konser1", UserID: "oguzhan", Seats: 3}
	_, _ = repo.CreateBooking(original)

	bookings, _ := repo.GetBookingsByUser("oguzhan")
	if len(bookings) == 0 {
		t.Fatal("rezervasyon kaydedilmedi")
	}

	got := bookings[0]
	if got.EventID != "konser1" { t.Errorf("event_id: beklenen konser1, alınan %s", got.EventID) }
	if got.UserID != "oguzhan"  { t.Errorf("user_id: beklenen oguzhan, alınan %s", got.UserID) }
	if got.Seats != 3           { t.Errorf("seats: beklenen 3, alınan %d", got.Seats) }
}

func TestCreateBooking_CokluKayit(t *testing.T) {
	repo := newInMemoryRepo()

	_, _ = repo.CreateBooking(models.Booking{EventID: "e1", UserID: "u1", Seats: 1})
	_, _ = repo.CreateBooking(models.Booking{EventID: "e2", UserID: "u1", Seats: 2})
	_, _ = repo.CreateBooking(models.Booking{EventID: "e3", UserID: "u1", Seats: 3})

	bookings, _ := repo.GetBookingsByUser("u1")
	if len(bookings) != 3 {
		t.Errorf("3 rezervasyon bekleniyordu, %d alındı", len(bookings))
	}
}

// GetBookingsByUser

func TestGetBookingsByUser_BosListe(t *testing.T) {
	repo := newInMemoryRepo()

	bookings, err := repo.GetBookingsByUser("olmayan_kullanici")
	if err != nil {
		t.Fatalf("hata olmamalıydı: %v", err)
	}
	if len(bookings) != 0 {
		t.Errorf("boş liste bekleniyordu, %d alındı", len(bookings))
	}
}

func TestGetBookingsByUser_FarkliKullanicilariKaristirmaz(t *testing.T) {
	repo := newInMemoryRepo()

	_, _ = repo.CreateBooking(models.Booking{EventID: "e1", UserID: "oguzhan", Seats: 2})
	_, _ = repo.CreateBooking(models.Booking{EventID: "e2", UserID: "ali", Seats: 1})
	_, _ = repo.CreateBooking(models.Booking{EventID: "e3", UserID: "oguzhan", Seats: 3})

	bookings, _ := repo.GetBookingsByUser("oguzhan")
	if len(bookings) != 2 {
		t.Errorf("oguzhan için 2 rezervasyon bekleniyordu, %d alındı", len(bookings))
	}
	for _, b := range bookings {
		if b.UserID != "oguzhan" {
			t.Errorf("yanlış kullanıcı rezervasyonu geldi: %s", b.UserID)
		}
	}
}

var _ repository.BookingStore = &inMemoryRepo{}