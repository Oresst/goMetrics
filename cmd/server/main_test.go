package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/Oresst/goMetrics/internal/services"
	"github.com/Oresst/goMetrics/internal/utils"
	"github.com/Oresst/goMetrics/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestAddMetricHandler(t *testing.T) {
	type waiting struct {
		code        int
		contentType string
	}

	testCases := []struct {
		testName string
		url      string
		method   string
		waiting  waiting
	}{
		{
			testName: "valid type counter",
			url:      "/update/counter/someMetric/527",
			method:   http.MethodPost,
			waiting: waiting{
				code:        200,
				contentType: "text/plain",
			},
		},
		{
			testName: "valid type gauge",
			url:      "/update/gauge/someMetric/527",
			method:   http.MethodPost,
			waiting: waiting{
				code:        200,
				contentType: "text/plain",
			},
		},
		{
			testName: "wrong type",
			url:      "/update/someType/someMetric/527",
			method:   http.MethodPost,
			waiting: waiting{
				code:        400,
				contentType: "text/plain",
			},
		},
		{
			testName: "wrong method (GET)",
			url:      "/update/gauge/someMetric/527",
			method:   http.MethodGet,
			waiting: waiting{
				code:        405,
				contentType: "text/plain",
			},
		},
		{
			testName: "empty name",
			url:      "/update/gauge//527",
			method:   http.MethodPost,
			waiting: waiting{
				code:        400,
				contentType: "text/plain",
			},
		},
		{
			testName: "wrong value",
			url:      "/update/gauge/someMetric/asdasd",
			method:   http.MethodPost,
			waiting: waiting{
				code:        400,
				contentType: "text/plain",
			},
		},
	}

	storage := getStorageMem()
	service := services.NewMetricsService(storage, nil)
	r := getRouter(service)

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, request)

			result := w.Result()
			defer result.Body.Close()

			require.Equal(t, tc.waiting.code, result.StatusCode)
			require.Equal(t, tc.waiting.contentType, result.Header.Get("Content-Type"))
		})
	}
}

func TestGetMetricHandler(t *testing.T) {
	type waiting struct {
		code        int
		contentType string
		value       float64
	}

	type testData struct {
		metricName  string
		metricType  string
		metricValue float64
	}

	testCases := []struct {
		testName   string
		method     string
		metricType string
		metricName string
		waiting    waiting
		testData   []testData
	}{
		{
			testName:   "valid type counter",
			method:     http.MethodGet,
			metricType: "counter",
			metricName: "someMetric",
			testData: []testData{
				{
					metricName:  "someMetric",
					metricType:  "counter",
					metricValue: 1,
				},
			},
			waiting: waiting{
				code:        200,
				contentType: "text/plain",
				value:       1,
			},
		},
		{
			testName:   "valid type gauge",
			method:     http.MethodGet,
			metricType: "gauge",
			metricName: "someMetric2",
			testData: []testData{
				{
					metricName:  "someMetric2",
					metricType:  "gauge",
					metricValue: 1.213123,
				},
			},
			waiting: waiting{
				code:        200,
				contentType: "text/plain",
				value:       1.213123,
			},
		},
		{
			testName:   "wrong type",
			method:     http.MethodGet,
			metricType: "someType",
			metricName: "someMetric3",
			testData: []testData{
				{
					metricName:  "someMetric2",
					metricType:  "gauge",
					metricValue: 1.213123,
				},
			},
			waiting: waiting{
				code:        400,
				contentType: "text/plain",
			},
		},
		{
			testName:   "2 counter calls",
			method:     http.MethodGet,
			metricType: "counter",
			metricName: "someMetric",
			testData: []testData{
				{
					metricName:  "someMetric",
					metricType:  "counter",
					metricValue: 1,
				},
				{
					metricName:  "someMetric",
					metricType:  "counter",
					metricValue: 2,
				},
			},
			waiting: waiting{
				code:        200,
				contentType: "text/plain",
				value:       3,
			},
		},
		{
			testName:   "test update value",
			method:     http.MethodGet,
			metricType: "gauge",
			metricName: "someMetric",
			testData: []testData{
				{
					metricName:  "someMetric",
					metricType:  "gauge",
					metricValue: 1.213123,
				},
				{
					metricName:  "someMetric",
					metricType:  "gauge",
					metricValue: 5.213123,
				},
			},
			waiting: waiting{
				code:        200,
				contentType: "text/plain",
				value:       5.213123,
			},
		},
		{
			testName:   "empty type",
			method:     http.MethodGet,
			metricType: "",
			metricName: "someMetric",
			testData:   []testData{},
			waiting: waiting{
				code:        400,
				contentType: "text/plain",
			},
		},
		{
			testName:   "not found",
			method:     http.MethodGet,
			metricType: "gauge",
			metricName: "someMetric",
			testData:   []testData{},
			waiting: waiting{
				code:        404,
				contentType: "text/plain",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			storage := getStorageMem()
			service := services.NewMetricsService(storage, nil)
			r := getRouter(service)

			for _, data := range tc.testData {
				err := storage.AddMetric(data.metricType, data.metricName, data.metricValue)

				require.NoError(t, err)
			}

			url := fmt.Sprintf("/value/%s/%s", tc.metricType, tc.metricName)
			request := httptest.NewRequest(tc.method, url, nil)
			response := httptest.NewRecorder()

			r.ServeHTTP(response, request)
			result := response.Result()
			defer result.Body.Close()

			require.Equal(t, tc.waiting.code, result.StatusCode)
			require.Equal(t, tc.waiting.contentType, result.Header.Get("Content-Type"))

			if result.StatusCode != 200 {
				return
			}

			body, err := io.ReadAll(result.Body)

			require.NoError(t, err)

			value, err := strconv.ParseFloat(string(body), 64)

			require.NoError(t, err)
			assert.Equal(t, tc.waiting.value, value)
		})
	}
}

