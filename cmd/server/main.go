package main

import (
	"fmt"
	"github.com/Oresst/goMetrics/internal/store"
	"github.com/Oresst/goMetrics/models"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"strconv"
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

	metricType := chi.URLParam(r, "type")
	if metricType == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if metricType != models.Counter && metricType != models.Gauge {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metricName := chi.URLParam(r, "name")
	if metricName == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metricValueStr := chi.URLParam(r, "value")
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

func (m *metricsService) getMetricHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	metricType := chi.URLParam(r, "type")
	if metricType == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if metricType != models.Counter && metricType != models.Gauge {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metricName := chi.URLParam(r, "name")
	if metricName == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metricValue, err := m.storage.GetMetric(metricName)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%f", metricValue)))
}

func main() {
	storage := getStorage()
	service := newMetricsService(storage)
	r := getRouter(service)

	if err := runServer(r); err != nil {
		log.Fatal(err)
	}
}

func getStorage() store.Store {
	return store.NewMemStorage()
}

func getRouter(service *metricsService) *chi.Mux {
	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", service.addMetricHandler)
	r.Get("/value/{type}/{name}", service.getMetricHandler)

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(405)
	})

	return r
}

func runServer(r *chi.Mux) error {
	return http.ListenAndServe(":8080", r)
}
