package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	serverClient "github.com/bz888/blab/internal/api/server/client"
	"github.com/bz888/blab/internal/api/server/handlers"
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

func ListModels() ([]string, error) {
	req, err := http.NewRequest("GET", "http://localhost:8080/models", nil)
	if err != nil {
		localLogger.Error("Failed to create get models request: %s\n", err)
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		localLogger.Error("Failed to perform models request: %s\n", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		localLogger.Error("Failed to get models: %s\n", resp.Status)
		return nil, errors.New(resp.Status)
	}

	var models []string
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		localLogger.Error("Failed to decode models response: %s\n", err)
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

	clientReq := serverClient.ChatRequest{Model: model, Text: content}

	localLogger.Info("Input request:", clientReq.Text)
	localLogger.Info("Input model:", clientReq.Model)

	requestData, err := json.Marshal(clientReq)
	if err != nil {
		localLogger.Error("Failed to serialize request: %s\n\n", err)
		return
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/chat", bytes.NewBuffer(requestData))

	if err != nil {
		localLogger.Error("Failed to create request: %s\n\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		localLogger.Error("Failed to send request: %s\n\n", err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			localLogger.Error("Failed to close response body: %s\n\n", err)
		}
	}()

	fmt.Fprintf(textView, "[green::]Bot:[-]\n")
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024) // Create an initial buffer of size 64 KB
	scanner.Buffer(buf, 512*1024)   // Set the maximum buffer size to 512 KB

	accumulatedText := ""
	for scanner.Scan() {
		var clientResp serverClient.ChatResponse

		err := json.Unmarshal(scanner.Bytes(), &clientResp)
		if err != nil {
			localLogger.Error("Failed to decode response: %s\n\n", err)
			continue
		}
		accumulatedText += clientResp.ProcessedText

		app.QueueUpdateDraw(func() {
			fmt.Fprintf(textView, "%s", clientResp.ProcessedText)
		})
	}

	handlers.ChatHistory = append(handlers.ChatHistory, serverClient.ServerChatMessage{
		Role:    serverClient.RoleAssistant,
		Content: accumulatedText,
	})

	localLogger.Info("Chat history:", handlers.ChatHistory)

	if err := scanner.Err(); err != nil {
		localLogger.Error("Failed to read stream: %s\n\n", err)
	}

}
