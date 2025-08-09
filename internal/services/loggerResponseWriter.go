package services

import "net/http"

type responseData struct {
	size       int
	statusCode int
}

type loggerResponseWriter struct {
	http.ResponseWriter
	data *responseData
}

func (l *loggerResponseWriter) WriteHeader(statusCode int) {
	l.data.statusCode = statusCode
	l.ResponseWriter.WriteHeader(statusCode)
}

func (l *loggerResponseWriter) Write(data []byte) (int, error) {
	var err error
	l.data.size, err = l.ResponseWriter.Write(data)
	return l.data.size, err
}
