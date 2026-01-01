package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var once sync.Once
var defaultLogger *log.Logger
var defaultLoggerFile *os.File

func GetLogger() *log.Logger {
	once.Do(func() {
		defaultLogger, defaultLoggerFile = initLogger()
	})
	return defaultLogger
}

func CloseLogger() {
	if defaultLoggerFile != nil {
		err := defaultLoggerFile.Close()
		if err != nil {
			panic(err)
		}
	}
}

func initLogger() (*log.Logger, *os.File) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	logDir := filepath.Join(homeDir, ".alter", "logs")
	err = os.MkdirAll(logDir, 0755)
	if err != nil {
		panic(err)
	}

	logFileName := filepath.Join(logDir, fmt.Sprintf("alter-%s.log", time.Now().Format("20060102-150405")))
	defaultloggerFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	mw := io.MultiWriter(os.Stdout, defaultloggerFile)
	defaultLogger = log.New(mw, "", log.LstdFlags)
	return defaultLogger, defaultloggerFile
}
