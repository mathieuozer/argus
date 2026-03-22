package metrics

import (
	"fmt"
	"net/http/httptest"
	"testing"
)

func BenchmarkCounter_Inc(b *testing.B) {
	reg := NewRegistry()
	c := reg.NewCounter("bench_counter", "Benchmark counter")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc("")
	}
}

func BenchmarkCounter_Inc_WithLabels(b *testing.B) {
	reg := NewRegistry()
	c := reg.NewCounter("bench_counter_labels", "Benchmark counter with labels")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc(`method="GET",path="/api/v1/agents"`)
	}
}

func BenchmarkCounter_Inc_Parallel(b *testing.B) {
	reg := NewRegistry()
	c := reg.NewCounter("bench_counter_parallel", "Benchmark counter parallel")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc(`method="GET"`)
		}
	})
}

func BenchmarkGauge_Set(b *testing.B) {
	reg := NewRegistry()
	g := reg.NewGauge("bench_gauge", "Benchmark gauge")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Set("", int64(i))
	}
}

func BenchmarkGauge_IncDec_Parallel(b *testing.B) {
	reg := NewRegistry()
	g := reg.NewGauge("bench_gauge_parallel", "Benchmark gauge parallel")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.Inc("")
			g.Dec("")
		}
	})
}

func BenchmarkHistogram_Observe(b *testing.B) {
	reg := NewRegistry()
	h := reg.NewHistogram("bench_histogram", "Benchmark histogram", HTTPLatencyBuckets)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Observe("", float64(i%1000)/1000.0)
	}
}

func BenchmarkHistogram_Observe_Parallel(b *testing.B) {
	reg := NewRegistry()
	h := reg.NewHistogram("bench_histogram_parallel", "Benchmark histogram parallel", HTTPLatencyBuckets)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			h.Observe("", float64(i%1000)/1000.0)
			i++
		}
	})
}

func BenchmarkRegistry_Handler(b *testing.B) {
	reg := NewRegistry()
	c := reg.NewCounter("bench_requests_total", "Total requests")
	g := reg.NewGauge("bench_active", "Active")
	h := reg.NewHistogram("bench_duration", "Duration", HTTPLatencyBuckets)

	// Populate with realistic data
	for i := 0; i < 10; i++ {
		labels := fmt.Sprintf(`service="svc-%d",method="GET"`, i)
		c.Add(labels, int64(i*100))
		g.Set(labels, int64(i))
		h.Observe(labels, float64(i)/10.0)
	}

	handler := reg.Handler()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/metrics", nil)
		handler(w, req)
	}
}
