// Package prometheus 提供与 Prometheus 监控系统的集成
package prometheus

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xudefa/go-boot/metrics"
)

// Exporter Prometheus 导出器
//
// 将 go-boot 的指标数据转换为 Prometheus 格式并暴露给 Prometheus 服务器抓取。
// 支持 Counter、Gauge 等指标类型的转换。
type Exporter struct {
	registry   *prometheus.Registry  // Prometheus 注册表
	metrics    metrics.MeterRegistry // go-boot 指标注册表
	mu         sync.RWMutex          // 读写锁保护并发访问
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

// NewExporter 创建新的 Prometheus 导出器
func NewExporter(bootMetrics metrics.MeterRegistry) *Exporter {
	return &Exporter{
		registry:   prometheus.NewRegistry(),
		metrics:    bootMetrics,
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}
}

// Start 启动 Prometheus 导出器
//
// 将指标收集器注册到 Prometheus 注册表，并开始收集指标数据。
func (e *Exporter) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 注册自定义收集器
	collector := &BootMetricsCollector{
		registry: e.metrics,
	}

	if err := e.registry.Register(collector); err != nil {
		return fmt.Errorf("failed to register metrics collector: %w", err)
	}

	return nil
}

// Stop 停止 Prometheus 导出器
func (e *Exporter) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 注销所有收集器
	collector := &BootMetricsCollector{
		registry: e.metrics,
	}
	e.registry.Unregister(collector)

	return nil
}

// Handler 返回 Prometheus HTTP 处理器
//
// 用于将指标数据通过 HTTP 端点暴露给 Prometheus 服务器抓取。
func (e *Exporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// BootMetricsCollector go-boot 指标收集器
//
// 实现 Prometheus Collector 接口，将 go-boot 指标转换为 Prometheus 指标。
type BootMetricsCollector struct {
	registry metrics.MeterRegistry
}

// Describe 实现 Collector 接口
func (c *BootMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	// 发送描述符
}

// Collect 实现 Collector 接口
func (c *BootMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	metrics := c.registry.Collect()

	for _, m := range metrics {
		var metric prometheus.Metric
		var err error

		// 获取标签
		labelNames, labelValues := extractLabels(m.Tags)

		// 根据指标类型创建相应的 Prometheus 指标
		switch m.Type {
		case "counter":
			desc := prometheus.NewDesc(
				m.Name,
				"Counter metric from go-boot",
				labelNames,
				nil,
			)
			metric, err = prometheus.NewConstMetric(desc, prometheus.CounterValue, m.Value, labelValues...)

		case "gauge":
			desc := prometheus.NewDesc(
				m.Name,
				"Gauge metric from go-boot",
				labelNames,
				nil,
			)
			metric, err = prometheus.NewConstMetric(desc, prometheus.GaugeValue, m.Value, labelValues...)

		case "histogram":
			// 对于直方图，我们将其作为摘要指标处理
			desc := prometheus.NewDesc(
				m.Name,
				"Histogram metric from go-boot",
				labelNames,
				nil,
			)
			metric, err = prometheus.NewConstMetric(desc, prometheus.GaugeValue, m.Value, labelValues...)
		}

		if err != nil {
			continue
		}

		ch <- metric
	}
}

// extractLabels 从标签映射中提取标签名称和值
func extractLabels(tags map[string]string) ([]string, []string) {
	if len(tags) == 0 {
		return nil, nil
	}

	labelNames := make([]string, 0, len(tags))
	labelValues := make([]string, 0, len(tags))

	for k, v := range tags {
		labelNames = append(labelNames, k)
		labelValues = append(labelValues, v)
	}

	return labelNames, labelValues
}

// GetMetricsRegistry 获取关联的指标注册表
func (e *Exporter) GetMetricsRegistry() metrics.MeterRegistry {
	return e.metrics
}
