// Package router 提供声明式路由注册辅助功能。
//
// 该包简化了 Hertz 路由的注册过程，支持路由分组和中间件配置。
//
// 快速开始:
//
//	r := router.New(container)
//	r.GET("/health", healthCheck)
//	r.Group("/api/v1", func(g *router.Group) {
//	    g.GET("/users", listUsers)
//	    g.POST("/users", createUser)
//	})
package router

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/xudefa/go-boot-hertz/server"
	"github.com/xudefa/go-boot/core"
)

// Router 路由注册辅助结构
type Router struct {
	container core.Container
	engine    *server.HertzServer
}

// New 创建新的路由注册辅助实例
//
// 参数:
//   - container: go-boot IoC 容器实例
//
// 返回值:
//   - *Router: 路由注册辅助实例
func New(container core.Container) *Router {
	engineBean, err := container.Get("hertzServer")
	if err != nil {
		panic("router: hertzServer bean not found in container")
	}

	engine, ok := engineBean.(*server.HertzServer)
	if !ok {
		panic("router: hertzServer bean is not of type *server.HertzServer")
	}

	return &Router{
		container: container,
		engine:    engine,
	}
}

// GET 注册 GET 路由
func (r *Router) GET(path string, handlers ...any) {
	r.engine.GET(path, handlers...)
}

// POST 注册 POST 路由
func (r *Router) POST(path string, handlers ...any) {
	r.engine.POST(path, handlers...)
}

// PUT 注册 PUT 路由
func (r *Router) PUT(path string, handlers ...any) {
	r.engine.PUT(path, handlers...)
}

// DELETE 注册 DELETE 路由
func (r *Router) DELETE(path string, handlers ...any) {
	r.engine.DELETE(path, handlers...)
}

// PATCH 注册 PATCH 路由
func (r *Router) PATCH(path string, handlers ...any) {
	r.engine.PATCH(path, handlers...)
}

// HEAD 注册 HEAD 路由
func (r *Router) HEAD(path string, handlers ...any) {
	r.engine.HEAD(path, handlers...)
}

// OPTIONS 注册 OPTIONS 路由
func (r *Router) OPTIONS(path string, handlers ...any) {
	r.engine.OPTIONS(path, handlers...)
}

// Any 注册接受所有 HTTP 方法的路由
func (r *Router) Any(path string, handlers ...any) {
	r.engine.Any(path, handlers...)
}

// Group 创建路由组
//
// 参数:
//   - relativePath: 路由组的基础路径
//   - fn: 路由组配置函数
func (r *Router) Group(relativePath string, fn func(g *Group)) {
	group := r.engine.Group(relativePath)
	g := &Group{
		routerGroup: group,
		engine:      r.engine,
	}
	fn(g)
}

// Group 路由组结构
type Group struct {
	routerGroup *route.RouterGroup
	engine      *server.HertzServer
}

// GET 注册 GET 路由到路由组
func (g *Group) GET(path string, handlers ...any) {
	g.routerGroup.GET(path, wrapHandlers(handlers)...)
}

// POST 注册 POST 路由到路由组
func (g *Group) POST(path string, handlers ...any) {
	g.routerGroup.POST(path, wrapHandlers(handlers)...)
}

// PUT 注册 PUT 路由到路由组
func (g *Group) PUT(path string, handlers ...any) {
	g.routerGroup.PUT(path, wrapHandlers(handlers)...)
}

// DELETE 注册 DELETE 路由到路由组
func (g *Group) DELETE(path string, handlers ...any) {
	g.routerGroup.DELETE(path, wrapHandlers(handlers)...)
}

// PATCH 注册 PATCH 路由到路由组
func (g *Group) PATCH(path string, handlers ...any) {
	g.routerGroup.PATCH(path, wrapHandlers(handlers)...)
}

// HEAD 注册 HEAD 路由到路由组
func (g *Group) HEAD(path string, handlers ...any) {
	g.routerGroup.HEAD(path, wrapHandlers(handlers)...)
}

// OPTIONS 注册 OPTIONS 路由到路由组
func (g *Group) OPTIONS(path string, handlers ...any) {
	g.routerGroup.OPTIONS(path, wrapHandlers(handlers)...)
}

// Use 添加路由组级中间件
func (g *Group) Use(handlers ...any) {
	g.routerGroup.Use(wrapHandlers(handlers)...)
}

// HandlerFunc 定义路由处理函数类型
type HandlerFunc func(ctx context.Context, c *app.RequestContext)

func wrapHandlers(handlers []any) []app.HandlerFunc {
	result := make([]app.HandlerFunc, 0, len(handlers))
	for _, h := range handlers {
		switch v := h.(type) {
		case app.HandlerFunc:
			result = append(result, v)
		case func(ctx context.Context, c *app.RequestContext):
			result = append(result, v)
		case HandlerFunc:
			result = append(result, app.HandlerFunc(v))
		}
	}
	return result
}
