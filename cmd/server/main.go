package main

import (
	"fmt"
	"github.com/Oresst/goMetrics/internal/store"
	"github.com/Oresst/goMetrics/models"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type metricsService struct {
	storage store.Store
}

func newMetricsService(storage store.Store) *metricsService {
	return &metricsService{
		storage: storage,
	}
}

func (m *metricsService) addMetricHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	url := r.URL.Path
	url = strings.Trim(url, "/")
	urlParams := strings.Split(url, "/")

	if len(urlParams) != 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metricType := urlParams[1]
	if metricType == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if metricType != models.Counter && metricType != models.Gauge {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metricName := urlParams[2]
	if metricName == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metricValueStr := urlParams[3]
	if metricValueStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metricValue, err := strconv.ParseFloat(metricValueStr, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Printf("Got metric name - %s type - %s value - %f\n", metricName, metricType, metricValue)

	err = m.storage.AddMetric(metricType, metricName, metricValue)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	storage := getStorage()
	service := newMetricsService(storage)

	if err := runServer(service); err != nil {
		log.Fatal(err)
	}
}

func getStorage() store.Store {
	return store.NewMemStorage()
}

func runServer(service *metricsService) error {
	http.HandleFunc("/update/{type}/{name}/{value}/", service.addMetricHandler)

	return http.ListenAndServe(":8080", nil)
}
