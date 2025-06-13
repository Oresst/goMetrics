package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type mockStore struct {
	updateGaugeMetricsCount  int
	increaseCountMetricCount int
}

func (s *mockStore) GetGaugeMetrics() map[string]string {
	return map[string]string{
		"Alloc": "10",
	}
}

func (s *mockStore) GetCountMetrics() map[string]int {
	return map[string]int{
		"counter": 10,
	}
}

func (s *mockStore) UpdateGaugeMetrics(metrics map[string]string) {
	s.updateGaugeMetricsCount++
}

func (s *mockStore) IncreaseCountMetric(metricName string, by int) {
	s.increaseCountMetricCount++
}

type mockSender struct {
	sendGaugeMetricCount int
	sendCountMetricCount int
}

func (s *mockSender) SendGaugeMetric(metricName string, metricValue string) {
	s.sendGaugeMetricCount++
}

func (s *mockSender) SendCountMetric(metricName string, metricValue int) {
	s.sendCountMetricCount++
}

func TestCollectStats(t *testing.T) {
	store := &mockStore{}
	sender := &mockSender{}
	collectInterval := 2 * time.Second
	sendInterval := 10 * time.Second
	service := NewCollectMetricsService(store, sender, collectInterval, sendInterval)

	defer func() {
		service.waitCollectStats <- true
	}()

	go service.collectStats()

	time.Sleep(collectInterval)
	assert.GreaterOrEqual(t, store.updateGaugeMetricsCount, 1)
	assert.GreaterOrEqual(t, store.increaseCountMetricCount, 1)
}

func TestSendStats(t *testing.T) {
	store := &mockStore{}
	sender := &mockSender{}
	collectInterval := 1 * time.Second
	sendInterval := 2 * time.Second

	service := NewCollectMetricsService(store, sender, collectInterval, sendInterval)

	defer func() {
		service.waitSendStats <- true
	}()

	go service.sendStats()

	time.Sleep(sendInterval)

	assert.GreaterOrEqual(t, sender.sendGaugeMetricCount, 1)
	assert.GreaterOrEqual(t, sender.sendCountMetricCount, 1)
}
