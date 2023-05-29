package utils

import (
	"io"
	"os"

	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
)

type ILogger struct {
	ProgressLeft     int
	log              *log.Logger
	bar              *progressbar.ProgressBar
	ignoreIncrements bool
	Verbose          bool
	allocationMap    map[string]int
}

var Logger *ILogger

func GetLogger(verbosity bool, description string, logname string) *ILogger {

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Error(err)
	}

	perunLocation := dirname + PERUN_HOME
	if err := os.MkdirAll(perunLocation, os.ModePerm); err != nil {

		log.Error(err)

	}

	if logname == "" {
		logname = "log"
	}
	logFile, err := os.OpenFile(perunLocation+logname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	// defer logFile.Close()
	logrusLogger := log.New()

	if os.Getenv("H_DEBUG") == "TRUE" {
		logrusLogger.SetLevel(log.DebugLevel)
	}

	logrusLogger.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	logger := &ILogger{
		log:           logrusLogger,
		ProgressLeft:  100,
		allocationMap: make(map[string]int),
	}

	if verbosity || os.Getenv("PerunLog") == "verbose" {
		mw := io.MultiWriter(os.Stdout, logFile)
		logrusLogger.SetOutput(mw)
		logger.Verbose = verbosity
	} else {
		logrusLogger.SetOutput(logFile)
		progressbar.OptionSetDescription(description)
		logger.bar = progressbar.Default(100)

	}

	return logger

}

func (l *ILogger) Warn(format string, args ...interface{}) {
	l.log.Warningf(format, args...)
}

func (l *ILogger) Info(format string, args ...interface{}) {
	l.log.Infof(format, args...)
}

func (l *ILogger) Debug(format string, args ...interface{}) {
	l.log.Debugf(format, args...)
}

func (l *ILogger) Error(format string, args ...interface{}) {
	l.log.Errorf(format, args...)
}

func (l *ILogger) Fatal(format string, args ...interface{}) {
	l.log.Fatalf(format, args...)
}

func (l *ILogger) Increment(increment int, description string) {
	if l.ignoreIncrements || l.Verbose {
		return
	}
	if description != "" {
		l.bar.Describe(description)
	}
	l.ProgressLeft -= increment
	l.bar.Add(increment)
}

func (l *ILogger) Finish() {
	if l.Verbose {
		return
	}
	l.bar.Add(l.ProgressLeft)
	l.ProgressLeft = 0
	l.bar.Finish()
}

func (l *ILogger) IgnoreIncrements(ignore bool) {
	l.ignoreIncrements = ignore
}

func (l *ILogger) GetOutput() io.Writer {
	return l.log.Out
}

func (l *ILogger) GetProgressAllocation(allocationNaame string) int {
	if allocationNaame == "" {
		return l.ProgressLeft
	}
	return l.allocationMap[allocationNaame]
}

func (l *ILogger) SetProgressAllocation(allocationNaame string, allocation int) error {
	if allocationNaame == "" {
		return nil
	}
	l.allocationMap[allocationNaame] = allocation
	return nil
}
