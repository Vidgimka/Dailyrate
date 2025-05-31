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

type HttpClientIntrface interface {
	GetByDailyRate(dateOfTheRequestedDay time.Time) (ValCurs, error)
}

// настраиваемый клиент, так-как на АПИ ЦБ висит защита
type httpClient struct {
	client     *http.Client
	baseUrl    string
	userAgent  string
	formatDate string
}

// функция-фабрика для создания экземпляра клинта и прописания в него моих параметров
func NewHttpClient(timeout time.Duration, baseUrl, userAgent, formatDate string) HttpClientIntrface {
	return &httpClient{
		client:     &http.Client{Timeout: timeout},
		baseUrl:    baseUrl,
		userAgent:  userAgent,
		formatDate: formatDate,
	}
}

func (c *httpClient) GetByDailyRate(dateOfTheRequestedDay time.Time) (ValCurs, error) {
	// метод для нттр клиента, который будет вызываться в рабочей функции
	stringDate := dateOfTheRequestedDay.Format(c.formatDate)
	url := c.baseUrl + stringDate

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ValCurs{}, fmt.Errorf("error request: %w", err)
	}
	req.Header.Set(userAgent, c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return ValCurs{}, fmt.Errorf("error HTTP request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ValCurs{}, fmt.Errorf("HTTP status error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ValCurs{}, fmt.Errorf("read error: %w", err)
	}

	daily, err := xmlDecoderInValCurs(body)
	if err != nil {
		return ValCurs{}, fmt.Errorf("xmlDecoder:%w", err)
	}
	return daily, nil
}
