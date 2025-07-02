package agent

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type InMemoryMetricsStore struct {
	gaugeMetrics map[string]string
	countMetrics map[string]int

	sync.RWMutex
}

func NewInMemoryMetricsStore() *InMemoryMetricsStore {
	return &InMemoryMetricsStore{
		gaugeMetrics: make(map[string]string),
		countMetrics: make(map[string]int),
	}
}

func (i *InMemoryMetricsStore) UpdateGaugeMetrics(metrics map[string]string) {
	i.Lock()
	defer i.Unlock()

	for k, v := range metrics {
		log.Printf("update metric %s - %s\n", k, v)

		i.gaugeMetrics[k] = v
	}
}

func (i *InMemoryMetricsStore) IncreaseCountMetric(metricName string, by int) {
	i.Lock()
	defer i.Unlock()

	if _, ok := i.countMetrics[metricName]; ok {
		i.countMetrics[metricName] += by
	} else {
		i.countMetrics[metricName] = 1
	}
}

func (i *InMemoryMetricsStore) GetGaugeMetrics() map[string]string {
	i.Lock()
	defer i.Unlock()

	return i.gaugeMetrics
}

func (i *InMemoryMetricsStore) GetCountMetrics() map[string]int {
	i.Lock()
	defer i.Unlock()

	return i.countMetrics
}

type HTTPMetricsSender struct {
	url    string
	client *http.Client
}

func NewHTTPMetricsSender(url string) *HTTPMetricsSender {
	return &HTTPMetricsSender{
		url: url,
		client: &http.Client{
			Timeout: 1 * time.Second,
		},
	}
}

func (h *HTTPMetricsSender) SendGaugeMetric(metricName string, metricValue string) {
	url := fmt.Sprintf("%s/update/gauge/%s/%s", h.url, metricName, metricValue)

	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		log.Printf("Failed to send metric %s - %s to %s (%v)\n", metricName, metricValue, h.url, err)
		return
	}

	var resp *http.Response
	resp, err = h.retryHTTP(request, 3, 300*time.Microsecond)()
	if err != nil {
		log.Printf("Failed to send metric %s - %s to %s (%v)\n", metricName, metricValue, h.url, err)
		return
	}

	defer resp.Body.Close()

	log.Printf("Sending metric %s - %s to %s status - %d\n", metricName, metricValue, url, resp.StatusCode)
}

func (h *HTTPMetricsSender) SendCountMetric(metricName string, metricValue int) {
	url := fmt.Sprintf("%s/update/counter/%s/%d", h.url, metricName, metricValue)

	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		log.Printf("Failed to send metric %s - %d to %s (%v)\n", metricName, metricValue, h.url, err)
		return
	}

	var resp *http.Response
	resp, err = h.retryHTTP(request, 3, 300*time.Microsecond)()

	if err != nil {
		log.Printf("Failed to send metric %s - %d to %s (%v)\n", metricName, metricValue, h.url, err)
		return
	}

	defer resp.Body.Close()

	log.Printf("Sending metric %s - %d to %s status - %d\n", metricName, metricValue, url, resp.StatusCode)
}

func (h *HTTPMetricsSender) retryHTTP(
	request *http.Request,
	retries int,
	delay time.Duration,
) func() (*http.Response, error) {
	return func() (*http.Response, error) {
		for i := 0; i < retries; i++ {
			response, err := h.client.Do(request)

			if err == nil {
				return response, nil
			}

			time.Sleep(delay)
			log.Printf("retry %d to send request to url: %s", i, request.URL.String())
		}

		return &http.Response{}, fmt.Errorf("retries exceed")
	}
}
