package main

import (
	"flag"
	"fmt"
	"github.com/Oresst/goMetrics/internal/services"
	"github.com/Oresst/goMetrics/internal/store"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
	"time"
)

func initLogger() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	address := flag.String("a", ":8080", "server port")
	flag.Parse()

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		*address = envAddress
	}

	initLogger()

	addressArray := strings.Split(*address, ":")
	if len(addressArray) != 2 {
		log.WithFields(log.Fields{
			"address": *address,
		}).Info("Wrong address format in env variable ADDRESS")
	}
	*address = addressArray[1]

	log.WithFields(log.Fields{
		"address": *address,
	}).Info("Run with args")

	storage := getStorage()
	fileService, err := services.NewFileService("/Users/maksimpanasenko/Desktop/goMetrics/test.txt", time.Second*200)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal(err)
	}
	fileService.Run()
	service := services.NewMetricsService(storage, fileService)
	r := getRouter(service)

	if err := runServer(*address, r); err != nil {
		log.WithFields(log.Fields{
			"address": *address,
		}).Fatal(err)
	}
}

func getStorage() store.Store {
	return store.NewMemStorage()
}

func getRouter(service *services.MetricsService) *chi.Mux {
	r := chi.NewRouter()

	r.Use(service.LoggerMiddleware)
	r.Use(service.GzipMiddleware)

	r.Route("/update/{type}/{name}/{value}", func(r chi.Router) {
		r.Post("/", service.AddMetricHandler)
	})
	r.Route("/update", func(r chi.Router) {
		r.Post("/", service.AddMetricJSONHandler)
	})
	r.Route("/value", func(r chi.Router) {
		r.Post("/", service.GetMetricJSONHandler)
	})
	r.Route("/value/{type}/{name}", func(r chi.Router) {
		r.Get("/", service.GetMetricHandler)
	})
	r.Get("/", service.GetAllMetricsHandler)

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(405)
	})

	return r
}

func runServer(port string, r *chi.Mux) error {
	return http.ListenAndServe(fmt.Sprintf(":%s", port), r)
}
