package service

import (
	"booking-service/models"
	"booking-service/repository"
	"errors"
)

type bookingService struct {
	repo        repository.BookingStore
	eventClient EventClient
}

func NewBookingService(repo repository.BookingStore, eventClient EventClient) BookingService {
	return &bookingService{repo: repo, eventClient: eventClient}
}

func (s *bookingService) CreateBooking(booking models.Booking) (string, error) {
	if booking.EventID == "" || booking.UserID == "" || booking.Seats <= 0 {
		return "", errors.New("event_id, user_id boş olamaz ve seats 0'dan büyük olmalıdır")
	}

	if err := s.eventClient.CheckCapacity(booking.EventID); err != nil {
		return "", err
	}

	id, err := s.repo.CreateBooking(booking)
	if err != nil {
		return "", errors.New("rezervasyon kaydedilemedi")
	}
	_ = s.eventClient.UpdateCapacity(booking.EventID, -booking.Seats)

	return id, nil
}

func (s *bookingService) GetBookingsByUser(userID string) ([]models.Booking, error) {
	if userID == "" {
		return nil, errors.New("user_id boş olamaz")
	}
	return s.repo.GetBookingsByUser(userID)
}