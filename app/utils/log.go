package log

import (
	"fmt"
	"github.com/rivo/tview"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Types int

const (
	Info Types = iota
	Error
	Warning
	Fatal
)

type DebugLogger struct {
	view      *tview.TextView
	tag       string
	dev       bool
	logFile   *os.File
	logChan   chan Message
	closeChan chan struct{}
	wg        sync.WaitGroup
}

type Message struct {
	Timestamp time.Time
	Tag       string
	Message   string
	LogTypes  Types
}

type Manager struct {
	file     *os.File
	logChan  chan Message
	closeSig chan struct{}
	wg       sync.WaitGroup
}

var logManager *Manager

func initLogManager(logPath string) {
	if logManager == nil {
		logManager = &Manager{
			logChan:  make(chan Message, 100),
			closeSig: make(chan struct{}),
		}
		timestamp := time.Now().Format("20060102_150405")
		fileName := fmt.Sprintf("blab_log_%s.log", timestamp)
		filePath := filepath.Join(logPath, fileName)

		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %s", err)
		}
		logManager.file = file

		logManager.wg.Add(1)
		go logManager.processLogs()
	}
}

func (lm *Manager) processLogs() {
	defer lm.wg.Done()
	for {
		select {
		case msg := <-lm.logChan:
			timestamp := msg.Timestamp.Format("2006-01-02 15:04:05")
			logMessage := fmt.Sprintf("%s [%s] %s: %s\n", timestamp, msg.Tag, msg.LogTypes.toString(), msg.Message)
			lm.file.WriteString(logMessage)
		case <-lm.closeSig:
			return
		}
	}
}

func (lm *Manager) close() {
	close(lm.closeSig)
	lm.wg.Wait()
	if lm.file != nil {
		lm.file.Close()
	}
}

// TODO improve logging
// https://github.com/microsoft/onnxruntime/blob/47a178b518cb8929b308959174b16fca2bef2cb5/onnxruntime/core/graph/graph.cc#L4138
// this is crowding the logs, need to find better ways to process debug logs

func NewLogger(debugView *tview.TextView, dev bool, tag string, logPath string) *DebugLogger {
	if logPath != "" && logManager == nil {
		initLogManager(logPath)
	}

	logger := &DebugLogger{
		view:      debugView,
		tag:       tag,
		dev:       dev,
		logFile:   logManager.file,
		logChan:   logManager.logChan,
		closeChan: make(chan struct{}),
	}

	return logger
}

func (d *DebugLogger) log(logTypes Types, v ...interface{}) {
	message := fmt.Sprint(v...)
	if d.dev {
		if d.view != nil {
			var format string
			switch logTypes {
			case Info:
				format = "[green]DEBUG (%s): %s[-]\n"
			case Error:
				format = "[red]DEBUG (%s): %s[-]\n"
			case Warning:
				format = "[yellow]DEBUG (%s): %s[-]\n"
			case Fatal:
				format = "[red]DEBUG (%s): %s[-]\n"
			}
			fmt.Fprintf(d.view, format, d.tag, message)
		} else {
			switch logTypes {
			case Fatal:
				log.Fatal(v...)
			default:
				log.Println(v...)
			}
		}
	}

	if d.logFile != nil {
		logMessage := Message{
			Timestamp: time.Now(),
			Tag:       d.tag,
			Message:   message,
			LogTypes:  logTypes,
		}
		d.logChan <- logMessage
	}

}

func (d *DebugLogger) Info(v ...interface{}) {
	d.log(Info, v...)
}

func (d *DebugLogger) Error(v ...interface{}) {
	d.log(Error, v...)
}

func (d *DebugLogger) Warning(v ...interface{}) {
	d.log(Warning, v...)
}
func (d *DebugLogger) Fatal(v ...interface{}) {
	d.log(Fatal, v...)
	os.Exit(1)
}

func (d *DebugLogger) Close() {
	close(d.closeChan)
	d.wg.Wait()
}

func CloseLogManager() {
	if logManager != nil {
		logManager.close()
	}
}

func (t Types) toString() string {
	switch t {
	case Info:
		return "INFO"
	case Error:
		return "ERROR"
	case Warning:
		return "WARNING"
	case Fatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
