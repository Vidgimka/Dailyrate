package service

import "errors"

var (
	// Валидация входных данных
	ErrUncorrectData      = errors.New("uncorrect data")
	ErrDataAfterFiltering = errors.New("no data after filtering")
)
