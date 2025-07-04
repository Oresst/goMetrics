package agent

import (
	"fmt"
	log "github.com/sirupsen/logrus"
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
	place := "UpdateGaugeMetrics"
	i.Lock()
	defer i.Unlock()

	for k, v := range metrics {
		log.WithFields(log.Fields{
			"metric": k,
			"value":  v,
			"place":  place,
		}).Info("update metric")

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
	place := "SendGaugeMetric"
	url := fmt.Sprintf("%s/update/gauge/%s/%s", h.url, metricName, metricValue)

	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"metricName":  metricName,
			"metricValue": metricValue,
			"url":         url,
			"error":       err,
			"place":       place,
		}).Error("Failed to send metric")
		return
	}

	var resp *http.Response
	resp, err = h.retryHTTP(request, 3, 300*time.Microsecond)()
	if err != nil {
		log.WithFields(log.Fields{
			"metricName":  metricName,
			"metricValue": metricValue,
			"url":         url,
			"error":       err,
			"place":       place,
		}).Error("Failed to send metric")
		return
	}
	defer resp.Body.Close()

	log.WithFields(log.Fields{
		"metricName":  metricName,
		"metricValue": metricValue,
		"url":         url,
		"place":       place,
		"statusCode":  resp.StatusCode,
	}).Info("Sent metric")
}

func (h *HTTPMetricsSender) SendCountMetric(metricName string, metricValue int) {
	place := "SendCountMetric"
	url := fmt.Sprintf("%s/update/counter/%s/%d", h.url, metricName, metricValue)

	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"metricName":  metricName,
			"metricValue": metricValue,
			"url":         url,
			"error":       err,
			"place":       place,
		}).Error("Failed to send metric")
		return
	}

	var resp *http.Response
	resp, err = h.retryHTTP(request, 3, 300*time.Microsecond)()
	if err != nil {
		log.WithFields(log.Fields{
			"metricName":  metricName,
			"metricValue": metricValue,
			"url":         url,
			"error":       err,
			"place":       place,
		}).Error("Failed to send metric")
		return
	}
	defer resp.Body.Close()

	log.WithFields(log.Fields{
		"metricName":  metricName,
		"metricValue": metricValue,
		"url":         url,
		"place":       place,
		"statusCode":  resp.StatusCode,
	}).Info("Sent metric")
}

func (h *HTTPMetricsSender) retryHTTP(
	request *http.Request,
	retries int,
	delay time.Duration,
) func() (*http.Response, error) {
	return func() (*http.Response, error) {
		place := "retryHTTP"

		for i := 0; i < retries; i++ {
			response, err := h.client.Do(request)

			if err == nil {
				return response, nil
			}

			time.Sleep(delay)

			log.WithFields(log.Fields{
				"url":     request.URL.String(),
				"method":  request.Method,
				"attempt": i + 1,
				"place":   place,
				"delay":   delay,
			}).Info("retry to send request")
		}

		return nil, fmt.Errorf("retries exceed")
	}
}
