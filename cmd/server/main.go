package main

import (
	"flag"
	"fmt"
	"github.com/Oresst/goMetrics/internal/store"
	"github.com/Oresst/goMetrics/models"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"net/http"
	"os"
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

func betterFormat(num float64) string {
	s := fmt.Sprintf("%f", num)
	return strings.TrimRight(strings.TrimRight(s, "0"), ".")
}

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
	w.Write([]byte(betterFormat(metricValue)))
}

func (m *metricsService) getAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
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

func main() {
	port := flag.String("a", "8080", "server port")
	fmt.Printf("Got port: %s from flags\n", *port)
	flag.Parse()

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		fmt.Printf("Got envAddress: %s from envs\n", envAddress)

		addressArray := strings.Split(envAddress, ":")

		if len(addressArray) != 2 {
			log.Fatalf("Wrong address format: %s in env variable ADDRESS", envAddress)
		}

		*port = strings.Split(envAddress, ":")[1]
	}

	storage := getStorage()
	service := newMetricsService(storage)
	r := getRouter(service)

	if err := runServer(*port, r); err != nil {
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
	r.Get("/", service.getAllMetricsHandler)

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(405)
	})

	return r
}

func runServer(port string, r *chi.Mux) error {
	return http.ListenAndServe(fmt.Sprintf(":%s", port), r)
}
