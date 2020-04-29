package logger

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

type Person struct {
	Name    string
	Age     int32
	Address string
}

var sugarLogger *zap.SugaredLogger

func StartZap() {
	_ = os.Mkdir("./logs", os.ModePerm)
	hook := &lumberjack.Logger{
		Filename:   "./logs/server.log", // filePath
		MaxSize:    1024,                // megabytes
		MaxBackups: 10,                  // amounts
		MaxAge:     7,                   // days
		Compress:   false,               // disabled by default
	}
	writer := zapcore.AddSync(hook)
	conf := zap.NewProductionEncoderConfig()
	conf.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(conf), writer, zap.DebugLevel)
	logger := zap.New(core)
	sugarLogger = logger.Sugar()
}
