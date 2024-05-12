package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"io"
	"net/http"
	"strconv"
	"strings"
)

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
		debugLog("info", "Debug mode is enabled")
		debugLog("info", "Server started on http://localhost"+address+"/")

		// Start the server
		err := http.ListenAndServe(address, nil)
		if err != nil {
			debugLog("error", "Error starting server: ", err)
		}
	}()

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

	//Enable mouse to have mouse scrolling working. We dont need SetScrollable because it is 'true' by default
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
			textArea.SetText("", false)
			textArea.SetDisabled(true)

			go func() {
				fmt.Fprintln(textView, "[red::]You:[-]")
				fmt.Fprintf(textView, "%s\n\n", content)

				clientReq := ClientRequest{Model: "llama3", Text: content}
				debugLog("info", clientReq.Text)
				requestData, err := json.Marshal(clientReq)
				if err != nil {
					debugLog("error", "Failed to serialize request: %s\n\n", err)
					textArea.SetDisabled(false)
					return
				}

				resp, err := http.Post("http://localhost:8080/process_text", "application/json", bytes.NewBuffer(requestData))
				if err != nil {
					debugLog("error", "Failed to send request: %s\n\n", err)
					textArea.SetDisabled(false)
					return
				}
				defer resp.Body.Close()

				var clientResp ClientResponse
				if err := json.NewDecoder(resp.Body).Decode(&clientResp); err != nil {
					debugLog("error", "Failed to decode response: %s\n\n", err)
					textArea.SetDisabled(false)
					return
				}

				fmt.Fprintf(textView, "[green::]Bot:[-]\n")
				fmt.Fprintf(textView, "%s\n\n", clientResp.ProcessedText)
				fmt.Fprintf(textView, "\n\n")
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
	debugLog("info", "Processing request:", clientReq)

	if err != nil {
		debugLog("error", "Error decoding client JSON: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(r.Body)

	// Prepare the request for the external API
	apiReq := APIRequest{
		Model: clientReq.Model, // this should be selectable
		Messages: []Message{
			{
				Role:    "user", // TODO update this
				Content: clientReq.Text,
			},
		},
		Stream: false,
	}

	requestData, err := json.Marshal(apiReq)
	debugLog("info", "Sending data to API:", string(requestData))

	if err != nil {
		debugLog("error", "Error marshaling API request JSON: %s", err)
		http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
		return
	}

	// Send the request to the external API
	apiURL := "http://localhost:11434/api/chat"
	apiResp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(requestData))
	if err != nil {
		debugLog("error", "Error calling external API: %s", err)
		http.Error(w, "Error calling external API", http.StatusInternalServerError)
		return
	}
	defer apiResp.Body.Close()

	// Read the response from external API
	var apiResponse APIResponse
	if err := json.NewDecoder(apiResp.Body).Decode(&apiResponse); err != nil {
		debugLog("error", "Error decoding API response JSON: %s", err)
		http.Error(w, "Error decoding API response JSON", http.StatusInternalServerError)
		return
	}
	debugLog("info", "Received response from API:", apiResponse)

	clientResp := ClientResponse{ProcessedText: apiResponse.Message.Content}
	if err := json.NewEncoder(w).Encode(clientResp); err != nil {
		debugLog("error", "Error encoding client response JSON: %s", err)
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
	}
}

func debugLog(errorType LogTypes, v ...interface{}) {
	if debug {
		switch errorType {
		case Info:
			fmt.Fprintf(debugConsole, "[green]DEBUG (Info): %v[-]\n", v)
		case Error:
			fmt.Fprintf(debugConsole, "[red]DEBUG (Error): %v[-]\n", v)
		case Warning:
			fmt.Fprintf(debugConsole, "[yellow]DEBUG (Warning): %v[-]\n", v)
		}
	}
}
