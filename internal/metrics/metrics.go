package metrics

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	},
	[]string{"method", "path", "status"},
)

var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "How long HTTP requests take",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "path"},
)

var WebsocketConnections = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "websocket_connections",
		Help: "Number of websocket connections open right now",
	},
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(WebsocketConnections)
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		if strings.HasPrefix(path, "/ws") || strings.HasSuffix(path, "/ws") {
			c.Next()
			return
		}

		start := time.Now()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		timeTaken := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(timeTaken)
	}
}