func TestAddMetricJSONHandler(t *testing.T) {
	type waiting struct {
		code        int
		contentType string
	}

	testCases := []struct {
		testName    string
		method      string
		url         string
		contentType string
		testData    models.Metrics
		waiting     waiting
	}{
		{
			testName:    "valid metric gauge type",
			method:      http.MethodPost,
			url:         "/update",
			contentType: "application/json",
			testData: models.Metrics{
				ID:    "test",
				MType: models.Gauge,
				Value: utils.PointFloat64(10.1235),
			},
			waiting: waiting{
				code: 200,
			},
		},
		{
			testName:    "valid metric Counter type",
			method:      http.MethodPost,
			url:         "/update",
			contentType: "application/json",
			testData: models.Metrics{
				ID:    "test",
				MType: models.Counter,
				Delta: utils.PointInt64(10),
			},
			waiting: waiting{
				code: 200,
			},
		},
		{
			testName:    "invalid metric type",
			method:      http.MethodPost,
			url:         "/update",
			contentType: "application/json",
			testData: models.Metrics{
				ID:    "test",
				MType: "asd",
				Value: utils.PointFloat64(10.123534),
			},
			waiting: waiting{
				code: 400,
			},
		},
		{
			testName:    "wrong content type",
			method:      http.MethodPost,
			url:         "/update",
			contentType: "",
			testData:    models.Metrics{},
			waiting: waiting{
				code: 400,
			},
		},
		{
			testName:    "empty test data",
			method:      http.MethodPost,
			url:         "/update",
			contentType: "",
			testData:    models.Metrics{},
			waiting: waiting{
				code: 400,
			},
		},
		{
			testName:    "without type",
			method:      http.MethodPost,
			url:         "/update",
			contentType: "",
			testData: models.Metrics{
				ID: "test",
			},
			waiting: waiting{
				code: 400,
			},
		},
		{
			testName:    "without value for gauge type",
			method:      http.MethodPost,
			url:         "/update",
			contentType: "",
			testData: models.Metrics{
				ID:    "test",
				MType: models.Gauge,
				Delta: utils.PointInt64(10),
			},
			waiting: waiting{
				code: 400,
			},
		},
		{
			testName:    "without delta for counter type",
			method:      http.MethodPost,
			url:         "/update",
			contentType: "",
			testData: models.Metrics{
				ID:    "test",
				MType: models.Counter,
				Value: utils.PointFloat64(10),
			},
			waiting: waiting{
				code: 400,
			},
		},
		{
			testName:    "wrong method",
			method:      http.MethodGet,
			url:         "/update",
			contentType: "",
			testData:    models.Metrics{},
			waiting: waiting{
				code: 405,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			storage := getStorageMem()
			service := services.NewMetricsService(storage, nil)
			r := getRouter(service)

			rawData, _ := json.Marshal(tc.testData)
			buffered := bytes.NewBuffer(rawData)

			request := httptest.NewRequest(tc.method, tc.url, buffered)
			request.Header.Set("Content-Type", tc.contentType)
			response := httptest.NewRecorder()
			r.ServeHTTP(response, request)

			result := response.Result()
			defer result.Body.Close()

			assert.Equal(t, tc.waiting.code, result.StatusCode)
		})
	}
}

