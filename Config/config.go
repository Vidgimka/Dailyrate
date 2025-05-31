package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Api ApiConfig `yaml:"api"`
	// DateF DateConfig `yaml:"formats_date"`
}
type ApiConfig struct {
	Timeout    time.Duration `yaml:"timeout"`
	BaseUrl    string        `yaml:"base_url"`
	UserAgent  string        `yaml:"user_agen"`
	DateFormat string        `yaml:"date_format"`
}

// type DateConfig struct {
// 	DateFormat string `yaml:"date_format"`
// }

func NewConfig(path string) (*Config, error) {
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
