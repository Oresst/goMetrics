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
	"github.com/jackc/pgx"
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
	dsn := flag.String("d", "", "database connection string")
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

	if envDsn := os.Getenv("DATABASE_DSN"); envDsn != "" {
		*dsn = envDsn
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
		"address":  *address,
		"dsn":      *dsn,
		"filePath": *filePath,
		"restore":  *restore,
		"interval": *interval,
	}).Info("Run with args")

	fileService, err := services.NewFileService(*filePath, time.Second*time.Duration(*interval))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal(err)
	}
	fileService.Run()

	var storage store.Store
	var db *pgx.Conn
	if *dsn != "" {
		config, err := pgx.ParseConnectionString(*dsn)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Fatal("Unable to parse connection string")
		}

		db, err = pgx.Connect(config)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Fatal("Unable to connect to database")
		}

		defer db.Close()

		storage = getDBStorage(db)
	} else {
		storage = getStorageMem()
	}

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

	if db != nil {
		addPingHandler(r, db)
	}

	if err := runServer(*address, r); err != nil {
		log.WithFields(log.Fields{
			"address": *address,
		}).Fatal(err)
	}

	fileService.Stop()
}

func getStorageMem() store.Store {
	return store.NewMemStorage()
}

func getDBStorage(db *pgx.Conn) store.Store {
	return store.NewDBStorage(db)
}

func addPingHandler(r chi.Router, db *pgx.Conn) {
	r.Route("/ping", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := db.Ping(ctx)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
		})
	})
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
