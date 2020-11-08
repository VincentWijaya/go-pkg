package log

import (
	"context"
	"strings"

	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

type ILogger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	WithField(key string, value interface{}) ILogger
	WithFields(fields map[string]interface{}) ILogger
	WithContext(ctx context.Context) ILogger
}

type LogConfig struct {
	StdoutFile string
	StderrFile string
	Level      string
}

type Logger struct {
	logger      *logrus.Logger
	contextData []string
}

type Entry struct {
	entry       *logrus.Entry
	contextData []string
}

type Fields map[string]interface{}

var logger = logrus.New()
var contextData = []string{}

func InitLogger(env string, conf LogConfig, ctxData []string) {
	var formatter logrus.Formatter
	formatter = &logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	}

	if env == "" || env == "development" || env == "local" {
		formatter = &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		}
	}

	pathMap := lfshook.PathMap{}
	if conf.StdoutFile != "" {
		pathMap[logrus.DebugLevel] = conf.StdoutFile
	}
	if conf.StderrFile != "" {
		pathMap[logrus.InfoLevel] = conf.StdoutFile
		pathMap[logrus.ErrorLevel] = conf.StdoutFile
	}

	rotateFileHook, _ := NewRotateFileHook(RotateFileConfig{
		Filename:   conf.StdoutFile,
		MaxSize:    50,
		MaxBackups: 7,
		MaxAge:     7,
		Level:      logrus.DebugLevel,
		Formatter:  formatter,
	})
	logger.AddHook(rotateFileHook)

	logger.SetFormatter(formatter)
	logger.SetLevel(getLevel(conf.Level))
	if len(pathMap) > 0 {
		logger.Hooks.Add(lfshook.NewHook(
			pathMap,
			formatter,
		))
	}
	contextData = ctxData
}

func getLevel(level string) logrus.Level {
	if level == "error" {
		return logrus.ErrorLevel
	} else if level == "info" {
		return logrus.InfoLevel
	} else if level == "debug" {
		return logrus.DebugLevel
	}
	return logrus.ErrorLevel
}

func getContextValue(ctx context.Context, key string, entry *logrus.Entry) *logrus.Entry {
	value := ctx.Value(key)
	if value != nil {
		fields := logrus.Fields{}
		fields[key] = value

		entry = entry.WithFields(fields)
	}
	return entry
}

// formatFilePath get caller file paths to be displayed in log
func formatFilePath(f string) string {
	paths := strings.Split(f, "/")
	paths = paths[len(paths)-4:]
	return strings.Join(paths, "/")
}

func Debug(args ...interface{}) {
	logger.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}

func WithField(key string, value interface{}) ILogger {
	entry := logger.WithField(key, value)
	return &Entry{entry: entry, contextData: contextData}
}

func WithFields(fields map[string]interface{}) ILogger {
	entry := logrus.NewEntry(logger)
	for k, v := range fields {
		entry = entry.WithField(k, v)
	}
	return &Entry{entry: entry, contextData: contextData}
}

func WithContext(ctx context.Context) ILogger {
	entry := logrus.NewEntry(logger)
	for _, v := range contextData {
		entry = getContextValue(ctx, v, entry)
	}
	return &Entry{entry: entry, contextData: contextData}
}

func (en *Entry) Debug(args ...interface{}) {
	en.entry.Debug(args...)
}

func (en *Entry) Debugf(format string, args ...interface{}) {
	en.entry.Debugf(format, args...)
}

func (en *Entry) Info(args ...interface{}) {
	en.entry.Info(args...)
}

func (en *Entry) Infof(format string, args ...interface{}) {
	en.entry.Infof(format, args...)
}

func (en *Entry) Error(args ...interface{}) {
	en.entry.Error(args...)
}

func (en *Entry) Errorf(format string, args ...interface{}) {
	en.entry.Errorf(format, args...)
}

func (en *Entry) WithField(key string, value interface{}) ILogger {
	entry := en.entry.WithField(key, value)
	en.entry = entry
	return en
}

func (en *Entry) WithFields(fields map[string]interface{}) ILogger {
	entry := en.entry
	for k, v := range fields {
		entry = entry.WithField(k, v)
	}
	en.entry = entry
	return en
}

func (en *Entry) WithContext(ctx context.Context) ILogger {
	entry := en.entry
	for _, v := range en.contextData {
		entry = getContextValue(ctx, v, entry)
	}
	en.entry = entry
	return en
}
