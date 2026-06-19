// Package prometheus 提供 Prometheus 监控的自动配置
package prometheus

import (
	"fmt"

	prometheuscore "github.com/xudefa/go-boot-prometheus"

	"github.com/xudefa/go-boot/boot"
	"github.com/xudefa/go-boot/condition"
	"github.com/xudefa/go-boot/constants"
	"github.com/xudefa/go-boot/core"
	"github.com/xudefa/go-boot/metrics"
)

// AutoConfiguration Prometheus 自动配置类
//
// 自动配置 Prometheus 导出器，使其能够将 go-boot 指标的收集数据转换为
// Prometheus 格式并通过 HTTP 端点暴露。
//
// 自动配置项（通过环境变量或配置文件设置）：
//   - prometheus.enabled: 是否启用 Prometheus 集成（默认 false）
//   - prometheus.endpoint: 指标暴露端点（默认 /metrics）
//   - prometheus.port: HTTP 服务器端口（默认 9090）
type AutoConfiguration struct{}

// Configure 执行 Prometheus 自动配置
//
// 从依赖注入容器获取 MeterRegistry，创建 Prometheus 导出器，
// 并将其注册到容器中供其他组件使用。
func (a *AutoConfiguration) Configure(ctx boot.ApplicationContext) error {
	// 获取 MeterRegistry
	obj, err := ctx.Get(constants.MeterRegistryBeanID)
	if err != nil {
		return err
	}

	meterRegistry, ok := obj.(metrics.MeterRegistry)
	if !ok {
		return fmt.Errorf("failed to cast object to metrics.MeterRegistry")
	}

	// 创建 Prometheus 导出器
	exporter := prometheuscore.NewExporter(meterRegistry)

	// 启动导出器
	if err := exporter.Start(); err != nil {
		return err
	}

	// 注册导出器到容器
	if err := ctx.Register(constants.PrometheusExporterBeanID,
		core.Bean(exporter),
		core.Singleton()); err != nil {
		return err
	}

	return nil
}

func init() {
	boot.RegisterAutoConfig(
		&AutoConfiguration{},
		condition.OnProperty(constants.PrometheusEnabled, constants.ConditionTrue),
	)
}
