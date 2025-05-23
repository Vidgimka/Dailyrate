package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	"gopkg.in/yaml.v3"
)

const (
	configYaml = "config.yaml"
	userAgent  = "User-Agent"
)

// юзаем алиас для автопарсинга  xml.Unmarshal
type FloatWithComma float64

type ValCurs struct {
	Date   string   `xml:"Date,attr"`
	Valute []Valute `xml:"Valute"`
}

type Valute struct {
	CharCode string         `xml:"CharCode"`
	Name     string         `xml:"Name"`
	Value    FloatWithComma `xml:"Value"`
}

func (f *FloatWithComma) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// автозамена запятой при парсинге с апи
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		log.Fatal("Decode:", err)
	}
	s = strings.Replace(s, ",", ".", 1)
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatal("Replace:", err)
	}
	*f = FloatWithComma(value)
	return nil
}

type operatingData struct {
	Date     string
	CharCode string
	Name     string
	Value    float64
}

type Config struct {
	Api   ApiConfig  `yaml:"api"`
	DateF DateConfig `yaml:"formats_date"`
}
type ApiConfig struct {
	Timeout   time.Duration `yaml:"timeout"`
	BaseUrl   string        `yaml:"base_url"`
	UserAgent string        `yaml:"user_agen"`
}
type DateConfig struct {
	DateFormat string `yaml:"dateFormat"`
}

func LoagYamlConf(path string) (*Config, error) {
	fileYamlDate, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read yaml-File: %w", err)
	}
	var config Config
	err = yaml.Unmarshal(fileYamlDate, &config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal yaml-File: %w", err)
	}
	return &config, err
}

func main() {
	config, err := LoagYamlConf(configYaml)
	if err != nil {
		log.Fatal("loag yaml file:", err)
	}

	httpClient := newHttpClient(config.Api.Timeout, config.Api.BaseUrl, config.Api.UserAgent)

	formatDate := config.DateF.DateFormat

	rates, err := getIn90DaysRates(formatDate, httpClient)
	if err != nil {
		log.Fatal("Error getting data for 90 days:", err)
	}
	dataReturn, err := getMinMaxAvg(rates)
	if err != nil {
		log.Fatal("Error getting data for 90 days:", err)
	}
	// min, max, avg := businessLogic(rates)

	fmt.Printf("Минимальный курс на дату: %s\n", dataReturn.Min.Date)
	fmt.Printf("Для валюты %s, c кодом %s\n", dataReturn.Min.Name, dataReturn.Min.CharCode)
	fmt.Printf("Составляет: %g\n", dataReturn.Min.Value)
	fmt.Printf("Максимальный курс на дату: %s\n", dataReturn.Max.Date)
	fmt.Printf("Для валюты %s, c кодом %s\n", dataReturn.Max.Name, dataReturn.Max.CharCode)
	fmt.Printf("Составляет: %g\n", dataReturn.Max.Value)
	fmt.Printf("Cреднее значение курса рубля за весь период по всем валютам: %f\n", dataReturn.Avg)
}

// настраиваемый клиент, так-как на АПИ ЦБ висит защита
type httpClient struct {
	client    *http.Client
	baseUrl   string
	userAgent string
}

// функция для создания экземпляра клинта и прописания в него моих параметров
func newHttpClient(timeout time.Duration, baseUrl, userAgent string) *httpClient {
	return &httpClient{
		client:    &http.Client{Timeout: timeout},
		baseUrl:   baseUrl,
		userAgent: userAgent,
	}
}

func (c *httpClient) GetByDailyRate(path string) ([]byte, error) {
	// метод для нттр клиента, который будет вызываться в рабочей функции
	url := c.baseUrl + path
	// url := fmt.Sprintf("http://www.cbr.ru/scripts/XML_daily_eng.asp?date_req=%s", dateOfTheRequestedDay)
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

func xmlDecoder(body []byte) (ValCurs, error) {
	// функция декодер для соблюдения Single Responsibility Principle!
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		if charset == "windows-1251" {
			return transform.NewReader(input, charmap.Windows1251.NewDecoder()), nil
		}
		return nil, fmt.Errorf("unsupported encoding: %s", charset)
	}

	var daily ValCurs
	if err := decoder.Decode(&daily); err != nil {
		log.Fatal("XML parsing error:", err)
	}
	return daily, nil
}

func dateBorder() (time.Time, time.Time, error) {
	end := time.Now()
	start := end.AddDate(0, 0, -90)
	return start, end, nil
}

// функция запроса данных с API ЦБ в течении 90 дней с момента даты запроса
func getIn90DaysRates(formatDate string, client *httpClient) ([]operatingData, error) {

	start, end, err := dateBorder()
	if err != nil {
		return nil, fmt.Errorf("date border error:%w", err)
	}

	var allRrate []operatingData
	// запускаем цикл с проверкой дат до текущей включительно
	for d := start; d.Before(end) || d.Equal(end); d = d.AddDate(0, 0, 1) {

		dateOfTheRequestedDay := d.Format(formatDate)

		body, err := client.GetByDailyRate(dateOfTheRequestedDay)
		if err != nil {
			log.Fatal("GET error:", err)
			continue
		}

		daily, err := xmlDecoder(body)
		if err != nil {
			return nil, fmt.Errorf("XML parsing error:%w", err)
		}

		for _, valute := range daily.Valute {
			allRrate = append(allRrate, operatingData{
				Date:     daily.Date,
				CharCode: valute.CharCode,
				Name:     valute.Name,
				Value:    float64(valute.Value),
			})
		}
		// fmt.Println("Запись выполнена")
	}
	return allRrate, nil
}

type dataReturn struct {
	Min operatingData
	Max operatingData
	Avg float64
}

func getMinMaxAvg(rates []operatingData) (*dataReturn, error) {
	if len(rates) == 0 {
		return nil, errors.New("uncorrect data")
	}
	// фильтруем  Специальное права заимствования)

	var filteredRates []operatingData
	for _, rate := range rates {
		if rate.CharCode != "XDR" {
			filteredRates = append(filteredRates, rate)
		}
	}
	if len(filteredRates) == 0 {
		return nil, errors.New("no data after filtering")
	}

	// набор данных с максимальной и минимальной ставкой ЦБ по валюте
	// var dataReturn dataReturn
	var max operatingData
	var min operatingData
	max = filteredRates[0]
	min = filteredRates[0]
	total := 0.0

	for _, rate := range filteredRates {
		if rate.Value > max.Value {
			max = rate
		}
		if rate.Value < min.Value {
			min = rate
		}
		total += rate.Value
	}
	var avg float64
	avg = total / float64(len(filteredRates))
	return &dataReturn{Min: min, Max: max, Avg: avg}, nil
}
