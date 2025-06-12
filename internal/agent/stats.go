package agent

import (
	"fmt"
	"net/http"
)

type InMemoryMetricsStore struct {
	gaugeMetrics map[string]string
	countMetrics map[string]int
}

func NewInMemoryMetricsStore() *InMemoryMetricsStore {
	return &InMemoryMetricsStore{
		gaugeMetrics: make(map[string]string),
		countMetrics: make(map[string]int),
	}
}

func (i *InMemoryMetricsStore) UpdateGaugeMetrics(metrics map[string]string) {
	for k, v := range metrics {
		fmt.Printf("update metric %s - %s\n", k, v)

		i.gaugeMetrics[k] = v
	}
}

func (i *InMemoryMetricsStore) IncreaseCountMetric(metricName string, by int) {
	if _, ok := i.countMetrics[metricName]; ok {
		i.countMetrics[metricName] += by
	} else {
		i.countMetrics[metricName] = 1
	}
}

func (i *InMemoryMetricsStore) GetGaugeMetrics() map[string]string {
	return i.gaugeMetrics
}

func (i *InMemoryMetricsStore) GetCountMetrics() map[string]int {
	return i.countMetrics
}

type HttpMetricsSender struct {
	url string
}

func NewHttpMetricsSender(url string) *HttpMetricsSender {
	return &HttpMetricsSender{url: url}
}

func (h *HttpMetricsSender) SendGaugeMetric(metricName string, metricValue string) {
	url := fmt.Sprintf("%s/update/gauge/%s/%s", h.url, metricName, metricValue)

	resp, err := http.Post(url, "text/plain", nil)
	if err != nil {
		fmt.Printf("Failed to send metric %s - %s to %s (%v)\n", metricName, metricValue, h.url, err)
		return
	}

	fmt.Printf("Sending metric %s - %s to %s status - %d\n", metricName, metricValue, url, resp.StatusCode)
}

func (h *HttpMetricsSender) SendCountMetric(metricName string, metricValue int) {
	url := fmt.Sprintf("%s/update/counter/%s/%d", h.url, metricName, metricValue)
	resp, err := http.Post(url, "text/plain", nil)

	if err != nil {
		fmt.Printf("Failed to send metric %s - %d to %s (%v)\n", metricName, metricValue, h.url, err)
		return
	}

	fmt.Printf("Sending metric %s - %d to %s status - %d\n", metricName, metricValue, url, resp.StatusCode)
}
