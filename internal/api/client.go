package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bz888/blab/internal/api/server"
	"github.com/bz888/blab/internal/logger"
	"github.com/rivo/tview"
	"net/http"
)

var (
	localLogger *logger.Logger
)

func Init() {
	localLogger = logger.NewLogger("api client")
}

func ListModels() ([]server.Model, error) {
	req, err := http.NewRequest("GET", "http://localhost:8080/models", nil)
	if err != nil {
		localLogger.Error("Failed get models request: %s\n\n", err)
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		localLogger.Error("Failed to perform models request: %s\n\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		localLogger.Error("Failed to get models: %s\n\n", resp.Status)
		return nil, err
	}

	var models []server.Model
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		localLogger.Error("Failed to decode models response: %s\n\n", err)
		return nil, err
	}
	return models, nil
}

// Chatting TODO, refactor, separate the request and the TUI display
func Chatting(model string, content string, app *tview.Application, textView *tview.TextView) {
	if content == "" {
		localLogger.Warn("No content parsed")
		return
	}

	fmt.Fprintln(textView, "\n\n[red::]You:[-]")
	fmt.Fprintf(textView, "%s\n\n", content)

	clientReq := server.ClientRequest{Model: model, Text: content}
	localLogger.Info("Input request:", clientReq.Text)
	requestData, err := json.Marshal(clientReq)
	if err != nil {
		localLogger.Error("Failed to serialize request: %s\n\n", err)
		return
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/process_text", bytes.NewBuffer(requestData))
	if err != nil {
		localLogger.Error("Failed to create request: %s\n\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		localLogger.Error("Failed to send request: %s\n\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Fprintf(textView, "[green::]Bot:[-]\n")
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024) // Create an initial buffer of size 64 KB
	scanner.Buffer(buf, 512*1024)   // Set the maximum buffer size to 512 KB

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
