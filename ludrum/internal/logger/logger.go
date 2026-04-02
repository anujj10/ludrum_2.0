package logger

import (
	"encoding/json"
	"fmt"
	"time"
)

type LogLevel string

const (
	INFO  LogLevel = "INFO"
	ERROR LogLevel = "ERROR"
	DEBUG LogLevel = "DEBUG"
)

type LogEntry struct {
	Level     LogLevel              `json:"level"`
	Module    string                `json:"module"`
	Message   string                `json:"message"`
	Error     string                `json:"error,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Timestamp int64                 `json:"timestamp"`
}

func log(level LogLevel, module, message string, err error, fields map[string]interface{}) {
	entry := LogEntry{
		Level:     level,
		Module:    module,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if fields != nil {
		entry.Fields = fields
	}

	data, _ := json.Marshal(entry)
	fmt.Println(string(data))
}

// ==========================
// HELPERS
// ==========================
func Info(module, message string, fields map[string]interface{}) {
	log(INFO, module, message, nil, fields)
}

func Error(module, message string, err error, fields map[string]interface{}) {
	log(ERROR, module, message, err, fields)
}

func Debug(module, message string, fields map[string]interface{}) {
	log(DEBUG, module, message, nil, fields)
}