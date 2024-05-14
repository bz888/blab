package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

const sampleRate = 16000
const numChannels = 1
const bitsPerSample = 16

type LogTypes string

const (
	Error   LogTypes = "error"
	Warning LogTypes = "warning"
	Info    LogTypes = "info"
)

var debugConsole *tview.TextView
var port = 8080

func init() {
	flag.BoolVar(&debug, "debug", false, "enable debug output")
	flag.Parse()
}

func main() {
	// Start the server in a goroutine to allow asynchronous execution

	go func() {
		http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			status := struct {
				PortWorking   bool `json:"port_working"`
				ServerWorking bool `json:"server_working"`
			}{
				PortWorking:   true,
				ServerWorking: true,
			}

			err := json.NewEncoder(w).Encode(status)
			if err != nil {
				return
			}
		})

		http.HandleFunc("/process_text", processTextHandler)

		address := ":" + strconv.Itoa(port)
		debugLog("info", debugConsole, "Debug mode is enabled")
		debugLog("info", debugConsole, "Server started on http://localhost"+address+"/")

		// Start the server
		err := http.ListenAndServe(address, nil)
		if err != nil {
			debugLog("error", debugConsole, "Error starting server: ", err)
		}
	}()

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
		debugConsole = tview.NewTextView().
			SetChangedFunc(func() {
				app.Draw()
			}).
			SetDynamicColors(true).
			SetRegions(true).
			SetWordWrap(true)

		debugConsole.SetTitle("Debugger").SetBorder(true)
		debugConsole.ScrollToEnd()
		mainFlex.
			AddItem(debugConsole, 0, 1, false)
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
				debugLog("info", debugConsole, "Exiting by command.")
				shutdown(app)
				return nil
			case "/debug":
				// toggling is no working if it is repeated execution
				go func() {
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
							fmt.Fprintf(textView, "Debug console enabled\n")
						})
					} else {
						app.QueueUpdateDraw(func() {
							mainFlex.RemoveItem(debugConsole)
							debugConsole = nil
							fmt.Fprintf(textView, "Debug console disabled\n")
						})
					}
				}()

				defer textArea.SetDisabled(false)
				return event
			case "/voice":
				// add a voice meter above the input

				// ping google speech api v2
				debugLog("info", debugConsole, "Voice recogniser Started")
				go speech.Run()

				debugLog("info", debugConsole, "Voice recogniser Completed")
				textArea.SetDisabled(false)
				return event
			}
			go func() {
				fmt.Fprintln(textView, "[red::]You:[-]")
				fmt.Fprintf(textView, "%s\n\n", content)

				clientReq := ClientRequest{Model: "llama3", Text: content}
				debugLog("info", debugConsole, "Input request:", clientReq.Text)
				requestData, err := json.Marshal(clientReq)
				if err != nil {
					debugLog("error", debugConsole, "Failed to serialize request: %s\n\n", err)
					textArea.SetDisabled(false)
					return
				}

				req, err := http.NewRequest("POST", "http://localhost:8080/process_text", bytes.NewBuffer(requestData))
				if err != nil {
					debugLog("error", debugConsole, "Failed to create request: %s\n\n", err)
					textArea.SetDisabled(false)
					return
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/x-ndjson")

				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					debugLog("error", debugConsole, "Failed to send request: %s\n\n", err)
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
						debugLog("error", debugConsole, "Failed to decode response: %s\n\n", err)
						continue
					}
					app.QueueUpdateDraw(func() {
						fmt.Fprintf(textView, "%s", clientResp.ProcessedText)
					})
				}
				if err := scanner.Err(); err != nil {
					debugLog("error", debugConsole, "Failed to read stream: %s\n\n", err)
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
func processTextHandler(w http.ResponseWriter, r *http.Request) {
	var clientReq ClientRequest
	err := json.NewDecoder(r.Body).Decode(&clientReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	apiReq := APIRequest{
		Model: clientReq.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: clientReq.Text,
			},
		},
		Stream: true,
	}

	client := &Client{
		base: &url.URL{Scheme: "http", Host: "localhost:11434"},
		http: &http.Client{},
	}

	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")

	encoder := json.NewEncoder(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	err = client.Chat(r.Context(), &apiReq, func(bts []byte) error {
		var apiResp APIResponse
		if err := json.Unmarshal(bts, &apiResp); err != nil {
			return err
		}

		err := encoder.Encode(ClientResponse{ProcessedText: apiResp.Message.Content})

		if !apiResp.Done {
			debugLog("info", debugConsole, "Received response:", apiResp.Message.Content)
		} else {
			debugLog("info", debugConsole, "Completed response", string(bts))
		}

		if err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})

	if err != nil {
		http.Error(w, "Failed to process request: "+err.Error(), http.StatusInternalServerError)
	}
}

func (c *Client) Chat(ctx context.Context, req *APIRequest, fn func([]byte) error) error {
	return c.stream(ctx, http.MethodPost, "/api/chat", req, fn)
}

func (c *Client) stream(ctx context.Context, method string, path string, data any, fn func([]byte) error) error {
	var buf *bytes.Buffer
	if data != nil {
		bts, err := json.Marshal(data)
		if err != nil {
			return err
		}
		buf = bytes.NewBuffer(bts)
	}

	requestURL := c.base.ResolveReference(&url.URL{Path: path})
	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), buf)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/x-ndjson")
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		if err := fn(scanner.Bytes()); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

func shutdown(app *tview.Application) {
	debugLog("info", debugConsole, "Shutting down gracefully.")
	app.Stop()
	os.Exit(0)
}

func debugLog(errorType LogTypes, debugView *tview.TextView, v ...interface{}) {
	if debugConsole != nil {
		switch errorType {
		case Info:
			fmt.Fprintf(debugView, "[green]DEBUG (Info): %v[-]\n", v)
		case Error:
			fmt.Fprintf(debugView, "[red]DEBUG (Error): %v[-]\n", v)
		case Warning:
			fmt.Fprintf(debugView, "[yellow]DEBUG (Warning): %v[-]\n", v)
		}
	}
}
