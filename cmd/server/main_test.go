package main

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
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
			testName: "empty value",
			url:      "/update/gauge/someMetric//",
			method:   http.MethodPost,
			waiting: waiting{
				code:        400,
				contentType: "text/plain",
			},
		},
		{
			testName: "wrong value",
			url:      "/update/gauge/someMetric/asdasd/",
			method:   http.MethodPost,
			waiting: waiting{
				code:        400,
				contentType: "text/plain",
			},
		},
	}

	storage := getStorage()
	service := newMetricsService(storage)

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.url, nil)
			w := httptest.NewRecorder()

			service.addMetricHandler(w, request)

			result := w.Result()

			require.Equal(t, tc.waiting.code, result.StatusCode)
			require.Equal(t, tc.waiting.contentType, result.Header.Get("Content-Type"))
		})
	}
}
