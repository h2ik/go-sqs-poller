package worker

import (
	"fmt"
	"log"
)

// LoggerIFace interface
type LoggerIFace interface {
	Debug(i ...interface{})
	Info(i ...interface{})
	Error(i ...interface{})
}

type logger struct {
}

func (l *logger) Debug(i ...interface{}) {
	log.Printf("[DEBUG] %s", fmt.Sprintln(i...))
}

func (l *logger) Info(i ...interface{}) {
	log.Printf("[INFO] %s", fmt.Sprintln(i...))
}

func (l *logger) Error(i ...interface{}) {
	log.Printf("[ERROR] %s", fmt.Sprintln(i...))
}
