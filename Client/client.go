package client

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	userAgent = "User-Agent"
)

// настраиваемый клиент, так-как на АПИ ЦБ висит защита
type HttpClient struct {
	client    *http.Client
	baseUrl   string
	userAgent string
}

// функция для создания экземпляра клинта и прописания в него моих параметров
func NewHttpClient(timeout time.Duration, baseUrl, userAgent string) *HttpClient {
	return &HttpClient{
		client:    &http.Client{Timeout: timeout},
		baseUrl:   baseUrl,
		userAgent: userAgent,
	}
}

func (c *HttpClient) GetByDailyRate(path string) ([]byte, error) {
	// метод для нттр клиента, который будет вызываться в рабочей функции
	url := c.baseUrl + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error request: %w", err)
	}
	req.Header.Set(userAgent, c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error HTTP request: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status error: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	return body, nil
}
