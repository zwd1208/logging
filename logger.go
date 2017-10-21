package logging

import (
	"fmt"
	"os"
)

const (
	_        = iota
	KB int64 = 1 << (iota * 10)
	MB
	GB
	TB
)

type LEVEL byte

const (
	DEBUG LEVEL = iota
	INFO
	WARNING
	ERROR
	FATAL
)

func (level LEVEL) String() string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	logLevel LEVEL
	handlers []Handler
}

func (l *Logger) SetLevel(level LEVEL) {
	if l.logLevel < FATAL {
		l.logLevel = level
	}
}

func (l *Logger) AddHandler(h Handler) error {
	for _, handler := range l.handlers {
		if handler.Name() == h.Name() {
			return fmt.Errorf("handler %s already exists.", h.Name())
		}
	}
	l.handlers = append(l.handlers, h)
	h.Run()
	return nil
}

func (l *Logger) Close() {
	for _, handler := range l.handlers {
		handler.Close()
	}
	l.handlers = nil
}

func (l Logger) log(level LEVEL, format string, v ...interface{}) {
	if l.logLevel <= level {
		for _, handler := range l.handlers {
			if !handler.IsOff() {
				f := fmt.Sprintf("%s %s\n", level.String(), format)
				handler.Log(f, v...)
			}
		}
	}
}

func (l Logger) Debug(format string, v ...interface{}) {
	l.log(DEBUG, format, v...)
}

func (l Logger) Info(format string, v ...interface{}) {
	l.log(INFO, format, v...)
}

func (l Logger) Warning(format string, v ...interface{}) {
	l.log(WARNING, format, v...)
}

func (l Logger) Error(format string, v ...interface{}) {
	l.log(ERROR, format, v...)
}

func (l Logger) Fatal(format string, v ...interface{}) {
	l.log(FATAL, format, v...)
	os.Exit(-1)
}

func NewLogger() *Logger {
	return &Logger{
		logLevel: INFO,
	}
}

func NewStdLogger() *Logger {
	stdhandler, _ := NewStdHandler()
	logger := NewLogger()
	logger.AddHandler(stdhandler)
	return logger
}

func NewSRFileLogger(filePath string, fileCount int, fileSize int64) (*Logger, error) {
	srfilehandler, err := NewSizeRotatingFileHandler("SizeRotatingFileHandler", filePath, fileCount, fileSize)
	if err != nil {
		return nil, err
	}
	logger := NewLogger()
	logger.AddHandler(srfilehandler)
	return logger, nil
}
