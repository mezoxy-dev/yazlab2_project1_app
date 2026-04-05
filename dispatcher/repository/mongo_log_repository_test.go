package repository_test

import (
	"dispatcher/models"
	"dispatcher/repository"
	"fmt"
	"testing"
	"time"

)

type inMemoryLogRepo struct {
	logs   []models.TrafficLog
	getErr error
}

func newInMemoryLogRepo() repository.LogStore {
	return &inMemoryLogRepo{}
}

func (r *inMemoryLogRepo) Insert(log models.TrafficLog) error {
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}
	r.logs = append(r.logs, log)
	return nil
}

func (r *inMemoryLogRepo) GetRecent(limit int64) ([]models.TrafficLog, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if int64(len(r.logs)) <= limit {
		return r.logs, nil
	}
	return r.logs[:limit], nil
}

// Yardımcı

func makeLog(method, path string, status int) models.TrafficLog {
	return models.TrafficLog{
		Method:    method,
		Path:      path,
		Status:    status,
		Duration:  10,
		IP:        "127.0.0.1",
		Timestamp: time.Now(),
	}
}

// Insert

func TestInsert_LogKaydedilmeli(t *testing.T) {
	repo := newInMemoryLogRepo()

	err := repo.Insert(makeLog("GET", "/events", 200))
	if err != nil {
		t.Fatalf("insert başarısız olmamalıydı: %v", err)
	}

	logs, _ := repo.GetRecent(10)
	if len(logs) != 1 {
		t.Errorf("1 log bekleniyordu, %d alındı", len(logs))
	}
}

func TestInsert_CokluLogKaydedilmeli(t *testing.T) {
	repo := newInMemoryLogRepo()

	entries := []models.TrafficLog{
		makeLog("GET", "/events", 200),
		makeLog("POST", "/bookings", 201),
		makeLog("DELETE", "/events/1", 204),
		makeLog("GET", "/admin/stats", 403),
	}

	for _, e := range entries {
		if err := repo.Insert(e); err != nil {
			t.Fatalf("insert başarısız: %v", err)
		}
	}

	logs, _ := repo.GetRecent(100)
	if len(logs) != 4 {
		t.Errorf("4 log bekleniyordu, %d alındı", len(logs))
	}
}

func TestInsert_TumAlanlariKorumali(t *testing.T) {
	repo := newInMemoryLogRepo()
	now := time.Now()

	original := models.TrafficLog{
		Method:    "POST",
		Path:      "/register",
		Status:    201,
		Duration:  55,
		IP:        "192.168.1.1",
		Timestamp: now,
	}
	_ = repo.Insert(original)

	logs, _ := repo.GetRecent(1)
	if len(logs) == 0 {
		t.Fatal("log kaydedilmedi")
	}

	got := logs[0]
	if got.Method != "POST"         { t.Errorf("method: beklenen POST, alınan %s", got.Method) }
	if got.Path != "/register"      { t.Errorf("path: beklenen /register, alınan %s", got.Path) }
	if got.Status != 201            { t.Errorf("status: beklenen 201, alınan %d", got.Status) }
	if got.Duration != 55           { t.Errorf("duration: beklenen 55, alınan %d", got.Duration) }
	if got.IP != "192.168.1.1"      { t.Errorf("ip: beklenen 192.168.1.1, alınan %s", got.IP) }
	if got.Timestamp.IsZero()       { t.Error("timestamp boş olmamalı") }
}

// GetRecent

func TestGetRecent_BosRepoDonmeli(t *testing.T) {
	repo := newInMemoryLogRepo()

	logs, err := repo.GetRecent(10)
	if err != nil {
		t.Fatalf("hata olmamalıydı: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("boş liste bekleniyordu, %d eleman alındı", len(logs))
	}
}

func TestGetRecent_LimitUygulanmali(t *testing.T) {
	repo := newInMemoryLogRepo()

	for i := 0; i < 10; i++ {
		_ = repo.Insert(makeLog("GET", fmt.Sprintf("/events/%d", i), 200))
	}

	logs, err := repo.GetRecent(3)
	if err != nil {
		t.Fatalf("hata olmamalıydı: %v", err)
	}
	if len(logs) > 3 {
		t.Errorf("limit=3 iken %d log döndü", len(logs))
	}
}

func TestGetRecent_LimitdenAzKayitVarsaTumunuDonmeli(t *testing.T) {
	repo := newInMemoryLogRepo()
	_ = repo.Insert(makeLog("GET", "/events", 200))
	_ = repo.Insert(makeLog("POST", "/bookings", 201))

	logs, err := repo.GetRecent(100) 
	if err != nil {
		t.Fatalf("hata olmamalıydı: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("2 log bekleniyordu, %d alındı", len(logs))
	}
}

func TestGetRecent_FarkliStatuslariKorumali(t *testing.T) {
	repo := newInMemoryLogRepo()
	statuses := []int{200, 201, 401, 403, 404, 500}

	for _, s := range statuses {
		_ = repo.Insert(makeLog("GET", "/test", s))
	}

	logs, _ := repo.GetRecent(100)
	if len(logs) != len(statuses) {
		t.Errorf("beklenen %d log, alınan %d", len(statuses), len(logs))
	}
}

var _ repository.LogStore = &inMemoryLogRepo{}

