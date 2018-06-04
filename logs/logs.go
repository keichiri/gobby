package logs

import (
	"fmt"
	"log"
)

func Debug(name, format string, args ...interface{}) {
	writeLog("DEBUG", name, format, args...)
}

func Info(name, format string, args ...interface{}) {
	writeLog("INFO", name, format, args...)
}

func Warn(name, format string, args ...interface{}) {
	writeLog("WARNING", name, format, args...)
}

func Error(name, format string, args ...interface{}) {
	writeLog("ERROR", name, format, args...)
}

func Critical(name, format string, args ...interface{}) {
	writeLog("CRITICAL", name, format, args...)
}

func writeLog(level, name, format string, args ...interface{}) {
	log.Printf("[%s] %s: %s\n", level, name, fmt.Sprintf(format, args...))
}
