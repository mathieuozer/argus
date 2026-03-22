package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Registry holds all registered metrics.
type Registry struct {
	mu       sync.RWMutex
	counters map[string]*Counter
	gauges   map[string]*Gauge
	histos   map[string]*Histogram
}

// NewRegistry creates a new metrics registry.
func NewRegistry() *Registry {
	return &Registry{
		counters: make(map[string]*Counter),
		gauges:   make(map[string]*Gauge),
		histos:   make(map[string]*Histogram),
	}
}

// Default is the global metrics registry.
var Default = NewRegistry()

// Counter is a monotonically increasing counter.
type Counter struct {
	name   string
	help   string
	values sync.Map // label key -> *int64
}

// NewCounter creates and registers a counter.
func (r *Registry) NewCounter(name, help string) *Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := &Counter{name: name, help: help}
	r.counters[name] = c
	return c
}

// Inc increments the counter with the given labels.
func (c *Counter) Inc(labels string) {
	v, _ := c.values.LoadOrStore(labels, new(int64))
	atomic.AddInt64(v.(*int64), 1)
}

// Add adds a value to the counter.
func (c *Counter) Add(labels string, delta int64) {
	v, _ := c.values.LoadOrStore(labels, new(int64))
	atomic.AddInt64(v.(*int64), delta)
}

// Gauge is a value that can go up and down.
type Gauge struct {
	name   string
	help   string
	values sync.Map // label key -> *int64
}

// NewGauge creates and registers a gauge.
func (r *Registry) NewGauge(name, help string) *Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	g := &Gauge{name: name, help: help}
	r.gauges[name] = g
	return g
}

// Set sets the gauge value.
func (g *Gauge) Set(labels string, value int64) {
	v, _ := g.values.LoadOrStore(labels, new(int64))
	atomic.StoreInt64(v.(*int64), value)
}

// Inc increments the gauge.
func (g *Gauge) Inc(labels string) {
	v, _ := g.values.LoadOrStore(labels, new(int64))
	atomic.AddInt64(v.(*int64), 1)
}

// Dec decrements the gauge.
func (g *Gauge) Dec(labels string) {
	v, _ := g.values.LoadOrStore(labels, new(int64))
	atomic.AddInt64(v.(*int64), -1)
}

// Histogram tracks the distribution of values using predefined buckets.
type Histogram struct {
	name    string
	help    string
	buckets []float64
	mu      sync.Mutex
	data    map[string]*histData
}

type histData struct {
	bucketCounts []int64
	sum          float64
	count        int64
}

// NewHistogram creates and registers a histogram.
func (r *Registry) NewHistogram(name, help string, buckets []float64) *Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	sort.Float64s(buckets)
	h := &Histogram{name: name, help: help, buckets: buckets, data: make(map[string]*histData)}
	r.histos[name] = h
	return h
}

// Observe records a value in the histogram.
func (h *Histogram) Observe(labels string, value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	d, ok := h.data[labels]
	if !ok {
		d = &histData{bucketCounts: make([]int64, len(h.buckets))}
		h.data[labels] = d
	}
	d.sum += value
	d.count++
	for i, b := range h.buckets {
		if value <= b {
			d.bucketCounts[i]++
			break
		}
	}
}

// Handler returns an HTTP handler that serves metrics in Prometheus text format.
func (r *Registry) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		var sb strings.Builder

		r.mu.RLock()
		defer r.mu.RUnlock()

		// Counters
		for _, c := range r.counters {
			sb.WriteString(fmt.Sprintf("# HELP %s %s\n", c.name, c.help))
			sb.WriteString(fmt.Sprintf("# TYPE %s counter\n", c.name))
			c.values.Range(func(key, value interface{}) bool {
				labels := key.(string)
				v := atomic.LoadInt64(value.(*int64))
				if labels == "" {
					sb.WriteString(fmt.Sprintf("%s %d\n", c.name, v))
				} else {
					sb.WriteString(fmt.Sprintf("%s{%s} %d\n", c.name, labels, v))
				}
				return true
			})
		}

		// Gauges
		for _, g := range r.gauges {
			sb.WriteString(fmt.Sprintf("# HELP %s %s\n", g.name, g.help))
			sb.WriteString(fmt.Sprintf("# TYPE %s gauge\n", g.name))
			g.values.Range(func(key, value interface{}) bool {
				labels := key.(string)
				v := atomic.LoadInt64(value.(*int64))
				if labels == "" {
					sb.WriteString(fmt.Sprintf("%s %d\n", g.name, v))
				} else {
					sb.WriteString(fmt.Sprintf("%s{%s} %d\n", g.name, labels, v))
				}
				return true
			})
		}

		// Histograms
		for _, h := range r.histos {
			sb.WriteString(fmt.Sprintf("# HELP %s %s\n", h.name, h.help))
			sb.WriteString(fmt.Sprintf("# TYPE %s histogram\n", h.name))
			h.mu.Lock()
			for labels, d := range h.data {
				var cumulative int64
				for i, b := range h.buckets {
					cumulative += d.bucketCounts[i]
					if labels == "" {
						sb.WriteString(fmt.Sprintf("%s_bucket{le=\"%.3f\"} %d\n", h.name, b, cumulative))
					} else {
						sb.WriteString(fmt.Sprintf("%s_bucket{%s,le=\"%.3f\"} %d\n", h.name, labels, b, cumulative))
					}
				}
				if labels == "" {
					sb.WriteString(fmt.Sprintf("%s_bucket{le=\"+Inf\"} %d\n", h.name, d.count))
					sb.WriteString(fmt.Sprintf("%s_sum %.6f\n", h.name, d.sum))
					sb.WriteString(fmt.Sprintf("%s_count %d\n", h.name, d.count))
				} else {
					sb.WriteString(fmt.Sprintf("%s_bucket{%s,le=\"+Inf\"} %d\n", h.name, labels, d.count))
					sb.WriteString(fmt.Sprintf("%s_sum{%s} %.6f\n", h.name, labels, d.sum))
					sb.WriteString(fmt.Sprintf("%s_count{%s} %d\n", h.name, labels, d.count))
				}
			}
			h.mu.Unlock()
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sb.String()))
	}
}

// HTTPLatencyBuckets are default latency buckets in seconds.
var HTTPLatencyBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0}

// HTTPMiddleware creates middleware that tracks request count, latency, and errors.
func HTTPMiddleware(reg *Registry, serviceName string) func(http.Handler) http.Handler {
	requestsTotal := reg.NewCounter("http_requests_total", "Total number of HTTP requests")
	requestDuration := reg.NewHistogram("http_request_duration_seconds", "HTTP request duration in seconds", HTTPLatencyBuckets)
	activeRequests := reg.NewGauge("http_active_requests", "Number of active HTTP requests")
	errorsTotal := reg.NewCounter("http_errors_total", "Total number of HTTP errors (4xx/5xx)")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metrics" || r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/health/") {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			labels := fmt.Sprintf(`service="%s",method="%s",path="%s"`, serviceName, r.Method, r.URL.Path)

			activeRequests.Inc(fmt.Sprintf(`service="%s"`, serviceName))
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			activeRequests.Dec(fmt.Sprintf(`service="%s"`, serviceName))

			duration := time.Since(start).Seconds()
			statusLabels := fmt.Sprintf(`%s,status="%d"`, labels, rw.statusCode)
			requestsTotal.Inc(statusLabels)
			requestDuration.Observe(labels, duration)

			if rw.statusCode >= 400 {
				errorsTotal.Inc(statusLabels)
			}
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
