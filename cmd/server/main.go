package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/Oresst/goMetrics/internal/services"
	"github.com/Oresst/goMetrics/internal/store"
	"github.com/Oresst/goMetrics/internal/utils"
	"github.com/Oresst/goMetrics/models"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func initLogger() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	address := flag.String("a", ":8080", "server port")
	interval := flag.Int("i", 300, "save interval in seconds")
	filePath := flag.String("f", "metrics.txt", "file path")
	restore := flag.Bool("r", false, "restore metrics")
	flag.Parse()

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		*address = envAddress
	}

	if envInterval := os.Getenv("STORE_INTERVAL"); envInterval != "" {
		*interval = utils.StrToInt(envInterval, *interval)
	}

	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		*filePath = envFilePath
	}

	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		*restore = envRestore == "true"
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

	fileService, err := services.NewFileService(*filePath, time.Second*time.Duration(*interval))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal(err)
	}
	fileService.Run()

	storage := getStorage()

	if *restore {
		data, err := fileService.ReadAllData(*filePath)

		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Fatal("Ошибка считывания файла")
		}

		for _, metric := range data {
			if metric.MType == models.Gauge {
				err := storage.AddMetric(metric.MType, metric.ID, *metric.Value)

				if err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
					}).Fatal("Ошибка загрузки метрик")
				}
			} else if metric.MType == models.Counter {
				err := storage.AddMetric(metric.MType, metric.ID, float64(*metric.Delta))
				if err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
					}).Fatal("Ошибка загрузки метрик")
				}
			} else {
				log.WithFields(log.Fields{
					"type": metric.MType,
					"id":   metric.ID,
				}).Info("Неверный тип метрики")
			}
		}
	}

	service := services.NewMetricsService(storage, fileService)
	r := getRouter(service)

	if err := runServer(*address, r); err != nil {
		log.WithFields(log.Fields{
			"address": *address,
		}).Fatal(err)
	}

	fileService.Stop()
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
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка при запуске сервера: %s", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return server.Shutdown(ctx)
}
