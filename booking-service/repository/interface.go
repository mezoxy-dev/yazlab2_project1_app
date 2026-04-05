package repository
 
import "booking-service/models"
 
type BookingStore interface {
	CreateBooking(booking models.Booking) (string, error)
	GetBookingsByUser(userID string) ([]models.Booking, error)
}
 