package service

import (
	"fmt"
	"log"
	client "main/clientapi"
	"time"
)

// 	CharCode string         `xml:"CharCode"`
// 	Name     string         `xml:"Name"`
// 	Value    FloatWithComma `xml:"Value"`
// }

type operatingData struct {
	Date     time.Time
	CharCode string
	Name     string
	Value    float64
}

func dateBorder() (time.Time, time.Time, error) {
	end := time.Now()
	start := end.AddDate(0, 0, -90)
	return start, end, nil
}

type ServiceInterface interface {
	GetIn90DaysRates() ([]operatingData, error)
}

type service struct {
	client client.HttpClientIntrface
}

func NewService(httpClient client.HttpClientIntrface) ServiceInterface {
	return &service{
		client: httpClient,
	}
}

// функция запроса данных с API ЦБ в течении 90 дней с момента даты запроса
func (s *service) GetIn90DaysRates() ([]operatingData, error) {
	start, end, err := dateBorder()
	if err != nil {
		return nil, fmt.Errorf("date border error:%w", err)
	}
	var allRrate []operatingData
	// запускаем цикл с проверкой дат до текущей включительно
	for d := start; d.Before(end) || d.Equal(end); d = d.AddDate(0, 0, 1) {

		// dateOfTheRequestedDay := d.Format(s.FormatDate)

		daily, err := s.client.GetByDailyRate(d)
		if err != nil {
			log.Fatal("GET error:", err)
			continue
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
