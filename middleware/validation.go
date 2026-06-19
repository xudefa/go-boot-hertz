package middleware

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xudefa/go-boot/validation"
)

// ValidationMiddleware 创建 Hertz 验证中间件
func ValidationMiddleware(validator validation.Validator) app.HandlerFunc {
	return ValidationMiddlewareWithConfig(&validation.MiddlewareConfig{
		Validator: validator,
	})
}

// ValidationMiddlewareWithGroups 创建带验证组的 Hertz 中间件
func ValidationMiddlewareWithGroups(validator validation.Validator, groups ...string) app.HandlerFunc {
	return ValidationMiddlewareWithConfig(&validation.MiddlewareConfig{
		Validator: validator,
		Groups:    groups,
	})
}

// ValidationMiddlewareWithConfig 创建自定义配置的 Hertz 中间件
func ValidationMiddlewareWithConfig(config *validation.MiddlewareConfig) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if shouldSkipHertzPath(string(c.URI().Path()), config.SkipPaths) {
			c.Next(ctx)
			return
		}

		obj := getHertzRequestObject(c)
		if obj == nil {
			c.Next(ctx)
			return
		}

		var err error
		if groupedValidator, ok := config.Validator.(*validation.GroupedTagValidator); ok && len(config.Groups) > 0 {
			err = groupedValidator.ValidateWithGroups(obj, config.Groups...)
		} else {
			err = config.Validator.Validate(obj)
		}

		if err != nil {
			if config.ErrorHandler != nil {
				config.ErrorHandler(c, err)
			} else {
				defaultHertzErrorHandler(c, err)
			}
			c.Abort()
			return
		}

		c.Next(ctx)
	}
}

// shouldSkipHertzPath 检查是否应该跳过 Hertz 路径
func shouldSkipHertzPath(path string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// getHertzRequestObject 从 Hertz 上下文获取请求对象
func getHertzRequestObject(c *app.RequestContext) interface{} {
	if string(c.Method()) == "GET" {
		return c.QueryArgs()
	}

	if c.Request.Body() == nil {
		return nil
	}

	body := c.Request.Body()
	var obj interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil
	}

	c.Request.SetBody(body)
	return obj
}

// defaultHertzErrorHandler 默认 Hertz 错误处理器
func defaultHertzErrorHandler(c *app.RequestContext, err error) {
	if c == nil {
		return
	}
	c.JSON(400, map[string]interface{}{
		"error": err.Error(),
	})
}

// BindAndValidate 绑定并验证请求
func BindAndValidate(ctx context.Context, c *app.RequestContext, obj interface{}, validator validation.Validator) error {
	if err := c.BindJSON(obj); err != nil {
		return err
	}

	if validator != nil {
		return validator.Validate(obj)
	}

	return nil
}

// BindAndValidateWithGroups 绑定并验证请求（带验证组）
func BindAndValidateWithGroups(ctx context.Context, c *app.RequestContext, obj interface{}, validator validation.Validator, groups ...string) error {
	if err := c.BindJSON(obj); err != nil {
		return err
	}

	if groupedValidator, ok := validator.(*validation.GroupedTagValidator); ok && len(groups) > 0 {
		return groupedValidator.ValidateWithGroups(obj, groups...)
	}

	if validator != nil {
		return validator.Validate(obj)
	}

	return nil
}
