package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"
)

// HertzWebSocketHandler Hertz WebSocket 处理器
// 封装 hertz-contrib/websocket 的升级和连接处理逻辑
type HertzWebSocketHandler struct {
	upgrader  websocket.HertzUpgrader
	onConnect func(*websocket.Conn)
}

// NewHertzWebSocketHandler 创建 Hertz WebSocket 处理器
func NewHertzWebSocketHandler(opts ...WebSocketOption) *HertzWebSocketHandler {
	handler := &HertzWebSocketHandler{
		upgrader: websocket.HertzUpgrader{},
	}
	for _, opt := range opts {
		opt(handler)
	}
	return handler
}

// WebSocketOption WebSocket 配置选项
type WebSocketOption func(*HertzWebSocketHandler)

// WithReadBufferSize 设置读取缓冲区大小
func WithReadBufferSize(size int) WebSocketOption {
	return func(h *HertzWebSocketHandler) {
		h.upgrader.ReadBufferSize = size
	}
}

// WithWriteBufferSize 设置写入缓冲区大小
func WithWriteBufferSize(size int) WebSocketOption {
	return func(h *HertzWebSocketHandler) {
		h.upgrader.WriteBufferSize = size
	}
}

// WithCheckOrigin 设置跨域检查函数
func WithCheckOrigin(fn func(ctx *app.RequestContext) bool) WebSocketOption {
	return func(h *HertzWebSocketHandler) {
		h.upgrader.CheckOrigin = fn
	}
}

// WithOnConnect 设置连接建立回调
func WithOnConnect(fn func(*websocket.Conn)) WebSocketOption {
	return func(h *HertzWebSocketHandler) {
		h.onConnect = fn
	}
}

// Handle 处理 WebSocket 连接请求
func (h *HertzWebSocketHandler) Handle(c context.Context, ctx *app.RequestContext) {
	handler := func(conn *websocket.Conn) {
		if h.onConnect != nil {
			h.onConnect(conn)
		}
	}

	if err := h.upgrader.Upgrade(ctx, handler); err != nil {
		ctx.JSON(400, map[string]any{"error": err.Error()})
	}
}

// Middleware 返回 WebSocket 中间件处理函数
func (h *HertzWebSocketHandler) Middleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		if string(ctx.GetHeader("Upgrade")) != "websocket" {
			ctx.Next(c)
			return
		}

		h.Handle(c, ctx)
		ctx.Abort()
	}
}

// WebSocketMiddleware 创建 WebSocket 中间件（快捷函数）
func WebSocketMiddleware(opts ...WebSocketOption) app.HandlerFunc {
	handler := NewHertzWebSocketHandler(opts...)
	return handler.Middleware()
}
