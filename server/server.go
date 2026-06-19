// Package server 基于 Hertz 框架提供 HTTP 服务器实现。
//
// 该包将 Hertz 框架与 go-boot 容器系统集成，
// 支持依赖注入、中间件和路由注册。
//
// 定义：
//
//   - HertzServer: HTTP 服务器实现了 net.Server 接口
//   - HandlerFunc: 请求处理函数类型
//   - ServerOption: 服务器配置选项
//
// 快速开始:
//
//	s := server.NewServer()
//	s.GET("/hello", func(ctx context.Context, c *app.RequestContext) {
//	    c.String(200, "Hello World")
//	})
//	s.Start()
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xudefa/go-boot/core"
	"github.com/xudefa/go-boot/net"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/route"
)

// HertzServer 是 Hertz HTTP 服务器。
//
// 字段说明:
//   - engine: Hertz 引擎实例
//   - container: go-boot IoC 容器
//   - config: 服务器通用配置
//   - middleware: 路由级中间件
//   - registerFuncs: 注册函数列表
//   - globalMiddleware: 全局中间件
//   - certFile: TLS 证书文件路径（HTTPS）
//   - keyFile: TLS 私钥文件路径（HTTPS）
type HertzServer struct {
	host             string
	port             int
	readTimeout      time.Duration
	writeTimeout     time.Duration
	idleTimeout      time.Duration
	shutdownTimeout  time.Duration
	engine           *server.Hertz
	container        core.Container
	middleware       []any
	registerFuncs    []func(container core.Container) error
	globalMiddleware []any
	certFile         string
	keyFile          string
}

type hertzHandlerContext struct {
	ctx     context.Context
	c       *app.RequestContext
	aborted bool
}

func (h *hertzHandlerContext) RequestMethod() string {
	return string(h.c.Method())
}

func (h *hertzHandlerContext) RequestURI() string {
	return string(h.c.URI().RequestURI())
}

func (h *hertzHandlerContext) Header(key string) string {
	return string(h.c.GetHeader(key))
}

func (h *hertzHandlerContext) SetStatusCode(code int) {
	h.c.SetStatusCode(code)
}

func (h *hertzHandlerContext) SetHeader(key, value string) {
	h.c.Header(key, value)
}

func (h *hertzHandlerContext) AbortWithStatus(code int) {
	h.aborted = true
	h.c.AbortWithStatus(code)
}

func (h *hertzHandlerContext) AbortWithStatusJSON(code int, body interface{}) {
	h.aborted = true
	h.c.AbortWithStatusJSON(code, body)
}

func (h *hertzHandlerContext) Next() {
	h.c.Next(h.ctx)
}

func (h *hertzHandlerContext) IsAborted() bool {
	return h.aborted
}

func (h *hertzHandlerContext) Context() context.Context {
	return h.ctx
}

func (h *hertzHandlerContext) SetContext(ctx context.Context) {
	h.ctx = ctx
}

// NewServer 创建新的 Hertz HTTP 服务器。
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - *HertzServer: 配置好的服务器实例
func NewServer(opts ...ServerOption) *HertzServer {
	s := &HertzServer{
		host:             "localhost",
		port:             8080,
		container:        core.New(),
		readTimeout:      30 * time.Second,
		writeTimeout:     30 * time.Second,
		idleTimeout:      60 * time.Second,
		shutdownTimeout:  5 * time.Second,
		middleware:       make([]any, 0),
		registerFuncs:    make([]func(container core.Container) error, 0),
		globalMiddleware: make([]any, 0),
	}

	for _, opt := range opts {
		opt(s)
	}

	hostPort := fmt.Sprintf("%s:%d", s.host, s.port)
	hertzOpts := []config.Option{
		server.WithHostPorts(hostPort),
		server.WithReadTimeout(s.readTimeout),
		server.WithWriteTimeout(s.writeTimeout),
		server.WithIdleTimeout(s.idleTimeout),
	}

	if s.certFile != "" && s.keyFile != "" {
		cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
		if err != nil {
			panic("hertz: load TLS cert/key failed: " + err.Error())
		}
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		hertzOpts = append(hertzOpts, server.WithTLS(tlsCfg))
	}

	s.engine = server.Default(hertzOpts...)

	return s
}

// WithServerContainer 设置自定义容器。
//
// 参数:
//   - c: go-boot 容器实例
//
// 返回值:
//   - ServerOption: 服务器配置选项函数
func WithServerContainer(c core.Container) ServerOption {
	return func(s *HertzServer) {
		s.container = c
	}
}

// WithHost 设置服务器监听地址。
//
// 参数:
//   - host: 监听地址，如 ":8080"
//
// 返回值:
//   - ServerOption: 服务器配置选项函数
func WithHost(host string) ServerOption {
	return func(s *HertzServer) {
		s.host = host
	}
}

