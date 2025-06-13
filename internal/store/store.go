package store

type Store interface {
	AddMetric(metricType string, name string, value float64) error
	GetMetric(name string) (float64, error)
}
