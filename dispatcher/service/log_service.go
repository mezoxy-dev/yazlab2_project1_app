package service

import (
	"dispatcher/models"
	"dispatcher/repository"
	"log"
	"time"
)

type logService struct {
	repo repository.LogStore
}

func NewLogService(repo repository.LogStore) LogService {
	return &logService{repo: repo}
}

// RecordAsync: Logu goroutine ile asenkron kaydeder.
func (s *logService) RecordAsync(method, path, ip string, status int, durationMs int64, message string) {
	entry := models.TrafficLog{
		Method:    method,
		Path:      path,
		Status:    status,
		Duration:  durationMs,
		IP:        ip,
		Timestamp: time.Now(),
		Message:   message,
	}
	go func(l models.TrafficLog) {
		if err := s.repo.Insert(l); err != nil {
			log.Printf("log kayıt hatası: %v", err)
		}
	}(entry)
}

// GetRecent: Son N logu döndürür.
func (s *logService) GetRecent(limit int64) ([]models.TrafficLog, error) {
	return s.repo.GetRecent(limit)
}