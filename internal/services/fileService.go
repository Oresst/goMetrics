package services

import (
	"bufio"
	"encoding/json"
	"github.com/Oresst/goMetrics/models"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

type FileService struct {
	file     *os.File
	interval time.Duration
	buffer   []models.Metrics
	mode     string
	stopChan chan bool
}

func NewFileService(filename string, duration time.Duration) (*FileService, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	var mode string
	if duration == time.Duration(0) {
		mode = "sync"
	} else {
		mode = "async"
	}

	return &FileService{
		file:     file,
		interval: duration,
		mode:     mode,
		buffer:   make([]models.Metrics, 0),
		stopChan: make(chan bool),
	}, nil
}

func (f *FileService) Run() {
	if f.mode == "async" {
		go f.writeAsync()
	}
}

func (f *FileService) writeToFile(metric models.Metrics) {
	place := "[FileService.writeToFile]"

	data, err := json.Marshal(metric)
	if err != nil {
		log.WithFields(log.Fields{
			"place": place,
			"err":   err.Error(),
		}).Error("Error marshalling metric")
		return
	}

	data = append(data, '\n')
	_, err = f.file.Write(data)
	if err != nil {
		log.WithFields(log.Fields{
			"place": place,
			"err":   err.Error(),
		}).Error("Error writing to file")
		return
	}
}

func (f *FileService) writeAsync() {
	for {
		f.flush()

		select {
		case <-f.stopChan:
			return
		case <-time.After(f.interval):
			continue
		}
	}
}

func (f *FileService) flush() {
	copied := make([]models.Metrics, len(f.buffer))
	copy(copied, f.buffer)
	f.buffer = make([]models.Metrics, 0)

	for _, metric := range copied {
		f.writeToFile(metric)
	}
}

func (f *FileService) Write(metric models.Metrics) {
	if f.mode == "sync" {
		f.writeToFile(metric)
	} else {
		f.buffer = append(f.buffer, metric)
	}
}

func (f *FileService) Stop() error {
	if f.mode == "async" {
		f.stopChan <- true
		f.flush()
	}

	return f.file.Close()
}

func (f *FileService) ReadAllData(fileName string) ([]models.Metrics, error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var result []models.Metrics

	for scanner.Scan() {
		data := scanner.Bytes()
		metric := models.Metrics{}
		err = json.Unmarshal(data, &metric)
		if err != nil {
			return nil, err
		}

		result = append(result, metric)
	}

	return result, nil
}