func TestGzipCompression(t *testing.T) {
	storage := getStorageMem()
	service := services.NewMetricsService(storage, nil)
	r := getRouter(service)

	t.Run("send gziped request", func(t *testing.T) {
		requestData := struct {
			Id    string  `json:"id"`
			Type  string  `json:"type"`
			Delta float64 `json:"delta"`
		}{
			Id:    "test",
			Type:  "counter",
			Delta: 100,
		}

		waitingData := struct {
			statusCode int
		}{
			statusCode: 200,
		}

		rawData, err := json.Marshal(requestData)
		require.NoError(t, err)

		buffered := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buffered)
		_, err = zb.Write(rawData)
		require.NoError(t, err)

		err = zb.Close()
		require.NoError(t, err)

		request := httptest.NewRequest(http.MethodPost, "/update", buffered)
		request.Header.Set("Content-Encoding", "gzip")
		request.Header.Set("content-type", "application/json")
		request.Header.Set("Accept-Encoding", "")

		writer := httptest.NewRecorder()
		r.ServeHTTP(writer, request)

		result := writer.Result()
		defer result.Body.Close()

		assert.Equal(t, waitingData.statusCode, result.StatusCode)
	})

	t.Run("accept gziped response", func(t *testing.T) {
		requestData := struct {
			Id   string `json:"id"`
			Type string `json:"type"`
		}{
			Id:   "test2",
			Type: "counter",
		}

		waitingData := struct {
			statusCode int
			value      int64
		}{
			statusCode: 200,
			value:      200,
		}

		responseData := struct {
			Id    string `json:"id"`
			Type  string `json:"type"`
			Delta int64  `json:"delta"`
		}{}

		err := storage.AddMetric(requestData.Type, requestData.Id, float64(waitingData.value))
		require.NoError(t, err)

		rawData, err := json.Marshal(requestData)
		require.NoError(t, err)

		buffered := bytes.NewBuffer(rawData)
		request := httptest.NewRequest(http.MethodPost, "/value", buffered)
		request.Header.Set("Accept-Encoding", "gzip")
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Content-Encoding", "")

		writer := httptest.NewRecorder()
		r.ServeHTTP(writer, request)

		result := writer.Result()
		defer result.Body.Close()

		assert.Equal(t, waitingData.statusCode, result.StatusCode)

		gr, err := gzip.NewReader(result.Body)
		require.NoError(t, err)

		rawResponseData, err := io.ReadAll(gr)
		require.NoError(t, err)

		err = json.Unmarshal(rawResponseData, &responseData)
		require.NoError(t, err)

		assert.Equal(t, waitingData.value, responseData.Delta)
		assert.Equal(t, requestData.Id, responseData.Id)
		assert.Equal(t, requestData.Type, responseData.Type)
	})
}
