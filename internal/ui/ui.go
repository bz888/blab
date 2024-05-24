package ui

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bz888/blab/internal/api"
	"github.com/bz888/blab/internal/api/server"
	"github.com/bz888/blab/internal/config"
	"github.com/bz888/blab/internal/logger"
	"github.com/bz888/blab/internal/speech"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"log"
	"os"
	"strings"
	"sync"
)

var app *tview.Application
var defaultModel = "llama3:latest" // default
var wg sync.WaitGroup

var (
	debugConsole *tview.TextView
	textView     *tview.TextView
	textArea     *tview.TextArea
	localLogger  *logger.Logger
)

func Init() {
	app = tview.NewApplication()
	app.EnablePaste(true)
	app.EnableMouse(true)

	debugConsole = initDebugConsole()

	textView = initChatViewer()
	textArea = initChatInput()
}

func initChatViewer() *tview.TextView {
	textView := tview.NewTextView().
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	textView.SetTitle("Conversation").SetBorder(true)
	textView.SetScrollable(true)
	textView.ScrollToEnd()
	textView.SetWordWrap(true)
	return textView
}

func initChatInput() *tview.TextArea {
	textArea := tview.NewTextArea()
	textArea.SetTitle("Question").SetBorder(true)
	return textArea
}

func initDebugConsole() *tview.TextView {
	console := tview.NewTextView().
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	console.SetTitle("Debugger").SetBorder(true)
	console.ScrollToEnd()
	return console
}

// Run InitUi logPath and dev should be set to a ()
func Run() {
	currentModel := defaultModel
	localLogger = logger.NewLogger("views")

	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			app.SetFocus(textArea)
		}
		return event
	})

	subFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, false).
		AddItem(textArea, 8, 2, true)
	mainFlex := tview.NewFlex().
		AddItem(subFlex, 0, 2, false)

	if config.Dev {
		mainFlex.AddItem(debugConsole, 0, 1, true)
	}

	// setup input capture logic
	setInputCapture(mainFlex, currentModel)

	if err := app.SetRoot(mainFlex, true).SetFocus(textArea).Run(); err != nil {
		panic(err)
	}
}

func setInputCapture(mainFlex *tview.Flex, currentModel string) {
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
			textArea.SetText("", true)
			textArea.SetDisabled(true)

			switch strings.TrimSpace(content) {

			// todo refactor into a const object of all commands and followed by the running function
			case "/help":
				listHelp(content)
				textArea.SetDisabled(false)
				return event
			case "/bye": // todo, /quit /exit should all work the same
				quitApp()
				return event
			case "/debug":
				toggleDebugConsole(mainFlex)
				textArea.SetDisabled(false)
				return event
			case "/voice":
				voiceRecognition(currentModel)
				return event
			case "/models":
				go func() {
					createModelModal(currentModel, mainFlex)
					textArea.SetDisabled(false)
				}()
				return event
			}

			go func() {
				api.Chatting(currentModel, content, app, textView)
				textArea.SetDisabled(false)
			}()
		}
		return event
	})
}

func voiceRecognition(currentModel string) {

	if os.Getenv("API_KEY") == "" {
		fmt.Fprintf(textView, "\nAPI_KEY is required to enable voice recognistion\n")
		localLogger.Warn("API_KEY is not set, voice recognition is disabled")
		textArea.SetDisabled(false)
		return
	}

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
		app.Draw()
		wg.Wait()
		api.Chatting(currentModel, voiceContent, app, textView)
		localLogger.Info("Voice recognizer Completed")
		textArea.SetDisabled(false)
	}()
}

func createModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

func createModelModal(currentModel string, mainFlex *tview.Flex) {
	models, err := api.ListModels()
	if err != nil {
	}

	var pages *tview.Pages
	list := tview.NewList()
	list.SetBorder(true)
	for i, model := range models {
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
	localLogger.Info("/models command executed and completed")
}

func toggleDebugConsole(mainFlex *tview.Flex) {
	go func() {
		// todo should be based on if the item is apart of the mainFlex
		if !config.Dev {
			app.QueueUpdateDraw(func() {
				mainFlex.AddItem(debugConsole, 0, 1, true) // Adjust size as needed
				fmt.Fprintf(textView, "\nDebug console enabled\n")
			})
		} else {
			app.QueueUpdateDraw(func() {
				mainFlex.RemoveItem(debugConsole)
				fmt.Fprintf(textView, "\nDebug console disabled\n")
			})
		}
	}()
}

func quitApp() {
	fmt.Fprintf(textView, "Bye bye\n")

	wg.Add(1)
	go func() {
		defer wg.Done()
		localLogger.Close()
		app.Stop()
		log.Println("Shutting down gracefully.")
	}()

	wg.Wait()
	os.Exit(0)
}

func listHelp(content string) {
	fmt.Fprintln(textView, "[red::]You:[-]")
	fmt.Fprintf(textView, "%s\n\n", content)

	// all commands
	fmt.Fprintf(textView, "[green::]Bot:[-]\n")
	fmt.Fprintf(textView, "Here are some commands you can use:\n")
	fmt.Fprintf(textView, "- /help: Display this help message\n")
	fmt.Fprintf(textView, "- /bye: Exit the application\n")
	fmt.Fprintf(textView, "- /debug: Toggle the debug console\n")
	fmt.Fprintf(textView, "- /voice: Activate voice input\n\n")
	fmt.Fprintf(textView, "- /models: Select between local LLM\n\n")
}

func GetDebugConsole() (*tview.TextView, error) {
	if debugConsole == nil {
		return nil, errors.New("debug console not initialized")
	}
	return debugConsole, nil
}

func streamChat(scanner *bufio.Scanner) {
	for scanner.Scan() {
		var clientResp server.ClientResponse
		err := json.Unmarshal(scanner.Bytes(), &clientResp)
		if err != nil {
			localLogger.Error("Failed to decode response: %s\n\n", err)
			continue
		}
		app.QueueUpdateDraw(func() {
			fmt.Fprintf(textView, "%s", clientResp.ProcessedText)
		})
	}
	if err := scanner.Err(); err != nil {
		localLogger.Error("Failed to read stream: %s\n\n", err)
	}
}
