package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Definimos os "marcadores" que vão aparecer no Grafana
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total de requisições HTTP processadas",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Tempo de resposta das requisições HTTP",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)
)

// responseWriterInterceptor é um truque do Go para conseguirmos ler o Status Code da resposta
type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterInterceptor) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// MetricsMiddleware engloba cada requisição para medir o tempo e o resultado
func (s *Server) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Intercepta a resposta para capturar o código HTTP
		wi := &responseWriterInterceptor{w, http.StatusOK}

		// Roda a requisição de verdade
		next.ServeHTTP(wi, r)

		// Calcula a duração
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wi.statusCode)
		route := r.URL.Path

		// Só logamos a métrica se não for a própria rota do Prometheus (para não poluir)
		if route != "/metrics" {
			httpRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
			httpRequestDuration.WithLabelValues(r.Method, route, status).Observe(duration)
		}
	})
}
