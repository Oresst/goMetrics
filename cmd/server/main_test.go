package main

import (
	"fmt"
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

	storage := getStorage()
	service := newMetricsService(storage)
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
			storage := getStorage()
			service := newMetricsService(storage)
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
