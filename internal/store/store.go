package store

type Store interface {
	AddMetric(metricType string, name string, value float64) error
	getMetric(name string) (float64, error)
}
