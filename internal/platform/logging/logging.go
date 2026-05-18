package logging

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type Logger struct {
	service string
	logger  *log.Logger
}

func New(service string) *Logger {
	return &Logger{
		service: service,
		logger:  log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) Info(message string, fields map[string]any) {
	l.write("info", message, fields)
}

func (l *Logger) Error(message string, err error, fields map[string]any) {
	if fields == nil {
		fields = map[string]any{}
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	l.write("error", message, fields)
}

func (l *Logger) write(level, message string, fields map[string]any) {
	if fields == nil {
		fields = map[string]any{}
	}
	fields["level"] = level
	fields["message"] = message
	fields["service"] = l.service
	fields["time"] = time.Now().UTC().Format(time.RFC3339Nano)
	bytes, err := json.Marshal(fields)
	if err != nil {
		l.logger.Printf(`{"level":"error","message":"log marshal failed","service":"%s"}`, l.service)
		return
	}
	l.logger.Print(string(bytes))
}
