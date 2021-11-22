package golib

import (
	"github.com/gin-gonic/gin"
	"github.com/myksc/ksc-base/golib/base"
	"github.com/myksc/ksc-base/golib/env"
	"github.com/myksc/ksc-base/golib/middleware"
	"github.com/myksc/ksc-base/golib/server/unix"
)

type BootstrapConf struct {
	Pprof bool `yaml:"pprof"`
}

func Bootstraps(router *gin.Engine, conf BootstrapConf) {
	// 环境判断 env GIN_MODE=release/debug
	gin.SetMode(env.RunMode)

	// Global middleware
	router.Use(middleware.Metadata())
	router.Use(middleware.AccessLog())
	router.Use(gin.Recovery())

	// unix socket
	if env.IsDockerPlatform() {
		unix.Start(router)
	}

	// 性能分析工具
	if conf.Pprof {
		base.RegisterProf()
	}

	// 就绪探针
	router.GET("/ready", base.ReadyProbe())
}
