package zlog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/ksc-base/golib/env"
)

// 对用户暴露的log配置
type LogConfig struct {
	Level  string `yaml:"level"`
	Stdout bool   `yaml:"stdout"`
}

type loggerConfig struct {
	ZapLevel zapcore.Level

	// 以下变量仅对开发环境生效
	Stdout   bool
	Log2File bool
	Path     string
}

// 全局配置 仅限Init函数进行变更
var logConfig = loggerConfig{
	ZapLevel: zapcore.InfoLevel,

	Stdout:   false,
	Log2File: true,
	Path:     "./log",
}

func InitLog(conf LogConfig) *zap.SugaredLogger {
	if err := RegisterZYBJSONEncoder(); err != nil {
		panic(err)
	}

	logConfig.ZapLevel = getLogLevel(conf.Level)
	if env.IsDockerPlatform() {
		// 容器环境
		logConfig.Log2File = false
		logConfig.Stdout = true
	} else {
		// 开发环境下默认输出到文件，支持自定义是否输出到终端
		logConfig.Log2File = true
		logConfig.Stdout = conf.Stdout
		logConfig.Path = env.GetLogDirPath()
	}
	SugaredLogger = GetLogger()
	return SugaredLogger
}
