package logger

import (
	"log"
	"os"
)

type Logger struct {
	log *log.Logger
}

func NewLogger(serviceName string) *Logger {
	file, err := os.OpenFile(serviceName+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}

	return &Logger{
		log: log.New(file, "", log.Ldate|log.Ltime),
	}
}

func (l *Logger) Info(message string) {
	l.log.Printf("[INFO] message=%s\n", message)
}

func (l *Logger) Error(message string) {
	l.log.Printf("[ERROR] message=%s\n", message)
}
