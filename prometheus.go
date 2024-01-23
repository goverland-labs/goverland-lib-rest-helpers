package resthelpers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_request_inflight",
		Help: "How many HTTP requests are currently being processed",
	}, []string{"endpoint", "method", "status"})

	httpRequestDurHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "The latency of the HTTP requests.",
		Buckets: prometheus.ExponentialBuckets(0.05, 2, 16), // start at 50ms, double buckets 16 times (so the last bucket is 1.6s)
	}, []string{"endpoint", "method"})

	httpRequestSizeHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_size_bytes",
		Help:    "The size of the HTTP requests.",
		Buckets: prometheus.ExponentialBuckets(100, 3, 16),
	}, []string{"endpoint", "method"})
)

func Prometheus(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rwInterceptor := &promResponseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(rwInterceptor, r)

		path := r.URL.Path
		requestsCounter.WithLabelValues(r.Method, path, strconv.Itoa(rwInterceptor.statusCode)).Inc()

		httpRequestDurHistogram.WithLabelValues(path, r.Method).Observe(time.Since(start).Seconds())
		httpRequestSizeHistogram.WithLabelValues(path, r.Method).Observe(float64(rwInterceptor.bytesWritten))
	})
}

type promResponseWriterInterceptor struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *promResponseWriterInterceptor) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *promResponseWriterInterceptor) Write(b []byte) (int, error) {
	w.bytesWritten += len(b)
	return w.ResponseWriter.Write(b)
}
