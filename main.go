package main

import (
	"fmt"
	"log"
	client "main/Client"
	config "main/Config"
	service "main/Service"
)

const (
	configYaml = "config.yaml"
	userAgent  = "User-Agent"
)

func main() {
	newConfig, err := config.NewConfig(configYaml)
	if err != nil {
		log.Fatal("loag yaml file:", err)
	}

	newService := &service.NewService{
		FormatDate: newConfig.DateF.DateFormat,
		Client:     client.NewHttpClient(newConfig.Api.Timeout, newConfig.Api.BaseUrl, newConfig.Api.UserAgent),
	}

	rates, err := newService.GetIn90DaysRates()
	if err != nil {
		log.Fatal("Error getting data for 90 days:", err)
	}

	dataReturn, err := service.GetMinMaxAvg(rates)
	if err != nil {
		log.Fatal("Error calculation Min Max Avg:", err)
	}

	fmt.Printf("Минимальный курс на дату: %s\n", dataReturn.Min.Date)
	fmt.Printf("Для валюты %s, c кодом %s\n", dataReturn.Min.Name, dataReturn.Min.CharCode)
	fmt.Printf("Составляет: %g\n", dataReturn.Min.Value)
	fmt.Printf("Максимальный курс на дату: %s\n", dataReturn.Max.Date)
	fmt.Printf("Для валюты %s, c кодом %s\n", dataReturn.Max.Name, dataReturn.Max.CharCode)
	fmt.Printf("Составляет: %g\n", dataReturn.Max.Value)
	fmt.Printf("Cреднее значение курса рубля за весь период по всем валютам: %f\n", dataReturn.Avg)
}
