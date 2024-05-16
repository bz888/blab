package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	server "github.com/bz888/blab/api"
	"github.com/bz888/blab/speech/output_api"
	"github.com/bz888/blab/speech/speech"
	logger "github.com/bz888/blab/utils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"net/http"
	"os"
	"strings"
	"sync"
)

// ClientRequest Request from client
type ClientRequest struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

// ClientResponse Response to client
type ClientResponse struct {
	ProcessedText string `json:"processedText"`
}

var (
	dev          bool
	debugConsole *tview.TextView
)

var localLogger *logger.DebugLogger

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
					sendAndRenderContents(textView, textArea, app, voiceContent)
					localLogger.Info("Voice recognizer Completed")
				}()
				fmt.Fprintf(textView, "\nVoice Input Enabled\n")

				textArea.SetDisabled(false)
				return event
			}

			go func() {
				sendAndRenderContents(textView, textArea, app, content)
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
	//go app.Run()

	if err := app.SetRoot(mainFlex, true).SetFocus(textArea).Run(); err != nil {
		panic(err)
	}
}

func sendAndRenderContents(textView *tview.TextView, textArea *tview.TextArea, app *tview.Application, content string) {
	fmt.Fprintln(textView, "[red::]You:[-]")
	fmt.Fprintf(textView, "%s\n\n", content)

	clientReq := ClientRequest{Model: "llama3", Text: content}
	localLogger.Info("Input request:", clientReq.Text)
	requestData, err := json.Marshal(clientReq)
	if err != nil {
		localLogger.Error("Failed to serialize request: %s\n\n", err)
		textArea.SetDisabled(false)
		return
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/process_text", bytes.NewBuffer(requestData))
	if err != nil {
		localLogger.Error("Failed to create request: %s\n\n", err)
		textArea.SetDisabled(false)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		localLogger.Error("Failed to send request: %s\n\n", err)
		textArea.SetDisabled(false)
		return
	}
	defer resp.Body.Close()

	fmt.Fprintf(textView, "[green::]Bot:[-]\n")
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024) // Create an initial buffer of size 64 KB
	scanner.Buffer(buf, 512*1024)   // Set the maximum buffer size to 512 KB

	for scanner.Scan() {
		var clientResp ClientResponse
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
