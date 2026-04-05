package service

import (
	"errors"
	"event-service/models"
	"event-service/repository"
)

type eventService struct {
	repo repository.EventStore
}

// NewEventService fonksiyonu, eventService struct'ını döndürür ancak geri dönüş tipi EventService arayüzüdür.
// eventService struct'ının detayları gizlenir.
func NewEventService(repo repository.EventStore) EventService {
	return &eventService{repo: repo}
} 

func (s *eventService) CreateEvent(event models.Event) error {
	if event.Name == "" || event.Location == "" || event.Capacity <= 0 {
		return errors.New("isim ve konum boş olamaz, kapasite 0'dan büyük olmalıdır")
	}
 
	event.Available = event.Capacity
 
	return s.repo.CreateEvent(event)
}

func (s *eventService) GetAllEvents() ([]models.Event, error) {
	return s.repo.GetAllEvents()
}

func (s *eventService) GetEventByID(id string) (*models.Event, error) {
	event, err := s.repo.GetEventByID(id)
	if err != nil {
		return nil, errors.New("etkinlik bulunamadı")
	}
	return event, nil
}

func (s *eventService) UpdateEvent(id string, event models.Event) error {
	return s.repo.UpdateEvent(id, event)
}
 
func (s *eventService) DeleteEvent(id string) error {
	return s.repo.DeleteEvent(id)
}

func (s *eventService) UpdateAvailableTickets(id string, amount int) error {
	return s.repo.UpdateAvailableTickets(id, amount)
}