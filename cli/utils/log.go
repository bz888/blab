package log

import (
	"fmt"
	"github.com/rivo/tview"
	"log"
	"os"
)

type Types int

const (
	Info Types = iota
	Error
	Warning
	Fatal
)

type DebugLogger struct {
	view *tview.TextView
	file string
	dev  bool
}

// TODO improve logging
// https://github.com/microsoft/onnxruntime/blob/47a178b518cb8929b308959174b16fca2bef2cb5/onnxruntime/core/graph/graph.cc#L4138
// this is crowding the logs, need to find better ways to process debug logs

func NewLogger(debugView *tview.TextView, dev bool, fileName string) *DebugLogger {
	return &DebugLogger{view: debugView, file: fileName, dev: dev}
}

func (d *DebugLogger) log(errorType Types, v ...interface{}) {
	if d.dev {
		if d.view != nil {
			var format string
			switch errorType {
			case Info:
				format = "[green]DEBUG (%s): %s[-]\n"
			case Error:
				format = "[red]DEBUG (%s): %s[-]\n"
			case Warning:
				format = "[yellow]DEBUG (%s): %s[-]\n"
			case Fatal:
				format = "[red]DEBUG (%s): %s[-]\n"
			}
			message := fmt.Sprint(v...)
			fmt.Fprintf(d.view, format, d.file, message)
		} else {
			switch errorType {
			case Fatal:
				log.Fatal(v...)
			default:
				log.Println(v...)
			}
		}
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
