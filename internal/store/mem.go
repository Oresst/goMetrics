package store

import (
	"errors"
	"github.com/Oresst/goMetrics/models"
	"github.com/google/uuid"
	"sync"
)

type MemStorage struct {
	metrics map[string]models.Metrics
	sync.Mutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metrics),
	}
}

func (m *MemStorage) AddMetric(metricType string, name string, value float64) error {
	m.Lock()
	defer m.Unlock()

	uuid4 := uuid.New()

	metric, ok := m.metrics[name]
	if !ok {
		metric = models.Metrics{
			ID:    uuid4.String(),
			MType: metricType,
			Value: &value,
		}

		m.metrics[name] = metric
		return nil
	}

	if metric.MType == models.Counter {
		*metric.Value += value
	} else if metric.MType == models.Gauge {
		*metric.Value = value
	} else {
		return errors.New("unknown metric type")
	}

	return nil
}

func (m *MemStorage) GetMetric(name string) (models.Metrics, error) {
	m.Lock()
	defer m.Unlock()

	metric, ok := m.metrics[name]
	if !ok {
		return models.Metrics{}, errors.New("metric not found")
	}

	return metric, nil
}

func (m *MemStorage) GetAllMetrics() map[string]models.Metrics {
	m.Lock()
	defer m.Unlock()

	return m.metrics
}
