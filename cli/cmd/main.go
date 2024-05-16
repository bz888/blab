package main

import (
	"flag"
	"fmt"
	server "github.com/bz888/blab/api"
	"github.com/bz888/blab/speech/output_api"
	"github.com/bz888/blab/speech/speech"
	logger "github.com/bz888/blab/utils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
	"strings"
	"sync"
)

var (
	dev          bool
	debugConsole *tview.TextView
)

var localLogger *logger.DebugLogger
var defaultModel = "llama3:latest" // default

// if dev is true, then should init on new window, so logging can be seen in terminal
func init() {
	flag.BoolVar(&dev, "dev", false, "Development mode")
	flag.Parse()
}

//	func main() {
//		speech.InitService(debugConsole, dev)
//		speech.Run()
//	}
func main() {
	localLogger = logger.NewLogger(debugConsole, dev, "main")

	currentModel := defaultModel

	app := tview.NewApplication()
	app.EnablePaste(true)

	textArea := tview.NewTextArea()
	textArea.SetTitle("Question").SetBorder(true)

	textView := tview.NewTextView().
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	app.EnableMouse(true)

	textView.SetTitle("Conversation").SetBorder(true)
	textView.SetScrollable(true)
	textView.ScrollToEnd()
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

	if dev {
		debugConsole = tview.NewTextView().
			SetChangedFunc(func() {
				app.Draw()
			}).
			SetDynamicColors(true).
			SetRegions(true).
			SetWordWrap(true)

		debugConsole.SetTitle("Debugger").SetBorder(true)
		debugConsole.ScrollToEnd()

		localLogger = logger.NewLogger(debugConsole, dev, "main")
		mainFlex.AddItem(debugConsole, 0, 1, true)
	}

	textArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		var wg sync.WaitGroup

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
			textArea.SetText("", true)
			textArea.SetDisabled(true)
			switch strings.TrimSpace(content) {
			case "/help":
				fmt.Fprintln(textView, "[red::]You:[-]")
				fmt.Fprintf(textView, "%s\n\n", content)

				fmt.Fprintf(textView, "[green::]Bot:[-]\n")
				fmt.Fprintf(textView, "Here are some commands you can use:\n")
				fmt.Fprintf(textView, "- /help: Display this help message\n")
				fmt.Fprintf(textView, "- /bye: Exit the application\n")
				fmt.Fprintf(textView, "- /debug: Toggle the debug console\n")
				fmt.Fprintf(textView, "- /voice: Activate voice commands\n\n")
				textArea.SetDisabled(false)
				return event

				textArea.SetDisabled(false)
				return event
			case "/bye":
				fmt.Fprintf(textView, "Bye bye\n")

				go func() {
					localLogger.Info("Exiting by command.")
					localLogger.Info("Shutting down gracefully.")
					app.Stop()
					os.Exit(0)
				}()

				return nil
			case "/debug":
				go func() {
					// todo should be based on if the item is apart of the mainFlex
					if debugConsole == nil {
						app.QueueUpdateDraw(func() {
							debugConsole = tview.NewTextView().
								SetChangedFunc(func() {
									app.Draw()
								}).
								SetDynamicColors(true).
								SetRegions(true).
								SetWordWrap(true)

							debugConsole.SetTitle("Debugger").SetBorder(true)
							debugConsole.ScrollToEnd()

							mainFlex.AddItem(debugConsole, 0, 1, true) // Adjust size as needed

							localLogger = logger.NewLogger(debugConsole, dev, "main")
							fmt.Fprintf(textView, "\nDebug console enabled\n")
						})
					} else {
						// toggle does not work, as it changes the address ?
						app.QueueUpdateDraw(func() {
							mainFlex.RemoveItem(debugConsole)

							fmt.Fprintf(textView, "\nDebug console disabled\n")
						})
					}
				}()

				defer textArea.SetDisabled(false)
				return event
			case "/voice":
				// ping google speech api v2
				localLogger.Info("Voice recogniser Started")
				var voiceContent string
				var err error

				wg.Add(1)
				go func() {
					defer wg.Done() // Ensure Done is called on completion
					voiceContent, err = speech.Run()
					if err != nil {
						localLogger.Error("Failed to process voice")
					}
					app.Draw()
				}()
				localLogger.Info("Voice to api")

				go func() {
					wg.Wait()
					server.Chatting(textView, textArea, app, currentModel, voiceContent)
					localLogger.Info("Voice recognizer Completed")
				}()
				fmt.Fprintf(textView, "\nVoice Input Enabled\n")

				textArea.SetDisabled(false)
				return event
			case "/models":
				go func() {
					var modelList []server.Model
					models, err := server.ListModels()
					if err != nil {
					}
					modelList = models

					createModal := func(p tview.Primitive, width, height int) tview.Primitive {
						return tview.NewFlex().
							AddItem(nil, 0, 1, false).
							AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
								AddItem(nil, 0, 1, false).
								AddItem(p, height, 1, true).
								AddItem(nil, 0, 1, false), width, 1, true).
							AddItem(nil, 0, 1, false)
					}

					var pages *tview.Pages

					list := tview.NewList()
					list.SetBorder(true)

					for i, model := range modelList {
						runeValue := '0' + rune(i)

						if model.Name == currentModel {
							list.AddItem(model.Name, "Current LLM", runeValue, func() {
								localLogger.Info("This model is currently in use", model.Name)
								fmt.Fprintf(textView, "\nAlready using model: %s\n\n", model.Name)
							})
						} else {
							list.AddItem(model.Name, "LLM", runeValue, func() {
								localLogger.Info("Selected: ", model.Name)
								currentModel = model.Name
								fmt.Fprintf(textView, "\nUsing Model: %s\n\n", model.Name)

								pages.RemovePage("modelModal")
								textArea.SetDisabled(false)
								app.SetFocus(textArea)
								return
							})
						}
					}

					modal := createModal(list, 40, 10)

					list.
						AddItem("Back", "", 'q', func() {
							pages.RemovePage("modelModal")
							textArea.SetDisabled(false)
							app.SetFocus(textArea)
							return
						})

					pages = tview.NewPages().
						AddPage("main", mainFlex, true, true).
						AddPage("modelModal", modal, true, true)

					if err := app.SetRoot(pages, true).Run(); err != nil {
						panic(err)
					}
				}()

				// todo double check if this is the correct execution, i feel like it can introduce leakage
				localLogger.Info("/models command executed and completed")
				return event
			}

			go func() {
				server.Chatting(textView, textArea, app, currentModel, content)
				textArea.SetDisabled(false)
			}()

			return event
		}
		return event
	})

	// init services
	server.InitService(debugConsole, dev)
	output_api.InitService(debugConsole, dev)
	speech.InitService(debugConsole, dev)

	go server.Run()

	if err := app.SetRoot(mainFlex, true).SetFocus(textArea).Run(); err != nil {
		panic(err)
	}
}
