package services

import (
	"compress/gzip"
	"net/http"
)

type gzipWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func newGzipWriter(w http.ResponseWriter) *gzipWriter {
	return &gzipWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (gw *gzipWriter) Header() http.Header {
	return gw.w.Header()
}

func (gw *gzipWriter) Write(b []byte) (int, error) {
	return gw.zw.Write(b)
}

func (gw *gzipWriter) WriteHeader(statusCode int) {
	gw.w.Header().Set("Content-Encoding", "gzip")
	gw.w.WriteHeader(statusCode)
}

func (gw *gzipWriter) Close() error {
	return gw.zw.Close()
}
