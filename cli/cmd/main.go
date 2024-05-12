package main

import (
	"flag"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
)

var (
	debug bool
)

func main() {

	flag.BoolVar(&debug, "debug", false, "enable debug output")
	flag.Parse()

	//go api.StartServer(debug)
	// Start the server in a goroutine to allow asynchronous execution
	app := tview.NewApplication()

	textArea := tview.NewTextArea()
	textArea.SetTitle("Question").SetBorder(true)

	textView := tview.NewTextView().
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)
	textView.SetTitle("Conversation").SetBorder(true)
	textView.SetScrollable(true).ScrollToEnd()
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			app.SetFocus(textArea)
		}
		return event
	})

	// Create a Flex layout to place the chat view and input field vertically
	subFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, false).
		AddItem(textArea, 8, 2, true)
	mainFlex := tview.NewFlex().
		AddItem(subFlex, 0, 2, false)

	if debug {
		debugConsole := tview.NewTextView().
			SetChangedFunc(func() {
				app.Draw()
			}).
			SetDynamicColors(true).
			SetRegions(true).
			SetWordWrap(true)

		debugConsole.SetTitle("Debugger").SetBorder(true)
		debugConsole.SetScrollable(true).ScrollToEnd()
		mainFlex.
			AddItem(debugConsole, 0, 1, false)
	}

	// Set up the application

	textArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyESC:
			if textView.GetText(false) != "" {
				app.SetFocus(textView)
			}
		case tcell.KeyEnter:
			content := textArea.GetText()
			if strings.TrimSpace(content) == "" {
				return nil
			}
			textArea.SetText("", false)
			textArea.SetDisabled(true)

			fmt.Fprintln(textView, "[red::]You:[-]")
			fmt.Fprintf(textView, "%s\n\n", content)

			mockResponse := "Yes, very hello world"
			fmt.Fprintf(textView, "[green]Bot: %s\n\n", mockResponse)

			fmt.Fprintf(textView, "\n\n")
			textArea.SetDisabled(false)

			return event
			//default:
			//	panic("unhandled default case text area")
		}
		return event
	})
	if err := app.SetRoot(mainFlex, true).SetFocus(textArea).Run(); err != nil {
		panic(err)
	}
}
