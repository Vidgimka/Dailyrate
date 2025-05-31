package client

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type FloatWithComma float64

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

type Valute struct {
	CharCode string         `xml:"CharCode"`
	Name     string         `xml:"Name"`
	Value    FloatWithComma `xml:"Value"`
}
type ValCurs struct {
	Date   string   `xml:"Date,attr"`
	Valute []Valute `xml:"Valute"`
}

func xmlDecoderInValCurs(body []byte) (ValCurs, error) {
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
		return ValCurs{}, fmt.Errorf("xml parsing error:%w", err)
	}
	return daily, nil
}
