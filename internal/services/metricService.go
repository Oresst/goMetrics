package services

import (
	"fmt"
	"github.com/Oresst/goMetrics/internal/store"
	"github.com/Oresst/goMetrics/internal/utils"
	"github.com/Oresst/goMetrics/models"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const template = `
	<html>
		<head>
		<title>Metrics</title>
		</head>
		<body>
			<h1>Метрики</h1>
			<ul>%s</ul>
		</body>
	</html>
`

type MetricsService struct {
	storage store.Store
}

func NewMetricsService(storage store.Store) *MetricsService {
	return &MetricsService{
		storage: storage,
	}
}

func (m *MetricsService) AddMetricHandler(w http.ResponseWriter, r *http.Request) {
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
	log.Printf("Got metric name - %s type - %s value - %f\n", metricName, metricType, metricValue)

	err = m.storage.AddMetric(metricType, metricName, metricValue)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (m *MetricsService) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
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
	w.Write([]byte(utils.BetterFormat(metricValue)))
}

func (m *MetricsService) GetAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
	allMetrics := m.storage.GetAllMetrics()
	strMetrics := make([]string, len(allMetrics))

	for k, v := range allMetrics {
		url := fmt.Sprintf("/value/%s/%s", v.MType, k)
		strMetrics = append(strMetrics, fmt.Sprintf("<li><a href=\"%s\">%s</a></li>", url, k))
	}

	responseText := fmt.Sprintf(template, strings.Join(strMetrics, "\n"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, responseText)
}
