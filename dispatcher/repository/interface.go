package repository
 
import "dispatcher/models"
 

// Middleware katmanı bu interface'e bağımlıdır, somut MongoDB
// implementasyonuna değil.
type LogStore interface {
	Insert(log models.TrafficLog) error
	GetRecent(limit int64) ([]models.TrafficLog, error)
}
 