package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type logRecord struct {
	Level string      `json:"level"`
	Msg   string      `json:"msg"`
	Ts    time.Time   `json:"ts"`
	KVs   interface{} `json:"kvs,omitempty"`
}

type Logger struct {
	std *log.Logger
}

func NewLogger() *Logger {
	return &Logger{std: log.New(os.Stdout, "", 0)}
}

func (l *Logger) log(level, msg string, kv any) {
	rec := logRecord{Level: level, Msg: msg, Ts: time.Now().UTC(), KVs: kv}
	b, _ := json.Marshal(rec)
	l.std.Println(string(b))
}

func (l *Logger) Info(msg string, kv any)  { l.log("INFO", msg, kv) }
func (l *Logger) Warn(msg string, kv any)  { l.log("WARN", msg, kv) }
func (l *Logger) Error(msg string, kv any) { l.log("ERROR", msg, kv) }
