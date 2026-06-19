package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xudefa/go-boot-hertz/server"
	"github.com/xudefa/go-boot/constants"
	"github.com/xudefa/go-boot/core"
	"github.com/xudefa/go-boot/security"
)

// HertzSecurityRequest Hertz安全请求适配器
// 将Hertz框架的*app.RequestContext适配为security.SecurityRequest接口
type HertzSecurityRequest struct {
	c *app.RequestContext
}

// NewHertzSecurityRequest 创建Hertz安全请求适配器
func NewHertzSecurityRequest(c *app.RequestContext) *HertzSecurityRequest {
	return &HertzSecurityRequest{c: c}
}

// GetMethod 获取HTTP请求方法
func (r *HertzSecurityRequest) GetMethod() string {
	return string(r.c.Method())
}

// GetURI 获取请求URI路径
func (r *HertzSecurityRequest) GetURI() string {
	return string(r.c.URI().Path())
}

// GetHeader 获取请求头值
func (r *HertzSecurityRequest) GetHeader(key string) string {
	return string(r.c.GetHeader(key))
}

// SetAttribute 设置请求属性
func (r *HertzSecurityRequest) SetAttribute(key string, value any) {
	r.c.Set(key, value)
}

// GetAttribute 获取请求属性
func (r *HertzSecurityRequest) GetAttribute(key string) (any, bool) {
	return r.c.Get(key)
}

// HertzSecurityResponse Hertz安全响应适配器
// 将Hertz框架的响应适配为security.SecurityResponse接口
type HertzSecurityResponse struct {
	c *app.RequestContext
}

// NewHertzSecurityResponse 创建Hertz安全响应适配器
func NewHertzSecurityResponse(c *app.RequestContext) *HertzSecurityResponse {
	return &HertzSecurityResponse{c: c}
}

// SetStatusCode 设置HTTP响应状态码
func (r *HertzSecurityResponse) SetStatusCode(code int) {
	r.c.SetStatusCode(code)
}

// SetHeader 设置响应头
func (r *HertzSecurityResponse) SetHeader(key, value string) {
	r.c.Header(key, value)
}

// Write 写入响应体
func (r *HertzSecurityResponse) Write(data []byte) error {
	_, err := r.c.Write(data)
	return err
}

// HertzSecurityFilterChain Hertz安全过滤器链
// 包装security.SecurityFilterChain以适配Hertz框架
type HertzSecurityFilterChain struct {
	securityChain security.SecurityFilterChain
}

// NewHertzSecurityFilterChain 创建Hertz安全过滤器链
func NewHertzSecurityFilterChain(securityChain security.SecurityFilterChain) *HertzSecurityFilterChain {
	return &HertzSecurityFilterChain{securityChain: securityChain}
}

// DoFilter 执行安全过滤器链
func (f *HertzSecurityFilterChain) DoFilter(ctx context.Context, request security.SecurityRequest, response security.SecurityResponse) error {
	return f.securityChain.DoFilter(ctx, request, response)
}

// HertzSecurityMiddleware Hertz安全中间件
// 将安全过滤器链转换为Hertz中间件
type HertzSecurityMiddleware struct {
	securityChain security.SecurityFilterChain
}

// NewHertzSecurityMiddleware 创建Hertz安全中间件
func NewHertzSecurityMiddleware(securityChain security.SecurityFilterChain) *HertzSecurityMiddleware {
	return &HertzSecurityMiddleware{securityChain: securityChain}
}

// HandlerFunc 返回app.HandlerFunc格式的中间件处理函数
func (m *HertzSecurityMiddleware) HandlerFunc() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		request := NewHertzSecurityRequest(c)
		response := NewHertzSecurityResponse(c)

		securityChain := NewHertzSecurityFilterChain(m.securityChain)

		err := securityChain.DoFilter(ctx, request, response)
		if err != nil {
			c.JSON(500, map[string]string{"error": err.Error()})
			c.Abort()
			return
		}

		c.Next(ctx)
	}
}

// WithSecurity 创建Hertz服务器的security配置选项
// 参数securityChain: 安全过滤器链，通常通过security.NewHttpSecurity().Build()构建
// 返回ServerOption函数，可传入server.NewServer()进行配置
//
// 使用示例:
//
//	import (
//	    "github.com/xudefa/go-boot/security"
//	    "github.com/xudefa/go-boot-hertz/server"
//	    "github.com/xudefa/go-boot-hertz/middleware"
//	)
//
//	// 1. 创建用户服务
//	userDetailsService := security.NewInMemoryUserDetailsService()
//	userDetailsService.CreateUser("admin", "password", []string{"ROLE_ADMIN"})
//
//	// 2. 创建密码编码器和认证提供者
//	passwordEncoder := security.NewBCryptPasswordEncoder(10)
//	authProvider := security.NewDaoAuthenticationProvider(userDetailsService, passwordEncoder)
//	authManager := security.NewProviderManager(authProvider)
//
//	// 3. 创建安全元数据源
//	metadataSource := security.NewExpressionBasedFilterInvocationSecurityMetadataSource()
//	metadataSource.AddMapping("/public/**", []string{"permitAll"})
//	metadataSource.AddMapping("/admin/**", []string{"hasRole('ADMIN')"})
//
//	// 4. 构建安全过滤器链
//	httpSecurity := security.NewHttpSecurity()
//	httpSecurity.AuthenticationManager(authManager)
//	httpSecurity.SecurityMetadataSource(metadataSource)
//	httpSecurity.Anonymous()
//	chain, _ := httpSecurity.Build()
//
//	// 5. 创建Hertz服务器并应用安全配置
//	s := server.NewServer(middleware.WithSecurity(chain))
//	s.GET("/public/test", func(ctx context.Context, c *app.RequestContext) {
//	    c.String(200, "public endpoint")
//	})
//	s.GET("/admin/test", func(ctx context.Context, c *app.RequestContext) {
//	    c.String(200, "admin endpoint")
//	})
func WithSecurity(securityChain security.SecurityFilterChain) server.ServerOption {
	return func(s *server.HertzServer) {
		securityMiddleware := NewHertzSecurityMiddleware(securityChain)
		s.Engine().Use(securityMiddleware.HandlerFunc())
	}
}

// WithSecurityFromContainer 从容器中获取安全过滤器链并应用
// 参数container: go-boot IoC容器
// 返回ServerOption函数
func WithSecurityFromContainer(container core.Container) server.ServerOption {
	return func(s *server.HertzServer) {
		bean, err := container.Get(constants.SecurityFilterChainBeanID)
		if err != nil {
			return
		}
		securityChain, ok := bean.(security.SecurityFilterChain)
		if !ok || securityChain == nil {
			return
		}
		securityMiddleware := NewHertzSecurityMiddleware(securityChain)
		s.Engine().Use(securityMiddleware.HandlerFunc())
	}
}
