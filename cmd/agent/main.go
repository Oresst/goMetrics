package main

import (
	"flag"
	"fmt"
	"github.com/Oresst/goMetrics/internal/agent"
	"github.com/Oresst/goMetrics/internal/services"
	"github.com/Oresst/goMetrics/internal/utils"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

func initLogger() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	address := flag.String("a", "0.0.0.0:8080", "server port")
	reportInterval := flag.Int("r", 10, "report interval in seconds")
	pollInterval := flag.Int("p", 2, "poll interval in seconds")
	flag.Parse()

	addressEnv := os.Getenv("ADDRESS")
	reportIntervalEnv := os.Getenv("REPORT_INTERVAL")
	pollIntervalEnv := os.Getenv("POLL_INTERVAL")

	if addressEnv != "" {
		*address = addressEnv
	}

	if reportIntervalEnv != "" {
		*reportInterval = utils.StrToInt(reportIntervalEnv, *reportInterval)
	}

	if pollIntervalEnv != "" {
		*pollInterval = utils.StrToInt(reportIntervalEnv, *pollInterval)
	}

	initLogger()

	log.WithFields(log.Fields{
		"address":        *address,
		"reportInterval": *reportInterval,
		"pollInterval":   *pollInterval,
	}).Infoln("starting goMetrics agent")

	store := agent.NewInMemoryMetricsStore()
	sender := agent.NewHTTPMetricsSender(fmt.Sprintf("http://%s", *address))

	service := services.NewCollectMetricsService(store, sender, time.Duration(*pollInterval)*time.Second, time.Duration(*reportInterval)*time.Second)
	service.Run()
}
