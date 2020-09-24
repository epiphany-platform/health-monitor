package logger

import (
	"fmt"
	"log"
	"log/syslog"
	"path/filepath"
	"runtime"
)

var (
	sysLog *syslog.Writer
)

// Init establishes a connection to a syslog daemon
func Init() (err error) {
	sysLog, err = syslog.Dial("", "", syslog.LOG_DEBUG|syslog.LOG_DAEMON, "healthd")
	if err != nil {
		log.Fatal(err)
	}
	return
}

// Close closes a connection to the syslog daemon.
func Close() error {
	return sysLog.Close()
}

// Crit logs a message with severity LOG_CRIT
func Crit(m string) error {
	_, fn, line, _ := runtime.Caller(1)
	return sysLog.Crit(fmt.Errorf("%s:%d %v", filepath.Base(fn), line, m).Error())
}

// Alert logs a message with severity LOG_ALERT
func Alert(m string) error {
	_, fn, line, _ := runtime.Caller(1)
	return sysLog.Alert(fmt.Errorf("%s:%d %v", filepath.Base(fn), line, m).Error())
}

// Debug logs a message with severity LOG_DEBUG
func Debug(m string) error {
	_, fn, line, _ := runtime.Caller(1)
	return sysLog.Debug(fmt.Errorf("%s:%d %v", filepath.Base(fn), line, m).Error())
}

// Emerg logs a message with severity LOG_EMERG
func Emerg(m string) error {
	_, fn, line, _ := runtime.Caller(1)
	return sysLog.Emerg(fmt.Errorf("%s:%d %v", filepath.Base(fn), line, m).Error())
}

// Err logs a message with severity LOG_ERR
func Err(m string) error {
	_, fn, line, _ := runtime.Caller(1)
	return sysLog.Err(fmt.Errorf("%s:%d %v", filepath.Base(fn), line, m).Error())
}

// Info logs a message with severity LOG_INFO
func Info(m string) error {
	_, fn, line, _ := runtime.Caller(1)
	return sysLog.Info(fmt.Errorf("%s:%d %v", filepath.Base(fn), line, m).Error())
}

// Warning logs a message with severity LOG_WARNING
func Warning(m string) error {
	_, fn, line, _ := runtime.Caller(1)
	return sysLog.Warning(fmt.Errorf("%s:%d %v", filepath.Base(fn), line, m).Error())
}
