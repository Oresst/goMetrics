package agent

type StatsStore interface {
	GetGaugeMetrics() map[string]string
	GetCountMetrics() map[string]int
	UpdateGaugeMetrics(metrics map[string]string)
	IncreaseCountMetric(metricName string, by int)
}

type StatsSender interface {
	SendGaugeMetric(metricName string, metricValue string)
	SendCountMetric(metricName string, metricValue int)
	SendMetricJSON(metricName string, metricType string, value string)
}