// WithPort 设置服务器监听端口。
//
// 参数:
//   - port: 监听端口，如 8080
//
// 返回值:
//   - ServerOption: 服务器配置选项函数
func WithPort(port int) ServerOption {
	return func(s *HertzServer) {
		s.port = port
	}
}

// WithShutdownTimeout 设置服务器优雅关机超时时间。
//
// 参数:
//   - timeout: 关机超时时间
//
// 返回值:
//   - ServerOption: 服务器配置选项函数
func WithShutdownTimeout(timeout time.Duration) ServerOption {
	return func(s *HertzServer) {
		s.shutdownTimeout = timeout
	}
}

// WithReadTimeout 设置读取超时时间。
//
// 参数:
//   - timeout: 读取超时时间
//
// 返回值:
//   - ServerOption: 服务器配置选项函数
func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(s *HertzServer) {
		s.readTimeout = timeout
	}
}

// WithWriteTimeout 设置写入超时时间。
//
// 参数:
//   - timeout: 写入超时时间
//
// 返回值:
//   - ServerOption: 服务器配置选项函数
func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(s *HertzServer) {
		s.writeTimeout = timeout
	}
}

// WithIdleTimeout 设置连接空闲超时时间。
//
// 参数:
//   - timeout: 空闲超时时间
//
// 返回值:
//   - ServerOption: 服务器配置选项函数
func WithIdleTimeout(timeout time.Duration) ServerOption {
	return func(s *HertzServer) {
		s.idleTimeout = timeout
	}
}

// WithTLS 配置 HTTPS，设置证书和密钥文件路径。
//
// 参数:
//   - certFile: PEM 格式的证书文件路径
//   - keyFile: PEM 格式的私钥文件路径
//
// 返回值:
//   - ServerOption: 服务器配置选项函数
//
// 示例:
//
//	s := server.NewServer(
//	    server.WithTLS("server.crt", "server.key"),
//	)
//	s.Start() // 使用 HTTPS 启动
func WithTLS(certFile, keyFile string) ServerOption {
	return func(s *HertzServer) {
		s.certFile = certFile
		s.keyFile = keyFile
	}
}

// ServerOption 是服务器配置选项函数。
type ServerOption func(*HertzServer)

// HandlerFunc 是请求处理函数类型。
type HandlerFunc func(ctx context.Context, c *app.RequestContext)

// Start 启动 HTTP 服务器并开始监听请求。
//
// 在启动时会：
// 1. 执行所有注册函数
// 2. 配置中间件
// 3. 启动服务器监听指定端口
// 4. 等待中断信号 (SIGINT/SIGTERM)
// 5. 优雅关闭服务器
//
// 返回值:
//   - error: 启动或关闭时的错误
func (s *HertzServer) Start() error {
	for _, fn := range s.registerFuncs {
		if err := fn(s.container); err != nil {
			return fmt.Errorf("failed to register function: %w", err)
		}
	}

	s.setupMiddleware()

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	log.Printf("Starting Hertz server on %s", addr)

	go s.engine.Spin()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Hertz server...")
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err := s.engine.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}
	log.Println("Hertz server stopped")

	return nil
}

// setupMiddleware 配置服务器中间件。
func (s *HertzServer) setupMiddleware() {
	for _, m := range s.globalMiddleware {
		switch v := m.(type) {
		case app.HandlerFunc:
			s.engine.Use(v)
		case net.MiddlewareFunc:
			s.engine.Use(s.AdaptMiddleware(v))
		}
	}

	for _, m := range s.middleware {
		switch v := m.(type) {
		case app.HandlerFunc:
			s.engine.Use(v)
		case net.MiddlewareFunc:
			s.engine.Use(s.AdaptMiddleware(v))
		}
	}
}

// AdaptMiddleware 将 net.MiddlewareFunc 适配为 Hertz 的 app.HandlerFunc。
//
// 参数:
//   - m: go-boot 中间件函数
//
// 返回值:
//   - app.HandlerFunc: Hertz 处理函数
func (s *HertzServer) AdaptMiddleware(m net.MiddlewareFunc) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		hc := &hertzHandlerContext{ctx: ctx, c: c}
		m(hc)
	}
}

// Use 向服务器的路由中间件链追加一个中间件。
//
// 参数:
//   - m: 中间件函数
//
// 返回值:
//   - net.Server: 服务器实例
func (s *HertzServer) Use(m any) {
	s.middleware = append(s.middleware, m)
}

// UseGlobal 向服务器的全局中间件链 prepend 一个中间件，全局中间件在所有路由之前执行。
//
// 参数:
//   - m: 中间件函数
//
// 返回值:
//   - net.Server: 服务器实例
func (s *HertzServer) UseGlobal(m any) {
	s.globalMiddleware = append(s.globalMiddleware, m)
}

