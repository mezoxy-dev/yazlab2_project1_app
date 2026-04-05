package repository

import "event-service/models"

type EventStore interface {
	CreateEvent(event models.Event) error
	GetAllEvents() ([]models.Event, error)
	GetEventByID(id string) (*models.Event, error)
	UpdateEvent(id string, event models.Event) error
	DeleteEvent(id string) error
	UpdateAvailableTickets(id string, amount int) error
}
 