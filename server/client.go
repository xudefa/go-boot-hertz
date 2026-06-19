// Package server 基于 Hertz 框架的客户端实现。
//
// 定义：
//
//   - HertzClient: HTTP 客户端实现了 net.HttpClient 接口
//   - HttpClientOption: 客户端配置选项
//
// 快速开始:
//
//	client, _ := server.NewHertzClient()
//	resp, _ := client.Get(context.Background(), "/api/hello")
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/xudefa/go-boot/net"
)

// HertzClient 是 Hertz HTTP 客户端，实现了 net.HertzClient 接口。
//
// 字段说明:
//   - hertzClient: Hertz 客户端实例
//   - baseURL: 基础 URL
//   - middleware: 客户端中间件
type HertzClient struct {
	hertzClient *client.Client
	baseURL     string
	middleware  []client.Middleware
}

// NewHertzClient 创建新的 Hertz HTTP 客户端。
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - *HertzClient: 配置好的客户端实例
//   - error: 创建错误
func NewHertzClient(opts ...HttpClientOption) (*HertzClient, error) {
	c, err := client.NewClient()
	if err != nil {
		return nil, err
	}

	h := &HertzClient{
		hertzClient: c,
		baseURL:     "http://localhost:8080",
	}

	for _, opt := range opts {
		opt(h)
	}

	for _, m := range h.middleware {
		h.hertzClient.Use(m)
	}

	return h, nil
}

// WithHertzClientBaseURL 设置客户端的基础 URL。
//
// 参数:
//   - baseURL: 基础 URL 地址
//
// 返回值:
//   - HttpClientOption: 客户端配置选项函数
func WithHertzClientBaseURL(baseURL string) HttpClientOption {
	return func(h *HertzClient) {
		h.baseURL = baseURL
	}
}

// WithHertzClientMiddleware 添加客户端中间件。
//
// 参数:
//   - m: 客户端中间件函数
//
// 返回值:
//   - HttpClientOption: 客户端配置选项函数
func WithHertzClientMiddleware(m client.Middleware) HttpClientOption {
	return func(h *HertzClient) {
		h.middleware = append(h.middleware, m)
	}
}

// HttpClientOption 是客户端配置选项函数。
type HttpClientOption func(*HertzClient)

// Get 发送 GET 请求。
//
// 参数:
//   - ctx: 上下文
//   - url: 请求路径
//   - opts: 可选的请求选项
//
// 返回值:
//   - *net.Response: 响应对象
//   - error: 请求错误
func (h *HertzClient) Get(ctx context.Context, url string, opts ...net.RequestOption) (*net.HttpResponse, error) {
	return h.doRequest(ctx, consts.MethodGet, url, nil, opts...)
}

// Post 发送 POST 请求。
//
// 参数:
//   - ctx: 上下文
//   - url: 请求路径
//   - body: 请求体
//   - opts: 可选的请求选项
//
// 返回值:
//   - *net.Response: 响应对象
//   - error: 请求错误
func (h *HertzClient) Post(ctx context.Context, url string, body any, opts ...net.RequestOption) (*net.HttpResponse, error) {
	return h.doRequest(ctx, consts.MethodPost, url, body, opts...)
}

// Put 发送 PUT 请求。
//
// 参数:
//   - ctx: 上下文
//   - url: 请求路径
//   - body: 请求体
//   - opts: 可选的请求选项
//
// 返回值:
//   - *net.Response: 响应对象
//   - error: 请求错误
func (h *HertzClient) Put(ctx context.Context, url string, body any, opts ...net.RequestOption) (*net.HttpResponse, error) {
	return h.doRequest(ctx, consts.MethodPut, url, body, opts...)
}

// Delete 发送 DELETE 请求。
//
// 参数:
//   - ctx: 上下文
//   - url: 请求路径
//   - opts: 可选的请求选项
//
// 返回值:
//   - *net.Response: 响应对象
//   - error: 请求错误
func (h *HertzClient) Delete(ctx context.Context, url string, opts ...net.RequestOption) (*net.HttpResponse, error) {
	return h.doRequest(ctx, consts.MethodDelete, url, nil, opts...)
}

