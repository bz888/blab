package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rivo/tview"
)

type Types int

const (
	Info Types = iota
	Error
	Warn
	Fatal
)

type Message struct {
	Timestamp time.Time
	Tag       string
	Message   string
	LogTypes  Types
}

type Logger struct {
	view      *tview.TextView
	tag       string
	dev       bool
	logFile   *os.File
	logChan   chan Message
	closeChan chan struct{}
}

var (
	logManager *Logger
	once       sync.Once
)

func InitLogger(dev bool, logPath string, view *tview.TextView) {
	once.Do(func() {
		logManager = &Logger{
			view:      view,
			dev:       dev,
			logChan:   make(chan Message, 100),
			closeChan: make(chan struct{}),
		}
		if logPath != "" {
			timestamp := time.Now().Format("20060102_150405")
			fileName := fmt.Sprintf("blab_log_%s.log", timestamp)
			filePath := filepath.Join(logPath, fileName)

			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Fatalf("Failed to open log file: %s", err)
			}
			logManager.logFile = file
		}

		go logManager.processLogs()
	})
}

func NewLogger(tag string) *Logger {
	return &Logger{
		view:      logManager.view,
		tag:       tag,
		dev:       logManager.dev,
		logFile:   logManager.logFile,
		logChan:   logManager.logChan,
		closeChan: logManager.closeChan,
	}
}

func (l *Logger) processLogs() {
	for msg := range l.logChan {
		timestamp := msg.Timestamp.Format("2006-01-02 15:04:05")
		logMessage := fmt.Sprintf("%s [%s] %s: %s\n", timestamp, msg.Tag, msg.LogTypes.toString(), msg.Message)
		if l.logFile != nil {
			l.logFile.WriteString(logMessage)
		}
	}
}

func (l *Logger) log(logTypes Types, v ...interface{}) {
	message := fmt.Sprint(v...)
	if l.dev {
		if l.view != nil {
			var format string
			switch logTypes {
			case Info:
				format = "[green]DEBUG (%s): %s[-]\n"
			case Error:
				format = "[red]DEBUG (%s): %s[-]\n"
			case Warn:
				format = "[yellow]DEBUG (%s): %s[-]\n"
			case Fatal:
				format = "[red]DEBUG (%s): %s[-]\n"
			}
			fmt.Fprintf(l.view, format, l.tag, message)
		} else {
			switch logTypes {
			case Fatal:
				log.Fatal(v...)
			default:
				log.Println(v...)
			}
		}
	}

	if l.logFile != nil {
		logMessage := Message{
			Timestamp: time.Now(),
			Tag:       l.tag,
			Message:   message,
			LogTypes:  logTypes,
		}
		l.logChan <- logMessage
	}
}

func (l *Logger) Info(v ...interface{}) {
	l.log(Info, v...)
}

func (l *Logger) Error(v ...interface{}) {
	l.log(Error, v...)
}

func (l *Logger) Warn(v ...interface{}) {
	l.log(Warn, v...)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.log(Fatal, v...)
	os.Exit(1)
}

func (l *Logger) Close() {
	close(l.closeChan)
	if l.logFile != nil {
		l.logFile.Close()
	}
}

func (t Types) toString() string {
	switch t {
	case Info:
		return "INFO"
	case Error:
		return "ERROR"
	case Warn:
		return "WARN"
	case Fatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
