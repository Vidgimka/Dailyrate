package service

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	client "main/Client"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

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

type operatingData struct {
	Date     time.Time
	CharCode string
	Name     string
	Value    float64
}

func (f *FloatWithComma) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// автозамена запятой при парсинге с апи
	// я пытаюсь заменить имя функции на  CommaReplace, но после этого код не работает
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
		return ValCurs{}, fmt.Errorf("XML parsing error:%w", err)
	}
	return daily, nil
}

func dateBorder() (time.Time, time.Time, error) {
	end := time.Now()
	start := end.AddDate(0, 0, -90)
	return start, end, nil
}

type NewService struct {
	FormatDate string
	Client     *client.HttpClient
}

// функция запроса данных с API ЦБ в течении 90 дней с момента даты запроса
func (s *NewService) GetIn90DaysRates() ([]operatingData, error) {
	start, end, err := dateBorder()
	if err != nil {
		return nil, fmt.Errorf("date border error:%w", err)
	}
	var allRrate []operatingData
	// запускаем цикл с проверкой дат до текущей включительно
	for d := start; d.Before(end) || d.Equal(end); d = d.AddDate(0, 0, 1) {

		dateOfTheRequestedDay := d.Format(s.FormatDate)

		body, err := s.Client.GetByDailyRate(dateOfTheRequestedDay)
		if err != nil {
			log.Fatal("GET error:", err)
			continue
		}

		daily, err := xmlDecoder(body)
		if err != nil {
			return nil, fmt.Errorf("xmlDecoder:%w", err)
		}
		parseDate, err := time.Parse("02.01.2006", daily.Date)
		if err != nil {
			return nil, fmt.Errorf("date parsing: %w", err)
		}

		// 24.05.2025
		for _, valute := range daily.Valute {
			allRrate = append(allRrate, operatingData{
				Date:     parseDate,
				CharCode: valute.CharCode,
				Name:     valute.Name,
				Value:    float64(valute.Value),
			})
		}
	}
	return allRrate, nil
}

type dataReturn struct {
	Min operatingData
	Max operatingData
	Avg float64
}

func GetMinMaxAvg(rates []operatingData) (*dataReturn, error) {
	if len(rates) == 0 {
		return nil, fmt.Errorf("length check error: %w", ErrUncorrectData)
	}
	// фильтруем  "Специальное права заимствования")

	var filteredRates []operatingData
	for _, rate := range rates {
		if rate.CharCode != "XDR" {
			filteredRates = append(filteredRates, rate)
		}
	}
	if len(filteredRates) == 0 {
		return nil, fmt.Errorf("length check error: %w", ErrDataAfterFiltering)
	}

	// набор данных с максимальной и минимальной ставкой ЦБ по валюте
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
	avg := total / float64(len(filteredRates))
	return &dataReturn{Min: min, Max: max, Avg: avg}, nil
}
