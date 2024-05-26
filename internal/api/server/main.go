package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/bz888/blab/internal/api/server/client"
	"github.com/bz888/blab/internal/api/server/handlers"
	"github.com/bz888/blab/internal/logger"
	"log"
	"net/http"
	"os"
	"strconv"
)

var (
	LocalLogger    *logger.Logger
	port           = 8080
	modelClientMap = make(map[string]string)
)

func Init() {
	LocalLogger = logger.NewLogger("Server")
}

func Run() {
	handler, err := initializeClients()
	if err != nil {
		log.Fatal(err)
	}

	registerRoutes(handler)

	address := ":" + strconv.Itoa(port)
	LocalLogger.Info("Debug mode is enabled")

	// Start the server
	LocalLogger.Info("Server started on http://localhost" + address + "/")
	err = http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

func initializeClients() (*handlers.Handler, error) {
	var openAIClient *client.OpenAIClient
	var ollamaClient *client.OllamaClient

	openAIAvailable := checkOpenAIAvailability()
	ollamaAvailable := checkOllamaAvailability()

	if openAIAvailable {
		openAIClient = client.NewOpenAIClient()
		openAIClient.GetModels()

		LocalLogger.Info("OpenAI client initialized.")
	}

	if ollamaAvailable {
		ollamaClient = client.NewOllamaClient()
		ollamaClient.GetModels()

		LocalLogger.Info("Ollama client initialized.")
	}

	LocalLogger.Info("Cached models", client.CacheModels)

	if !openAIAvailable && !ollamaAvailable {
		return nil, errors.New("no clients available")
	}

	return handlers.NewHandler(openAIClient, ollamaClient, modelClientMap), nil
}

func checkOllamaAvailability() bool {
	resp, err := http.Get("http://localhost:11434")
	if err != nil {
		LocalLogger.Error("Ollama server not available:", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// checkOpenAIAvailability verifies the API key by making requests to both /v1/models and /v1/chat/completions endpoints
func checkOpenAIAvailability() bool {
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		LocalLogger.Warn("OpenAI API key not provided.")
		return false
	}

	// Check /v1/models endpoint
	//if !checkEndpoint(openAIKey, "https://api.openai.com/v1/models") {
	//	return false
	//}

	//// Check /v1/chat/completions endpoint
	//if !checkChatCompletionsEndpoint(openAIKey) {
	//	return false
	//}

	return true
}

// todo refactor these to be tests
func checkEndpoint(apiKey, url string) bool {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		LocalLogger.Error("Failed to create request:", err)
		return false
	}
	req.Header.Add("Authorization", "Bearer "+apiKey)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		LocalLogger.Error("Failed to access endpoint:", url, "Status:", resp.Status)
		return false
	}
	defer resp.Body.Close()
	return true
}

func checkChatCompletionsEndpoint(apiKey string) bool {
	testChatCompletion := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "Hello!",
			},
		},
	}
	reqBody, err := json.Marshal(testChatCompletion)
	if err != nil {
		LocalLogger.Error("Failed to create request body for OpenAI chat completion check:", err)
		return false
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		LocalLogger.Error("Failed to create request:", err)
		return false
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		LocalLogger.Error("Failed to access endpoint: https://api.openai.com/v1/chat/completions", "Status:", resp.Status)
		return false
	}
	defer resp.Body.Close()
	return true
}
