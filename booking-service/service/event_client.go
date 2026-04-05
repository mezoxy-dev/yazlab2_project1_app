package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type httpEventClient struct {
	baseURL     string
	internalKey string
	httpClient  *http.Client
}

func NewEventClient() EventClient {
	return &httpEventClient{
		baseURL:     os.Getenv("EVENT_SERVICE_URL"),
		internalKey: os.Getenv("INTERNAL_GATEWAY_KEY"),
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *httpEventClient) CheckCapacity(eventID string) error {
	url := fmt.Sprintf("%s/events/%s", c.baseURL, eventID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Internal-Secret", c.internalKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("etkinlik bulunamadı veya yer yok")
	}
	return nil
}

func (c *httpEventClient) UpdateCapacity(eventID string, amount int) error {
	url := fmt.Sprintf("%s/events/%s", c.baseURL, eventID)

	body, _ := json.Marshal(map[string]int{"amount": amount})
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Internal-Secret", c.internalKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("event kapasitesi güncellenemedi")
	}
	return nil
}