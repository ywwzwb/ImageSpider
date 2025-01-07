package util

import (
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"ywwzwb/imagespider/models/config"

	llumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// Logger is global logger object
var logLevel = new(slog.LevelVar)

func InitLogger(loggerConfig config.LoggerConfig) {
	logFileWriter := &llumberjack.Logger{
		Filename:   loggerConfig.File.Path,
		MaxSize:    loggerConfig.File.MaxLogFileSize, // megabytes
		MaxBackups: loggerConfig.File.MaxLogFileCount,
		MaxAge:     1, //days
		LocalTime:  true,
	}
	logWriter := io.MultiWriter(logFileWriter)
	if loggerConfig.Console {
		logWriter = io.MultiWriter(logWriter, os.Stdout)
	}
	handler := slog.NewJSONHandler(logWriter, &slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
	})
	logLevel.Set(loggerConfig.Level)
	l := slog.New(handler)
	slog.SetDefault(l)
	slog.Info("Logger is initialized", "config", loggerConfig)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for {
			<-c
			logFileWriter.Rotate()
		}
	}()
}
func SetLogLevel(level slog.Level) {
	slog.Info("SetLogLevel", "level", level)
	logLevel.Set((level))
}
