package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Oresst/goMetrics/internal/store"
	"github.com/Oresst/goMetrics/internal/utils"
	"github.com/Oresst/goMetrics/models"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
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

func (m *MetricsService) LoggerMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		data := &responseData{}
		writer := loggerResponseWriter{ResponseWriter: w, data: data}

		next.ServeHTTP(&writer, r)

		log.WithFields(log.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"duration":   time.Since(start),
			"statusCode": data.statusCode,
			"size":       data.size,
		}).Info()
	})
}

func (m *MetricsService) AddMetricHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	place := "[AddMetricHandler]"

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

	log.WithFields(log.Fields{
		"place":      place,
		"metricName": metricName,
		"type":       metricType,
		"value":      metricValue,
	}).Info("New metric")

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

func (m *MetricsService) AddMetricJSONHandler(w http.ResponseWriter, r *http.Request) {
	place := "[MetricsService.AddMetricHandler]"

	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rawData, _ := io.ReadAll(r.Body)
	log.WithFields(log.Fields{
		"place":   place,
		"rawData": string(rawData),
	}).Info("input data")
	buffered := bytes.NewBuffer(rawData)
	decoder := json.NewDecoder(buffered)
	defer r.Body.Close()

	var data models.Metrics

	err := decoder.Decode(&data)
	if err != nil {
		log.WithFields(log.Fields{
			"place": place,
			"error": err.Error(),
		}).Error("Ошибка при парсинге JSON")

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	if data.MType != models.Counter && data.MType != models.Gauge {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Поле type должно быть равно %s или %s", models.Counter, models.Gauge)))
		return
	}

	if data.MType == models.Counter && data.Value == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Поле value обязательно при type %s", models.Counter)))
		return
	}

	if data.MType == models.Gauge && data.Delta == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Поле delta обязательно при type %s", models.Gauge)))
		return
	}

	if data.MType == models.Counter {
		err = m.storage.AddMetric(data.MType, data.ID, *data.Value)

		if err != nil {
			log.WithFields(log.Fields{
				"place": place,
				"error": err.Error(),
				"type":  data.MType,
				"value": *data.Value,
				"id":    data.ID,
			}).Error("Ошибка при добавлении метрики")

			w.WriteHeader(http.StatusBadRequest)
			return
		}

	} else if data.MType == models.Gauge {
		err = m.storage.AddMetric(data.MType, data.ID, float64(*data.Delta))

		if err != nil {
			log.WithFields(log.Fields{
				"place": place,
				"error": err.Error(),
				"type":  data.MType,
				"value": *data.Value,
				"id":    data.ID,
			}).Error("Ошибка при добавлении метрики")

			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (m *MetricsService) GetMetricJSONHandler(w http.ResponseWriter, r *http.Request) {
	place := "[MetricsService.GetMetricJSONHandler]"

	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestRawData, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		log.WithFields(log.Fields{
			"place": place,
			"error": err.Error(),
			"data":  string(requestRawData),
		}).Error("Ошибка чтения r.Body")

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var data models.Metrics
	err = json.Unmarshal(requestRawData, &data)
	if err != nil {
		log.WithFields(log.Fields{
			"place": place,
			"error": err.Error(),
			"data":  string(requestRawData),
		}).Error("Ошибка парсинга JSON")

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if data.MType != models.Counter && data.MType != models.Gauge {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Поле type должно быть равно %s или %s", models.Counter, models.Gauge)))
		return
	}

	if data.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Поле id не может быть пустым"))
	}

	var metric float64
	metric, err = m.storage.GetMetric(data.ID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	metric, err = strconv.ParseFloat(utils.BetterFormat(metric), 64)
	if err != nil {
		log.WithFields(log.Fields{
			"place": place,
			"error": err.Error(),
			"value": metric,
		}).Error("Ошибка парсинга метрики")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responseData := struct {
		ID    string  `json:"id"`
		Type  string  `json:"type"`
		Value float64 `json:"value"`
	}{
		ID:    data.ID,
		Type:  data.MType,
		Value: metric,
	}

	var responseRawData []byte
	responseRawData, err = json.Marshal(responseData)
	if err != nil {
		log.WithFields(log.Fields{
			"place": place,
			"error": err.Error(),
			"Id":    data.ID,
			"Type":  data.MType,
			"value": metric,
		}).Error("Ошибка сериализации JSON")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseRawData)
}
