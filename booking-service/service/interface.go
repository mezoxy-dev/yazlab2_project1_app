package service

import "booking-service/models"

type BookingService interface {
	CreateBooking(booking models.Booking) (string, error)
	GetBookingsByUser(userID string) ([]models.Booking, error)
}

type EventClient interface {
	CheckCapacity(eventID string) error
	UpdateCapacity(eventID string, amount int) error
}