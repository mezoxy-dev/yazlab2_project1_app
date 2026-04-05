package service_test

import (
	"dispatcher/models"
	"dispatcher/service"
	"errors"
	"testing"
	"time"
)

type mockLogRepo struct {
	inserted []models.TrafficLog
	getErr   error
}

func (m *mockLogRepo) Insert(log models.TrafficLog) error {
	m.inserted = append(m.inserted, log)
	return nil
}

func (m *mockLogRepo) GetRecent(limit int64) ([]models.TrafficLog, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if int64(len(m.inserted)) > limit {
		return m.inserted[:limit], nil
	}
	return m.inserted, nil
}

func waitForAsync(condition func() bool) {
	for i := 0; i < 20; i++ {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestRecordAsync_LogKaydedilmeli(t *testing.T) {
	repo := &mockLogRepo{}
	svc := service.NewLogService(repo)

	svc.RecordAsync("GET", "/events", "127.0.0.1", 200, 45)
	waitForAsync(func() bool { return len(repo.inserted) > 0 })

	if len(repo.inserted) == 0 {
		t.Fatal("log kaydedilmedi")
	}
	l := repo.inserted[0]
	if l.Method != "GET"    { t.Errorf("method: beklenen GET, alınan %s", l.Method) }
	if l.Path != "/events"  { t.Errorf("path: beklenen /events, alınan %s", l.Path) }
	if l.Status != 200      { t.Errorf("status: beklenen 200, alınan %d", l.Status) }
	if l.Duration != 45     { t.Errorf("duration: beklenen 45, alınan %d", l.Duration) }
	if l.IP != "127.0.0.1" { t.Errorf("ip: beklenen 127.0.0.1, alınan %s", l.IP) }
	if l.Timestamp.IsZero() { t.Error("timestamp boş olmamalı") }
}

func TestRecordAsync_FarkliStatusKodlari(t *testing.T) {
	cases := []struct{ status int; path string }{
		{200, "/events"}, {401, "/events"}, {403, "/admin/stats"},
		{404, "/olmayan"}, {500, "/bookings"},
	}
	for _, c := range cases {
		repo := &mockLogRepo{}
		svc := service.NewLogService(repo)
		svc.RecordAsync("GET", c.path, "::1", c.status, 10)
		waitForAsync(func() bool { return len(repo.inserted) > 0 })
		if len(repo.inserted) == 0 {
			t.Errorf("status=%d için log kaydedilmedi", c.status)
			continue
		}
		if repo.inserted[0].Status != c.status {
			t.Errorf("status: beklenen %d, alınan %d", c.status, repo.inserted[0].Status)
		}
	}
}

func TestGetRecent_KayitlariDondurur(t *testing.T) {
	repo := &mockLogRepo{}
	svc := service.NewLogService(repo)
	svc.RecordAsync("GET", "/events", "::1", 200, 10)
	svc.RecordAsync("POST", "/bookings", "::1", 201, 20)
	svc.RecordAsync("DELETE", "/events/1", "::1", 204, 5)
	waitForAsync(func() bool { return len(repo.inserted) >= 3 })

	logs, err := svc.GetRecent(100)
	if err != nil { t.Fatalf("hata olmamalıydı: %v", err) }
	if len(logs) != 3 { t.Errorf("3 log bekleniyordu, %d alındı", len(logs)) }
}

func TestGetRecent_LimitUygulanir(t *testing.T) {
	repo := &mockLogRepo{}
	svc := service.NewLogService(repo)
	for i := 0; i < 10; i++ {
		svc.RecordAsync("GET", "/events", "::1", 200, 5)
	}
	waitForAsync(func() bool { return len(repo.inserted) >= 10 })

	logs, err := svc.GetRecent(3)
	if err != nil { t.Fatalf("hata olmamalıydı: %v", err) }
	if len(logs) > 3 { t.Errorf("limit 3 iken %d log döndü", len(logs)) }
}

func TestGetRecent_HataDurumu(t *testing.T) {
	repo := &mockLogRepo{getErr: errors.New("db hatası")}
	svc := service.NewLogService(repo)
	_, err := svc.GetRecent(10)
	if err == nil { t.Error("repo hata verince servis de hata dönmeliydi") }
}