package service_test

import (
	"booking-service/models"
	"booking-service/service"
	"errors"
	"testing"
)

// Mock Repository

type mockRepo struct {
	bookings   map[string][]models.Booking
	shouldFail bool
}

func (m *mockRepo) CreateBooking(b models.Booking) (string, error) {
	if m.shouldFail {
		return "", errors.New("db hatası")
	}
	b.ID = "mock-id-123"
	if m.bookings == nil {
		m.bookings = make(map[string][]models.Booking)
	}
	m.bookings[b.UserID] = append(m.bookings[b.UserID], b)
	return b.ID, nil
}

func (m *mockRepo) GetBookingsByUser(userID string) ([]models.Booking, error) {
	if m.shouldFail {
		return nil, errors.New("db hatası")
	}
	return m.bookings[userID], nil
}

// Mock EventClient

type mockEventClient struct {
	checkErr  error
	updateErr error
}

func (m *mockEventClient) CheckCapacity(_ string) error  { return m.checkErr }
func (m *mockEventClient) UpdateCapacity(_ string, _ int) error { return m.updateErr }

// Yardımcı

func newService(repo *mockRepo, client *mockEventClient) service.BookingService {
	return service.NewBookingService(repo, client)
}

func validBooking() models.Booking {
	return models.Booking{EventID: "konser123", UserID: "oguzhan", Seats: 2}
}

// CreateBooking

func TestCreateBooking_Basarili(t *testing.T) {
	svc := newService(&mockRepo{}, &mockEventClient{})

	id, err := svc.CreateBooking(validBooking())
	if err != nil {
		t.Fatalf("hata olmamalıydı: %v", err)
	}
	if id == "" {
		t.Error("başarılı rezervasyonda ID dönmeliydi")
	}
}

func TestCreateBooking_EksikEventID(t *testing.T) {
	svc := newService(&mockRepo{}, &mockEventClient{})

	_, err := svc.CreateBooking(models.Booking{UserID: "oguzhan", Seats: 2})
	if err == nil {
		t.Error("event_id boşken hata dönmeliydi")
	}
}

func TestCreateBooking_EksikUserID(t *testing.T) {
	svc := newService(&mockRepo{}, &mockEventClient{})

	_, err := svc.CreateBooking(models.Booking{EventID: "konser123", Seats: 2})
	if err == nil {
		t.Error("user_id boşken hata dönmeliydi")
	}
}

func TestCreateBooking_SifirKoltuk(t *testing.T) {
	svc := newService(&mockRepo{}, &mockEventClient{})

	_, err := svc.CreateBooking(models.Booking{EventID: "konser123", UserID: "oguzhan", Seats: 0})
	if err == nil {
		t.Error("seats=0 iken hata dönmeliydi")
	}
}

func TestCreateBooking_NegatifKoltuk(t *testing.T) {
	svc := newService(&mockRepo{}, &mockEventClient{})

	_, err := svc.CreateBooking(models.Booking{EventID: "konser123", UserID: "oguzhan", Seats: -1})
	if err == nil {
		t.Error("seats<0 iken hata dönmeliydi")
	}
}

func TestCreateBooking_EventKapasitesiYok(t *testing.T) {
	client := &mockEventClient{checkErr: errors.New("etkinlik bulunamadı veya yer yok")}
	svc := newService(&mockRepo{}, client)

	_, err := svc.CreateBooking(validBooking())
	if err == nil {
		t.Error("event kapasite yokken hata dönmeliydi")
	}
}

func TestCreateBooking_DBHatasindaHataDoner(t *testing.T) {
	svc := newService(&mockRepo{shouldFail: true}, &mockEventClient{})

	_, err := svc.CreateBooking(validBooking())
	if err == nil {
		t.Error("db hatası olduğunda hata dönmeliydi")
	}
}

func TestCreateBooking_KapasiteGuncellemeHatasiSessizGecilir(t *testing.T) {
	client := &mockEventClient{updateErr: errors.New("güncelleme hatası")}
	svc := newService(&mockRepo{}, client)

	id, err := svc.CreateBooking(validBooking())
	if err != nil {
		t.Fatalf("kapasite güncelleme hatası rezervasyonu engellememelidir: %v", err)
	}
	if id == "" {
		t.Error("ID dönmeliydi")
	}
}

// GetBookingsByUser

func TestGetBookingsByUser_BosListe(t *testing.T) {
	svc := newService(&mockRepo{}, &mockEventClient{})

	bookings, err := svc.GetBookingsByUser("oguzhan")
	if err != nil {
		t.Fatalf("hata olmamalıydı: %v", err)
	}
	if len(bookings) != 0 {
		t.Errorf("boş liste bekleniyordu, %d eleman geldi", len(bookings))
	}
}

func TestGetBookingsByUser_KayitliRezervasyon(t *testing.T) {
	repo := &mockRepo{}
	svc := newService(repo, &mockEventClient{})

	_, _ = svc.CreateBooking(models.Booking{EventID: "konser1", UserID: "oguzhan", Seats: 2})
	_, _ = svc.CreateBooking(models.Booking{EventID: "konser2", UserID: "oguzhan", Seats: 1})

	bookings, err := svc.GetBookingsByUser("oguzhan")
	if err != nil {
		t.Fatalf("hata olmamalıydı: %v", err)
	}
	if len(bookings) != 2 {
		t.Errorf("2 rezervasyon bekleniyordu, %d alındı", len(bookings))
	}
}

func TestGetBookingsByUser_BosIDHata(t *testing.T) {
	svc := newService(&mockRepo{}, &mockEventClient{})

	_, err := svc.GetBookingsByUser("")
	if err == nil {
		t.Error("boş user_id'de hata dönmeliydi")
	}
}

func TestGetBookingsByUser_FarkliKullanicilariKaristirmaz(t *testing.T) {
	repo := &mockRepo{}
	svc := newService(repo, &mockEventClient{})

	_, _ = svc.CreateBooking(models.Booking{EventID: "konser1", UserID: "oguzhan", Seats: 2})
	_, _ = svc.CreateBooking(models.Booking{EventID: "konser2", UserID: "ali", Seats: 1})

	bookings, _ := svc.GetBookingsByUser("oguzhan")
	if len(bookings) != 1 {
		t.Errorf("sadece oguzhan'ın rezervasyonu gelmeli, %d geldi", len(bookings))
	}
	if bookings[0].UserID != "oguzhan" {
		t.Errorf("yanlış kullanıcı rezervasyonu geldi: %s", bookings[0].UserID)
	}
}