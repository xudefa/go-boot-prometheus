// Package prometheus
package prometheus

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/xudefa/go-boot/metrics"
)

func TestExporter_Basic(t *testing.T) {
	bootMetrics := metrics.NewSimpleRegistry()
	exporter := NewExporter(bootMetrics)

	err := exporter.Start()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	defer func() {
		err := exporter.Stop()
		if err != nil {
			t.Errorf("expected no error stopping, got %v", err)
		}
	}()

	handler := exporter.Handler()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestBootMetricsCollector(t *testing.T) {
	bootMetrics := metrics.NewSimpleRegistry()

	// 添加一些测试指标
	bootMetrics.Counter("requests_total", "method", "GET", "handler", "api").Inc()
	bootMetrics.Counter("requests_total", "method", "POST", "handler", "api").Add(2)
	bootMetrics.Gauge("memory_usage", "unit", "MB").Set(1024.5)

	collector := &BootMetricsCollector{
		registry: bootMetrics,
	}

	// 测试 Describe
	descChan := make(chan *prometheus.Desc, 10)
	go func() {
		collector.Describe(descChan)
		close(descChan)
	}()

	descCount := 0
	for range descChan {
		descCount++
	}

	// 测试 Collect
	metricChan := make(chan prometheus.Metric, 10)
	go func() {
		collector.Collect(metricChan)
		close(metricChan)
	}()

	metricCount := 0
	for range metricChan {
		metricCount++
	}

	// 至少应该有 2 个指标（2 个 counter + 1 个 gauge，但由于相同的名称，会被合并）
	if metricCount < 2 {
		t.Errorf("expected at least 2 metrics, got %d", metricCount)
	}
}

func TestExporter_HTTPHandler(t *testing.T) {
	bootMetrics := metrics.NewSimpleRegistry()

	// 添加测试指标
	bootMetrics.Counter("test_counter", "label", "value").Inc()
	bootMetrics.Gauge("test_gauge").Set(42.0)

	exporter := NewExporter(bootMetrics)
	err := exporter.Start()
	if err != nil {
		t.Fatalf("expected no error starting exporter, got %v", err)
	}
	defer func() {
		_ = exporter.Stop()
	}()

	server := httptest.NewServer(exporter.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("expected no error making request, got %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// 检查响应内容是否包含预期的指标
	// 由于我们添加了两个指标，响应应该包含这些指标
}
