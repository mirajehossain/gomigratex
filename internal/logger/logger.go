package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type Logger struct {
	json bool
}

func New(jsonOutput bool) *Logger {
	l := &Logger{json: jsonOutput}
	log.SetFlags(0)
	return l
}

func (l *Logger) log(level string, msg string, fields map[string]any) {
	if !l.json {
		if len(fields) > 0 {
			b, _ := json.Marshal(fields)
			fmt.Printf("[%s] %s %s\n", level, msg, string(b))
		} else {
			fmt.Printf("[%s] %s\n", level, msg)
		}
		return
	}
	payload := map[string]any{
		"ts":    time.Now().UTC().Format(time.RFC3339Nano),
		"level": level,
		"msg":   msg,
	}
	for k, v := range fields {
		payload[k] = v
	}
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(payload)
}

func (l *Logger) Info(msg string, fields map[string]any)  { l.log("INFO", msg, fields) }
func (l *Logger) Warn(msg string, fields map[string]any)  { l.log("WARN", msg, fields) }
func (l *Logger) Error(msg string, fields map[string]any) { l.log("ERROR", msg, fields) }

// JSONEnabled reports whether this logger is configured to emit JSON output.
func (l *Logger) JSONEnabled() bool { return l.json }