// Head 发送 HEAD 请求。
//
// 参数:
//   - ctx: 上下文
//   - url: 请求路径
//   - opts: 可选的请求选项
//
// 返回值:
//   - *net.Response: 响应对象
//   - error: 请求错误
func (h *HertzClient) Head(ctx context.Context, url string, opts ...net.RequestOption) (*net.HttpResponse, error) {
	return h.doRequest(ctx, consts.MethodHead, url, nil, opts...)
}

// Options 发送 OPTIONS 请求。
//
// 参数:
//   - ctx: 上下文
//   - url: 请求路径
//   - opts: 可选的请求选项
//
// 返回值:
//   - *net.Response: 响应对象
//   - error: 请求错误
func (h *HertzClient) Options(ctx context.Context, url string, opts ...net.RequestOption) (*net.HttpResponse, error) {
	return h.doRequest(ctx, consts.MethodOptions, url, nil, opts...)
}

// Patch 发送 PATCH 请求。
//
// 参数:
//   - ctx: 上下文
//   - url: 请求路径
//   - body: 请求体
//   - opts: 可选的请求选项
//
// 返回值:
//   - *net.Response: 响应对象
//   - error: 请求错误
func (h *HertzClient) Patch(ctx context.Context, url string, body any, opts ...net.RequestOption) (*net.HttpResponse, error) {
	return h.doRequest(ctx, consts.MethodPatch, url, body, opts...)
}

// Do 发送自定义请求。
//
// 参数:
//   - ctx: 上下文
//   - req: 请求对象
//
// 返回值:
//   - *net.Response: 响应对象
//   - error: 请求错误
func (h *HertzClient) Do(ctx context.Context, req any) (*net.HttpResponse, error) {
	hzReq, ok := req.(*protocol.Request)
	if !ok {
		return nil, fmt.Errorf("hertz: Do requires *protocol.Request, got %T", req)
	}

	resp := &protocol.Response{}
	err := h.hertzClient.Do(ctx, hzReq, resp)
	if err != nil {
		return nil, err
	}

	return h.buildResponse(resp), nil
}

// Close 关闭客户端连接。
func (h *HertzClient) Close() error {
	return nil
}

func (h *HertzClient) buildRequest(ctx context.Context, method string, url string, body any, opts ...net.RequestOption) (*protocol.Request, error) {
	req := &protocol.Request{}

	cfg := &net.HttpRequest{}
	for _, opt := range opts {
		opt(cfg)
	}

	requestURI := h.baseURL + url
	if len(cfg.Query) > 0 {
		separator := "?"
		if strings.Contains(requestURI, "?") {
			separator = "&"
		}
		requestURI = requestURI + separator + cfg.Query.Encode()
	}

	req.Header.SetMethod(method)
	req.Header.SetRequestURI(requestURI)

	for key, values := range cfg.Header {
		for _, value := range values {
			req.Header.Set(key, value)
		}
	}

	if cfg.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
	}

	if cfg.BasicAuth.Username != "" || cfg.BasicAuth.Password != "" {
		req.SetBasicAuth(cfg.BasicAuth.Username, cfg.BasicAuth.Password)
	}

	if body != nil {
		data, err := marshalBody(body)
		if err != nil {
			return nil, err
		}
		req.SetBody(data)
		req.Header.SetContentTypeBytes([]byte(consts.MIMEApplicationJSON))
	}

	return req, nil
}

func (h *HertzClient) doRequest(ctx context.Context, method string, url string, body any, opts ...net.RequestOption) (*net.HttpResponse, error) {
	req, err := h.buildRequest(ctx, method, url, body, opts...)
	if err != nil {
		return nil, err
	}

	resp := &protocol.Response{}
	err = h.hertzClient.Do(ctx, req, resp)
	if err != nil {
		return nil, err
	}

	return h.buildResponse(resp), nil
}

func (h *HertzClient) buildResponse(resp *protocol.Response) *net.HttpResponse {
	header := make(http.Header)
	resp.Header.VisitAll(func(key, value []byte) {
		header.Set(string(key), string(value))
	})

	return &net.HttpResponse{
		StatusCode: resp.StatusCode(),
		Header:     header,
		Body:       resp.Body(),
	}
}

func marshalBody(v any) ([]byte, error) {
	switch data := v.(type) {
	case []byte:
		return data, nil
	case string:
		return []byte(data), nil
	case io.Reader:
		return io.ReadAll(data)
	default:
		return json.Marshal(v)
	}
}
