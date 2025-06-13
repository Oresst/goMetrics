package main

import (
	"fmt"
	"github.com/Oresst/goMetrics/internal/agent"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type CollectMetricsService struct {
	collectInterval time.Duration
	sendInterval    time.Duration

	store  agent.StatsStore
	sender agent.StatsSender

	waitCollectStats chan bool
	waitSendStats    chan bool

	wg sync.WaitGroup
}

func NewCollectMetricsService(
	store agent.StatsStore,
	sender agent.StatsSender,
	collectInterval time.Duration,
	sendInterval time.Duration,
) *CollectMetricsService {
	return &CollectMetricsService{
		collectInterval: collectInterval,
		sendInterval:    sendInterval,

		store:  store,
		sender: sender,

		waitCollectStats: make(chan bool),
		waitSendStats:    make(chan bool),
	}
}

func (s *CollectMetricsService) Run() {
	go func() {
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
		<-exit
		s.stop()
	}()

	go s.collectStats()
	go s.sendStats()

	s.wg.Wait()

	fmt.Println("stopped")
}

func (s *CollectMetricsService) stop() {
	s.waitCollectStats <- true
	s.waitSendStats <- true
}

func (s *CollectMetricsService) collectStats() {
	s.wg.Add(1)
	defer s.wg.Done()

	fmt.Println("starting collect stats")

	gaugeMetrics := make(map[string]string)
	var memStats runtime.MemStats

	for {
		fmt.Println("collecting stats")

		runtime.ReadMemStats(&memStats)

		gaugeMetrics["Alloc"] = fmt.Sprintf("%d", memStats.Alloc)
		gaugeMetrics["BuckHashSys"] = fmt.Sprintf("%d", memStats.BuckHashSys)
		gaugeMetrics["Frees"] = fmt.Sprintf("%d", memStats.Frees)
		gaugeMetrics["GCCPUFraction"] = fmt.Sprintf("%f", memStats.GCCPUFraction)
		gaugeMetrics["GCSys"] = fmt.Sprintf("%d", memStats.GCSys)
		gaugeMetrics["HeapAlloc"] = fmt.Sprintf("%d", memStats.HeapAlloc)
		gaugeMetrics["HeapIdle"] = fmt.Sprintf("%d", memStats.HeapIdle)
		gaugeMetrics["HeapInuse"] = fmt.Sprintf("%d", memStats.HeapInuse)
		gaugeMetrics["HeapObjects"] = fmt.Sprintf("%d", memStats.HeapObjects)
		gaugeMetrics["HeapReleased"] = fmt.Sprintf("%d", memStats.HeapReleased)
		gaugeMetrics["HeapSys"] = fmt.Sprintf("%d", memStats.HeapSys)
		gaugeMetrics["LastGC"] = fmt.Sprintf("%d", memStats.LastGC)
		gaugeMetrics["Lookups"] = fmt.Sprintf("%d", memStats.Lookups)
		gaugeMetrics["MCacheInuse"] = fmt.Sprintf("%d", memStats.MCacheInuse)
		gaugeMetrics["MCacheSys"] = fmt.Sprintf("%d", memStats.MCacheSys)
		gaugeMetrics["MSpanInuse"] = fmt.Sprintf("%d", memStats.MSpanInuse)
		gaugeMetrics["MSpanSys"] = fmt.Sprintf("%d", memStats.MSpanSys)
		gaugeMetrics["Mallocs"] = fmt.Sprintf("%d", memStats.Mallocs)
		gaugeMetrics["NextGC"] = fmt.Sprintf("%d", memStats.NextGC)
		gaugeMetrics["NumForcedGC"] = fmt.Sprintf("%d", memStats.NumForcedGC)
		gaugeMetrics["NumGC"] = fmt.Sprintf("%d", memStats.NumGC)
		gaugeMetrics["OtherSys"] = fmt.Sprintf("%d", memStats.OtherSys)
		gaugeMetrics["PauseTotalNs"] = fmt.Sprintf("%d", memStats.PauseTotalNs)
		gaugeMetrics["StackInuse"] = fmt.Sprintf("%d", memStats.StackInuse)
		gaugeMetrics["StackSys"] = fmt.Sprintf("%d", memStats.StackSys)
		gaugeMetrics["Sys"] = fmt.Sprintf("%d", memStats.Sys)
		gaugeMetrics["TotalAlloc"] = fmt.Sprintf("%d", memStats.TotalAlloc)
		gaugeMetrics["RandomValue"] = fmt.Sprintf("%d", rand.Int())

		s.store.UpdateGaugeMetrics(gaugeMetrics)
		s.store.IncreaseCountMetric("PollCount", 1)

		select {
		case <-s.waitCollectStats:
			fmt.Println("Stop collect stats")
			return
		case <-time.After(s.collectInterval):
			continue
		}
	}
}

func (s *CollectMetricsService) sendStats() {
	s.wg.Add(1)
	defer s.wg.Done()
	fmt.Println("starting send stats")

	for {
		fmt.Println("sending stats")

		var wg sync.WaitGroup
		gougeMetricStats := s.store.GetGaugeMetrics()

		for key, value := range gougeMetricStats {
			wg.Add(1)

			go func(metricName string, value string) {
				defer wg.Done()
				s.sender.SendGaugeMetric(metricName, value)
			}(key, value)
		}

		countMetrics := s.store.GetCountMetrics()
		for key, value := range countMetrics {
			wg.Add(1)

			go func(metricName string, value int) {
				defer wg.Done()
				s.sender.SendCountMetric(metricName, value)
			}(key, value)
		}

		wg.Wait()

		select {
		case <-s.waitSendStats:
			fmt.Println("Stop sending stats")
			return
		case <-time.After(s.sendInterval):
			continue
		}
	}
}

func main() {
	store := agent.NewInMemoryMetricsStore()
	sender := agent.NewHTTPMetricsSender("http://localhost:8080")

	service := NewCollectMetricsService(store, sender, 2*time.Second, 2*time.Second)
	service.Run()
}
