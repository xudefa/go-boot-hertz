# go-boot-hertz

[![Go Version](https://img.shields.io/github/go-mod/go-version/xudefa/go-boot-hertz)](https://go.dev/) [![License](https://img.shields.io/github/license/xudefa/go-boot-hertz)](./LICENSE) [![Build Status](https://img.shields.io/github/actions/workflow/status/xudefa/go-boot-hertz/test.yml?branch=master)](https://github.com/xudefa/go-boot-hertz/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/xudefa/go-boot-hertz.svg)](https://pkg.go.dev/github.com/xudefa/go-boot-hertz) [![Go Report Card](https://goreportcard.com/badge/github.com/xudefa/go-boot-hertz)](https://goreportcard.com/report/github.com/xudefa/go-boot-hertz)

基于 [go-boot](https://github.com/xudefa/go-boot) 的 hertz Web 框架集成模块。将 hertz 无缝集成到 go-boot 的 IoC 容器和自动配置体系中，提供声明式的路由注册、中间件配置和优雅启停能力。

> 设计理念：遵循 go-boot 的开发规范，将 hertz 作为 `net.Server` 接口的实现，通过自动配置实现零代码启动 Web 服务。

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
                    │     go-boot-hertz Starter     │
                    │  ┌─────────────────────────┐  │
                    │  │ hertzEngine Bean        │  │
                    │  │ hertzServer (net.Server)│  │
                    │  │ Router Configuration    │  │
                    │  │ Middleware Chain        │  │
                    │  └─────────────────────────┘  │
                    └───────────────────────────────┘
```

## 目录

- [快速开始](#快速开始)
- [功能特性](#功能特性)
- [路由注册](#路由注册)
- [中间件配置](#中间件配置)
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

# 安装 hertz 集成模块
go get github.com/xudefa/go-boot-hertz
```

### 最小示例

```go
package main

import (
    "context"

    "github.com/cloudwego/hertz/pkg/app"
    "github.com/cloudwego/hertz/pkg/app/server"
    "github.com/xudefa/go-boot/boot"
    "github.com/xudefa/go-boot/core"
)

func main() {
    app, err := boot.NewApplication(
        boot.WithAppName("my-web-app"),
        boot.WithVersion("1.0.0"),
    )
    if err != nil {
        panic(err)
    }
    defer app.Stop()

    // 注册 hertz Engine
    app.Container().Register("hertzEngine", core.Bean(server.Default()))

    // 注册路由
    engine := app.Container().Get("hertzEngine").(*server.Hertz)
    engine.GET("/hello", func(ctx context.Context, c *app.RequestContext) {
        c.JSON(200, map[string]interface{}{"message": "Hello from go-boot-hertz!"})
    })

    // 启动应用（自动启动 hertz 服务器）
    app.Start()

    // 等待终止信号
    app.WaitForSignal()
}
```

## 功能特性

| 特性 | 说明 |
|------|------|
| hertz 集成 | 将 hertz Engine 注册为 Bean，支持依赖注入 |
| net.Server 实现 | hertzServer 实现 go-boot 的 `net.Server` 接口 |
| 自动配置 | 通过 `hertz.enabled=true` 自动启动 Web 服务器 |
| 优雅启停 | 支持优雅关闭和生命周期管理 |
| 声明式路由 | 支持通过 Handler Bean 声明式注册路由 |
| 中间件链 | 支持全局和路由级中间件配置 |
| 配置驱动 | 端口、模式、超时等均可通过配置控制 |

## 路由注册

### 方式一：直接注册

```go
engine := app.Container().Get("hertzEngine").(*server.Hertz)
engine.GET("/users", listUsers)
engine.POST("/users", createUser)
engine.PUT("/users/:id", updateUser)
engine.DELETE("/users/:id", deleteUser)
```

### 方式二：Handler Bean

```go
type UserHandler struct {
    Engine  *server.Hertz   `inject:"hertzEngine"`
    Service *UserService    `inject:"userService"`
}

func (h *UserHandler) RegisterRoutes() {
    group := h.Engine.Group("/users")
    group.GET("", h.List)
    group.POST("", h.Create)
    group.GET("/:id", h.GetByID)
    group.PUT("/:id", h.Update)
    group.DELETE("/:id", h.Delete)
}
```

### 方式三：Router 辅助

```go
import "github.com/xudefa/go-boot-hertz/router"

r := router.New(app.Container())
r.GET("/health", healthCheck)
r.Group("/api/v1", func(g *router.Group) {
    g.GET("/users", listUsers)
    g.POST("/users", createUser)
})
```

## 中间件配置

### 全局中间件

```go
engine := app.Container().Get("hertzEngine").(*server.Hertz)

// 内置中间件
engine.Use(recovery.Recovery())
engine.Use(logger.Logger())

// 自定义中间件
engine.Use(func(ctx context.Context, c *app.RequestContext) {
    start := time.Now()
    c.Next(ctx)
    log.Printf("Request took %v", time.Since(start))
})
```

### CORS 中间件

```go
import "github.com/xudefa/go-boot-hertz/middleware"

engine.Use(middleware.CORS(middleware.CORSConfig{
    AllowOrigins:     []string{"*"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
}))
```

## 配置选项

通过 `boot.WithProperty()` 或配置文件设置：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `hertz.enabled` | `false` | 是否启用 hertz 服务器 |
| `hertz.port` | `8080` | 服务器监听端口 |
| `hertz.host` | `localhost` | 服务器监听地址 |
| `hertz.mode` | `debug` | hertz 模式：debug / release / test |
| `hertz.read-timeout` | `10` | 读取超时（秒） |
| `hertz.write-timeout` | `10` | 写入超时（秒） |
| `hertz.idle-timeout` | `60` | 空闲超时（秒） |
| `hertz.shutdown-timeout` | `5` | 优雅关闭超时（秒） |

### 示例配置

```yaml
# application.yml
hertz:
  enabled: true
  port: 8080
  mode: debug
  read-timeout: 30
  write-timeout: 30
  idle-timeout: 60
  shutdown-timeout: 10
```

## 项目结构

```
go-boot-hertz/
├── hertz/                    # hertz 自动配置
│   └── autoconfig.go       # 自动配置注册
├── server/                 # hertz Server 实现
│   ├── server.go           # hertzServer 实现 net.Server
│   ├── options.go          # Server 选项配置
│   └── server_test.go      # 单元测试
├── router/                 # 路由注册辅助
│   └── router.go           # 声明式路由注册
├── middleware/             # 中间件
│   ├── security.go         # 安全中间件
│   ├── tracing.go          # 分布式追踪中间件
│   ├── validation.go       # 请求验证中间件
│   └── websocket.go        # WebSocket 适配器
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