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
	service := services.NewMetricsService(storage)
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

	r.Post("/update/{type}/{name}/{value}", service.AddMetricHandler)
	r.Get("/value/{type}/{name}", service.GetMetricHandler)
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
