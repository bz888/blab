package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	server "github.com/bz888/blab/api"
	"github.com/bz888/blab/speech/speech"
	logger "github.com/bz888/blab/utils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Client struct {
	base *url.URL
	http *http.Client
}

// ClientRequest Request from client
type ClientRequest struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

// APIRequest Request to external API
type APIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type APIResponse struct {
	Model              string  `json:"model"`
	CreatedAt          string  `json:"created_at"`
	Message            Message `json:"message"`
	Done               bool    `json:"done"`
	TotalDuration      int64   `json:"total_duration"`
	LoadDuration       int64   `json:"load_duration"`
	PromptEvalCount    int     `json:"prompt_eval_count"`
	PromptEvalDuration int64   `json:"prompt_eval_duration"`
	EvalCount          int     `json:"eval_count"`
	EvalDuration       int64   `json:"eval_duration"`
}

// ClientResponse Response to client
type ClientResponse struct {
	ProcessedText string `json:"processedText"`
}

var (
	debug bool
)

func init() {
	flag.BoolVar(&debug, "debug", false, "enable debug output")
	flag.Parse()
}

func main() {
	app := tview.NewApplication()

	debugConsole := tview.NewTextView().
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	localLogger := logger.NewDebugLogger(debugConsole, "main")

	go server.Run()

	// Start the server in a goroutine to allow asynchronous execution
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

	//Enable mouse to have mouse scrolling working. We don't need SetScrollable because it is 'true' by default
	app.EnableMouse(true)

	textView.SetTitle("Conversation").SetBorder(true)
	// textView.SetScrollable(true)
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

	if debug {
		mainFlex.
			AddItem(debugConsole, 0, 1, false)
		debugConsole.SetTitle("Debugger").SetBorder(true)
		debugConsole.ScrollToEnd()
	}

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

				func() {
					localLogger.Info("Exiting by command.")
					localLogger.Info("Shutting down gracefully.")
					app.Stop()
					os.Exit(0)
				}()

				return nil
			case "/debug":
				go func() {
					if debugConsole == nil {
						debug = true

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

							localLogger = logger.NewDebugLogger(debugConsole, "main")
							fmt.Fprintf(textView, "Debug console enabled\n")
						})
					} else {
						app.QueueUpdateDraw(func() {
							mainFlex.RemoveItem(debugConsole)
							debugConsole = nil
							fmt.Fprintf(textView, "Debug console disabled\n")
						})

						debug = false
					}
				}()

				defer textArea.SetDisabled(false)
				return event
			case "/voice":
				// ping google speech api v2
				localLogger.Info("Voice recogniser Started")
				var voiceContent string
				var err error
				//voiceContent, err := speech.Run()
				//if err != nil {
				//	log.Fatal("Failed on voice recogniser", err)
				//}

				//speech.PrintAvailableDevices(debugConsole)
				// Channel to signal completion of the goroutine

				go func() {
					voiceContent, err = speech.Run(debugConsole)
					if err != nil {
						localLogger.Error("Failed to process voice")
					}
				}()

				localLogger.Info("Voice recognizer Completed")
				content = voiceContent
				return event
			}

			go func() {
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
				textArea.SetDisabled(false)
			}()

			return event
		}
		return event
	})

	if err := app.SetRoot(mainFlex, true).SetFocus(textArea).Run(); err != nil {
		panic(err)
	}
}
