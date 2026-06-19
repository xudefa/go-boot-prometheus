# go-boot-prometheus

[![Go Version](https://img.shields.io/github/go-mod/go-version/xudefa/go-boot-prometheus)](https://go.dev/) [![License](https://img.shields.io/github/license/xudefa/go-boot-prometheus)](./LICENSE) [![Build Status](https://img.shields.io/github/actions/workflow/status/xudefa/go-boot-prometheus/test.yml?branch=master)](https://github.com/xudefa/go-boot-prometheus/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/xudefa/go-boot-prometheus.svg)](https://pkg.go.dev/github.com/xudefa/go-boot-prometheus) [![Go Report Card](https://goreportcard.com/badge/github.com/xudefa/go-boot-prometheus)](https://goreportcard.com/report/github.com/xudefa/go-boot-prometheus)

基于 [go-boot](https://github.com/xudefa/go-boot) 的 Prometheus 监控集成模块。将 go-boot 指标系统无缝对接到 Prometheus 生态，提供自动化的指标转换、HTTP 端点暴露和优雅启停能力。

> 设计理念：遵循 go-boot 的开发规范，将 Prometheus Exporter 实现为可独立运行的组件，通过自动配置实现零代码启动指标暴露服务。

## 整体架构

```
┌───────────────────────────────────────────────────────────────────────┐
│                    go-boot ApplicationContext                         │
│  ┌───────────┐ ┌──────────────┐ ┌───────────┐ ┌───────────┐           │
│  │ Container │ │  Environment │ │ Lifecycle │ │ EventBus  │           │
│  └───────────┘ └──────────────┘ └───────────┘ └───────────┘           │
│                       ┌─────────────────────┐                         │
│                       │ AutoConfig Registry │                         │
│                       └─────────────────────┘                         │
└───────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                    ┌───────────────────────────────┐
                    │  go-boot-prometheus Starter   │
                    │  ┌─────────────────────────┐  │
                    │  │ Exporter Bean           │  │
                    │  │ BootMetricsCollector    │  │
                    │  │ HTTP Handler (/metrics) │  │
                    │  │ Prometheus Registry     │  │
                    │  └─────────────────────────┘  │
                    └───────────────────────────────┘
```

## 目录

- [快速开始](#快速开始)
- [功能特性](#功能特性)
- [指标转换](#指标转换)
- [HTTP 端点](#http-端点)
- [配置选项](#配置选项)
- [项目结构](#项目结构)
- [开发指南](#开发指南)
- [贡献](#贡献)
- [许可证](#许可证)

## 快速开始

### 安装

```bash
# 安装核心框架
go get github.com/xudefa/go-boot

# 安装 Prometheus 集成模块
go get github.com/xudefa/go-boot-prometheus
```

### 最小示例

```go
package main

import (
    "net/http"

    "github.com/xudefa/go-boot/boot"
    "github.com/xudefa/go-boot/metrics"
    prometheus "github.com/xudefa/go-boot-prometheus"
)

func main() {
    app, err := boot.NewApplication(
        boot.WithAppName("my-metrics-app"),
        boot.WithVersion("1.0.0"),
    )
    if err != nil {
        panic(err)
    }
    defer app.Stop()

    // 获取 Prometheus Exporter（由自动配置注册）
    exporter := app.Container().Get("prometheusExporter").(*prometheus.Exporter)

    // 暴露 /metrics 端点
    http.Handle("/metrics", exporter.Handler())
    http.ListenAndServe(":9090", nil)

    // 等待终止信号
    app.WaitForSignal()
}
```

## 功能特性

| 特性 | 说明 |
|------|------|
| 指标转换 | 将 go-boot 的 Counter、Gauge、Histogram 转换为 Prometheus 格式 |
| 自动配置 | 通过 `prometheus.enabled=true` 自动启用 |
| HTTP 端点 | 内置 `/metrics` HTTP Handler 供 Prometheus 抓取 |
| OpenMetrics | 支持 OpenMetrics 格式，兼容 Prometheus 2.x |
| 依赖注入 | Exporter 自动注册为 Bean |
| 优雅启停 | 支持优雅关闭和生命周期管理 |
| 并发安全 | 读写锁保护并发访问 |
| 自定义收集器 | 支持自定义 BootMetricsCollector 实现 |

## 指标转换

### Counter 指标

```go
// go-boot 指标注册
registry := app.Container().Get("meterRegistry").(metrics.MeterRegistry)
counter := registry.Counter("http_requests_total", map[string]string{
    "method": "GET",
    "path":   "/api/users",
})
counter.Inc()

// Prometheus 自动转换并暴露
// http_requests_total{method="GET",path="/api/users"} 1.0
```

### Gauge 指标

```go
// go-boot 指标注册
gauge := registry.Gauge("active_connections", map[string]string{
    "service": "user-service",
})
gauge.Set(42.0)

// Prometheus 自动转换并暴露
// active_connections{service="user-service"} 42.0
```

### Histogram 指标

```go
// go-boot 指标注册
histogram := registry.Histogram("request_duration_seconds", map[string]string{
    "handler": "GetUser",
})
histogram.Observe(0.125)

// Prometheus 自动转换并暴露
// request_duration_seconds{handler="GetUser"} 0.125
```

## HTTP 端点

### 基本用法

```go
exporter := app.Container().Get("prometheusExporter").(*prometheus.Exporter)

// 创建 HTTP 服务器
mux := http.NewServeMux()
mux.Handle("/metrics", exporter.Handler())

// 启动服务器
http.ListenAndServe(":9090", mux)
```

### 自定义路径

```go
mux := http.NewServeMux()
mux.Handle("/api/metrics", exporter.Handler())
http.ListenAndServe(":9090", mux)
```

### 与其他端点集成

```go
mux := http.NewServeMux()

// Prometheus 指标
mux.Handle("/metrics", exporter.Handler())

// 健康检查
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})

// 启动服务器
http.ListenAndServe(":9090", mux)
```

## 配置选项

通过 `boot.WithProperty()` 或配置文件设置：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `prometheus.enabled` | `false` | 是否启用 Prometheus 集成 |
| `prometheus.endpoint` | `/metrics` | 指标暴露端点路径 |
| `prometheus.port` | `9090` | HTTP 服务器监听端口 |

### 示例配置

```yaml
# application.yml
prometheus:
  enabled: true
  endpoint: /metrics
  port: 9090
```

## 项目结构

```
go-boot-prometheus/
├── autoconfig.go           # 自动配置注册
├── exporter.go             # Prometheus Exporter 实现
├── exporter_test.go        # 单元测试
├── README.md
├── LICENSE
└── go.mod
```

## 开发指南

### 构建

```bash
go build ./...
```

### 测试

```bash
go test ./...
go test -cover ./...       # 带覆盖率
go test -race ./...        # 数据竞争检测
```

### 代码规范

```bash
go fmt ./...
golangci-lint run
```

## 贡献

欢迎提交 Issue 和 Pull Request！详细贡献指南请参阅 [CONTRIBUTING.md](./CONTRIBUTING.md)。

## 许可证

本项目采用 MIT 许可证 — 详情请参阅 [LICENSE](./LICENSE) 文件。