package log

import (
	"fmt"
	"github.com/rivo/tview"
)

type Types int

const (
	Info Types = iota
	Error
	Warning
	Fatal
)

type DebugLogger struct {
	debugView *tview.TextView
	file      string
}

func NewDebugLogger(debugView *tview.TextView, fileName string) *DebugLogger {
	return &DebugLogger{debugView: debugView, file: fileName}
}

func (d *DebugLogger) log(errorType Types, v ...interface{}) {
	if d.debugView != nil {
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
		fmt.Fprintf(d.debugView, format, d.file, message)
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
}
