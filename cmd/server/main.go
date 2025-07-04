package main

import (
	"flag"
	"fmt"
	"github.com/Oresst/goMetrics/internal/services"
	"github.com/Oresst/goMetrics/internal/store"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	address := flag.String("a", ":8080", "server port")
	flag.Parse()

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		*address = envAddress
	}

	addressArray := strings.Split(*address, ":")
	if len(addressArray) != 2 {
		log.Fatalf("Wrong address format: %s in env variable ADDRESS", *address)
	}
	*address = addressArray[1]

	storage := getStorage()
	service := services.NewMetricsService(storage)
	r := getRouter(service)

	if err := runServer(*address, r); err != nil {
		log.Fatal(err)
	}
}

func getStorage() store.Store {
	return store.NewMemStorage()
}

func getRouter(service *services.MetricsService) *chi.Mux {
	r := chi.NewRouter()

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