// Register 注册一个处理函数到容器中，用于依赖注入。
//
// 参数:
//   - fn: 注册函数，接受 core.Container 参数
//
// 返回值:
//   - net.Server: 服务器实例
func (s *HertzServer) Register(fn func(container core.Container) error) {
	s.registerFuncs = append(s.registerFuncs, fn)
}

// Container 返回 go-boot IoC 容器实例。
func (s *HertzServer) Container() any {
	return s.container
}

// Stop 优雅地停止服务器，等待正在处理的请求完成。
//
// 参数:
//   - ctx: 上下文，用于控制停止超时
//
// 返回值:
//   - error: 关闭错误
func (s *HertzServer) Stop(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()
	return s.engine.Shutdown(shutdownCtx)
}

// 编译时检查 HertzServer 是否实现 net.Server 接口
var _ net.Server = (*HertzServer)(nil)

// Engine 返回底层的 Hertz 引擎实例，用于高级配置。
//
// 返回值:
//   - *server.Hertz: Hertz 引擎实例
func (s *HertzServer) Engine() *server.Hertz {
	return s.engine
}

// GET 注册 GET 路由。
//
// 参数:
//   - path: 路由路径
//   - handlers: 处理函数
//
// 返回值:
//   - *HertzServer: 服务器实例，支持链式调用
func (s *HertzServer) GET(path string, handlers ...any) *HertzServer {
	s.engine.GET(path, s.wrapHandlers(handlers)...)
	return s
}

// POST 注册 POST 路由。
//
// 参数:
//   - path: 路由路径
//   - handlers: 处理函数
//
// 返回值:
//   - *HertzServer: 服务器实例，支持链式调用
func (s *HertzServer) POST(path string, handlers ...any) *HertzServer {
	s.engine.POST(path, s.wrapHandlers(handlers)...)
	return s
}

// PUT 注册 PUT 路由。
//
// 参数:
//   - path: 路由路径
//   - handlers: 处理函数
//
// 返回值:
//   - *Server: 服务器实例，支持链式调用
func (s *HertzServer) PUT(path string, handlers ...any) *HertzServer {
	s.engine.PUT(path, s.wrapHandlers(handlers)...)
	return s
}

// DELETE 注册 DELETE 路由。
//
// 参数:
//   - path: 路由路径
//   - handlers: 处理函数
//
// 返回值:
//   - *Server: 服务器实例，支持链式调用
func (s *HertzServer) DELETE(path string, handlers ...any) *HertzServer {
	s.engine.DELETE(path, s.wrapHandlers(handlers)...)
	return s
}

// PATCH 注册 PATCH 路由。
//
// 参数:
//   - path: 路由路径
//   - handlers: 处理函数
//
// 返回值:
//   - *Server: 服务器实例，支持链式调用
func (s *HertzServer) PATCH(path string, handlers ...any) *HertzServer {
	s.engine.PATCH(path, s.wrapHandlers(handlers)...)
	return s
}

// HEAD 注册 HEAD 路由。
//
// 参数:
//   - path: 路由路径
//   - handlers: 处理函数
//
// 返回值:
//   - *Server: 服务器实例，支持链式调用
func (s *HertzServer) HEAD(path string, handlers ...any) *HertzServer {
	s.engine.HEAD(path, s.wrapHandlers(handlers)...)
	return s
}

// OPTIONS 注册 OPTIONS 路由。
//
// 参数:
//   - path: 路由路径
//   - handlers: 处理函数
//
// 返回值:
//   - *Server: 服务器实例，支持链式调用
func (s *HertzServer) OPTIONS(path string, handlers ...any) *HertzServer {
	s.engine.OPTIONS(path, s.wrapHandlers(handlers)...)
	return s
}

// Any 注册接受所有 HTTP 方法的路由。
//
// 参数:
//   - path: 路由路径
//   - handlers: 处理函数
//
// 返回值:
//   - *Server: 服务器实例，支持链式调用
func (s *HertzServer) Any(path string, handlers ...any) *HertzServer {
	s.engine.Any(path, s.wrapHandlers(handlers)...)
	return s
}

// Group 创建一个路由组。
//
// 参数:
//   - relativePath: 路由组的基础路径
//   - handlers: 路由组级中间件
//
// 返回值:
//   - *route.RouterGroup: 路由组
func (s *HertzServer) Group(relativePath string, handlers ...any) *route.RouterGroup {
	return s.engine.Group(relativePath, s.wrapHandlers(handlers)...)
}

func (s *HertzServer) wrapHandlers(handlers []any) []app.HandlerFunc {
	result := make([]app.HandlerFunc, 0, len(handlers))
	for _, h := range handlers {
		switch v := h.(type) {
		case app.HandlerFunc:
			result = append(result, v)
		case func(ctx context.Context, c *app.RequestContext):
			result = append(result, v)
		case HandlerFunc:
			result = append(result, app.HandlerFunc(v))
		case net.MiddlewareFunc:
			result = append(result, s.AdaptMiddleware(v))
		}
	}
	return result
}
