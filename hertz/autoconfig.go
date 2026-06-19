// Package hertz 提供 Hertz HTTP 服务器的自动配置。
//
// 当 hertz.enabled=true 时自动启用，从 Environment 中读取 hertz.host、hertz.read-timeout、
// hertz.write-timeout 等配置项，
// 创建并注册 Hertz Server Bean 到 IoC 容器中（Bean ID: hertzServer）。
package hertz

import (
	"time"

	"github.com/xudefa/go-boot-hertz/server"
	"github.com/xudefa/go-boot/boot"
	"github.com/xudefa/go-boot/condition"
	"github.com/xudefa/go-boot/constants"
	"github.com/xudefa/go-boot/core"
)

// init 注册 Hertz 自动配置，由 hertz.enabled=true 条件控制。
func init() {
	boot.RegisterAutoConfig(&HertzAutoConfiguration{},
		condition.OnProperty(constants.HertzEnabled, constants.ConditionTrue),
	)
}

// HertzAutoConfiguration Hertz HTTP 服务器的自动配置。
//
// 从 Environment 中读取 hertz.host、hertz.read-timeout、hertz.write-timeout 等配置项，
// 创建 Hertz Server 实例并注册到 IoC 容器中。
// 启用条件：hertz.enabled=true
type HertzAutoConfiguration struct{}

// Configure 执行自动配置逻辑，创建 Hertz Server 并注册为 Bean。
func (h *HertzAutoConfiguration) Configure(ctx boot.ApplicationContext) error {
	env := ctx.Environment()

	opts := []server.ServerOption{
		server.WithServerContainer(ctx.Container()),
	}

	if host := env.GetString(constants.HertzHost, ""); host != "" {
		opts = append(opts, server.WithHost(host))
	}

	opts = append(opts,
		server.WithReadTimeout(time.Duration(env.GetInt(constants.HertzReadTimeout, constants.DefaultHertzReadTimeout))*time.Second),
		server.WithWriteTimeout(time.Duration(env.GetInt(constants.HertzWriteTimeout, constants.DefaultHertzWriteTimeout))*time.Second),
	)

	s := server.NewServer(opts...)

	if err := ctx.Register(constants.HertzServerBeanID,
		core.Bean(s),
		core.Singleton(),
	); err != nil {
		return err
	}

	return nil
}
