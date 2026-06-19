package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/xudefa/go-boot/tracing"
)

// hertzCarrier 实现 tracing.TextMapCarrier 接口，用于在 Hertz 请求中提取和注入追踪上下文
type hertzCarrier struct {
	ctx *app.RequestContext
}

// Get 获取指定键的 header 值
func (c *hertzCarrier) Get(key string) string {
	return string(c.ctx.Request.Header.Peek(key))
}

// Set 设置指定键的 header 值
func (c *hertzCarrier) Set(key string, value string) {
	c.ctx.Request.Header.Set(key, value)
}

// Keys 返回所有 header 键的列表
func (c *hertzCarrier) Keys() []string {
	keys := make([]string, 0)
	c.ctx.Request.Header.VisitAll(func(key, value []byte) {
		keys = append(keys, string(key))
	})
	return keys
}

// GetTraceID 从 context.Context 中提取当前 Span 的 TraceID
// 返回空字符串如果没有有效的追踪上下文
func GetTraceID(c context.Context) string {
	return getTraceIDFromContext(c)
}

// GetSpanID 从 context.Context 中提取当前 Span 的 SpanID
// 返回空字符串如果没有有效的追踪上下文
func GetSpanID(c context.Context) string {
	return getSpanIDFromContext(c)
}

func getTraceIDFromContext(ctx context.Context) string {
	span := tracing.SpanFromContext(ctx)
	if span == nil || span.GetTraceID() == "" {
		return ""
	}
	return span.GetTraceID()
}

func getSpanIDFromContext(ctx context.Context) string {
	span := tracing.SpanFromContext(ctx)
	if span == nil || span.GetSpanID() == "" {
		return ""
	}
	return span.GetSpanID()
}

// AddTraceToResponseHeaders 将 TraceID 和 SpanID 添加到响应头中
// 便于客户端获取追踪信息进行问题排查
func AddTraceToResponseHeaders(c context.Context, ctx *app.RequestContext) {
	traceID := GetTraceID(c)
	spanID := GetSpanID(c)
	if traceID != "" {
		ctx.Response.Header.Set("X-Trace-ID", traceID)
	}
	if spanID != "" {
		ctx.Response.Header.Set("X-Span-ID", spanID)
	}
}

// TraceIDMiddleware 简单的中间件，仅将追踪 ID 添加到响应头
// 适用于不需要完整追踪功能但需要暴露追踪 ID 的场景
func TraceIDMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		AddTraceToResponseHeaders(c, ctx)
		ctx.Next(c)
	}
}

// HTTPServerTracingMiddleware 创建 Hertz HTTP 服务器端追踪中间件
// serviceName 参数用于标识服务名称，默认为 "hertz-server"
//
// 该中间件提供以下功能：
// 1. 从请求头中提取父追踪上下文
// 2. 创建服务端 Span，记录 HTTP 方法、路径、主机等信息
// 3. 将 Span 上下文注入到请求中
// 4. 在请求结束时记录响应状态码和错误状态
// 5. 将 TraceID/SpanID 添加到响应头
func HTTPServerTracingMiddleware(serviceName ...string) app.HandlerFunc {
	tracerName := "hertz-server"
	if len(serviceName) > 0 {
		tracerName = serviceName[0]
	}

	return func(c context.Context, ctx *app.RequestContext) {
		carrier := &hertzCarrier{ctx: ctx}
		parentCtx := tracing.ExtractTraceContext(c, carrier)

		spanName := string(ctx.Request.URI().RequestURI())
		if spanName == "" {
			spanName = "HTTP " + string(ctx.Method())
		}

		tracer := tracing.GetTracer(tracerName)
		ctx2, span := tracing.StartHTTPServerSpan(parentCtx, tracer, spanName,
			string(ctx.Method()),
			string(ctx.Request.URI().RequestURI()),
			string(ctx.Request.URI().Host()),
		)
		defer span.End()

		ctx.Next(ctx2)

		statusCode := ctx.Response.StatusCode()
		span.SetAttribute("http.status_code", statusCode)

		if statusCode >= consts.StatusInternalServerError {
			span.SetStatus(tracing.SpanStatusError)
		} else {
			span.SetStatus(tracing.SpanStatusOK)
		}

		AddTraceToResponseHeaders(ctx2, ctx)
	}
}

// InjectTraceHeaders 将当前追踪上下文注入到请求头中
// 用于客户端向外发起请求时传递追踪信息
func InjectTraceHeaders(c context.Context, ctx *app.RequestContext) {
	tracing.InjectTraceContext(c, &hertzCarrier{ctx: ctx})
}

// HTTPClientTracingMiddleware 创建 Hertz HTTP 客户端追踪中间件
// 用于在向外发起 HTTP 请求时注入追踪上下文
func HTTPClientTracingMiddleware(serviceName ...string) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		InjectTraceHeaders(c, ctx)
		ctx.Next(c)
	}
}

// 编译期类型断言，确保 hertzCarrier 实现了 TextMapCarrier 接口
var _ tracing.TextMapCarrier = (*hertzCarrier)(nil)
