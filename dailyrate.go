package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// юзаем алиас для автопарсинга  xml.Unmarshal
type FloatWithComma float64

type ValCurs struct {
	// XMLNAME xml.Name `xml:"ValCurs"`
	Date string `xml:"Date,attr"`
	// Name   string   `xml:"name,attr"`
	Valute []Valute `xml:"Valute"`
}

// type Valute struct {
// 	// XMLName   xml.Name `xml:"Valute"`
// 	// ID        string `xml:"ID,attr"`
// 	// NumCode   string `xml:"NumCode"`
// 	CharCode string `xml:"CharCode"`
// 	// Nominal   int    `xml:"Nominal"`
// 	Name  string `xml:"Name"`
// 	Value string `xml:"Value"`
// 	// VunitRate string `xml:"VunitRate"`
// }

type Valute struct {
	// XMLName   xml.Name       `xml:"Valute"`
	// ID        string         `xml:"ID,attr"`
	// NumCode   string         `xml:"NumCode"`
	CharCode string `xml:"CharCode"`
	// Nominal   int            `xml:"Nominal"`
	Name  string         `xml:"Name"`
	Value FloatWithComma `xml:"Value"`
	// VunitRate FloatWithComma `xml:"VunitRate"`
}

// //делаем через метод UnmarshalXML интерфейса Unmarshaler

func (f *FloatWithComma) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// успел сделать автозамену при парсинге с апи
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		log.Fatal("Ошибка декодирования:", err)
	}
	s = strings.Replace(s, ",", ".", 1)
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatal("Ошибка преобразования:", err)
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

func main() {
	rates := in90DaysRates()
	//fmt.Println(rates)
	// businessLogic(rates)
	min, max, avg := businessLogic(rates)

	fmt.Printf("Минимальный курс на дату: %s\n", min.Date)
	fmt.Printf("Для валюты %s, c кодом %s\n", min.Name, min.CharCode)
	fmt.Printf("Составляет: %g\n", min.Value)

	fmt.Printf("Максимальный курс на дату: %s\n", max.Date)
	fmt.Printf("Для валюты %s, c кодом %s\n", max.Name, max.CharCode)
	fmt.Printf("Составляет: %g\n", max.Value)

	fmt.Printf("Cреднее значение курса рубля за весь период по всем валютам: %f\n", avg)
}

func in90DaysRates() []operatingData {
	// формируем границы дат
	end := time.Now()
	start := end.AddDate(0, 0, -90)
	var allRrate []operatingData
	// запускаем цикл с проверкой дат до текущей включительно
	for d := start; d.Before(end) || d.Equal(end); d = d.AddDate(0, 0, 1) {
		srtDate := d.Format("02/01/2006")
		// запрос на апи на каждую дату
		url := fmt.Sprintf("http://www.cbr.ru/scripts/XML_daily_eng.asp?date_req=%s", srtDate)
		// полный ручной запрос, так-как на АПИ ЦБ висит защита
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal("Ошибка запроса:", err)
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		c := &http.Client{
			Timeout: 5 * time.Second,
		}
		resp, err := c.Do(req)
		if resp.StatusCode != 200 {
			log.Fatal("Ошибка HTTP:", resp.Status)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("Ошибка чтения:", err)
			continue
		}

		//апи ЦБ возвращает данные в кодировке  Windows-1251, поэтому перекодируем в потоке

		decoder := xml.NewDecoder(bytes.NewReader(body))
		decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
			if charset == "windows-1251" {
				return transform.NewReader(input, charmap.Windows1251.NewDecoder()), nil
			}
			return nil, fmt.Errorf("неподдерживаемая кодировка: %s", charset)
		}
		var daily ValCurs
		if err := decoder.Decode(&daily); err != nil {
			log.Fatal("Ошибка парсинга XML:", err)
			continue
		}
		// for _, valute := range daily.Value {
		// 	stringVal := strings.Replace(valute, ",", ".", 1)
		// 	valinFloat64 := strconv.ParseFloat(stringVal, 64)
		// 	if err != nil {
		// 		log.Fatal("Ошибка преобразования в float64:", err)
		// 		continue
		// 	}
		// }
		//log.Println("Парсинг выполнен")

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
	return allRrate
}

func businessLogic(rates []operatingData) (min, max operatingData, avg float64) {
	if len(rates) == 0 {
		log.Fatalf("Не корретные данные:")
	}
	// фильтруем  Специальное права заимствования)

	var filteredRates []operatingData
	for _, rate := range rates {
		if rate.CharCode != "XDR" {
			filteredRates = append(filteredRates, rate)
		}
	}
	if len(filteredRates) == 0 {
		log.Fatal("Нет данных после фильтрации XDR")
	}

	// набор данных с максимальной и минимальной ставкой ЦБ по валюте
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
	avg = total / float64(len(filteredRates))
	// fmt.Printf("%s,%s,%g\n", max.Date, max.Name, max.Value)
	// fmt.Printf("%s,%s,%g\n", min.Date, min.Name, min.Value)
	return
}
