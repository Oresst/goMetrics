package store

import "github.com/Oresst/goMetrics/models"

type Store interface {
	AddMetric(metricType string, name string, value float64) error
	GetMetric(name string) (models.Metrics, error)
	GetAllMetrics() map[string]models.Metrics
}
