package main

import (
	"github.com/Nrich-sunny/crawler/log"
	"go.uber.org/zap/zapcore"
)

func main() {
	plugin, c := log.NewFilePlugin("./log.txt", zapcore.InfoLevel) // 文件名和日志级别现在是写死的，后续放入配置文件
	defer c.Close()
	logger := log.NewLogger(plugin)
	logger.Info("log init end")
}
