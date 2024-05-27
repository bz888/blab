package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bz888/blab/internal/logger"
	"io"
	"net/http"
	"os"
)

// OpenAIClient represents a client for the OpenAI API
type OpenAIClient struct {
	Client
}

type OpenAIClientInterface interface {
	GetModels() ([]OpenAIModel, error)
	Chat(ctx context.Context, req *OpenAIChatRequest, fn func([]byte) error) error
}

var openAIConfig = ClientConfig{
	Scheme:     "https",
	Host:       "api.openai.com",
	ModelsPath: "/v1/models",
	ChatPath:   "/v1/chat/completions",
}

// NewOpenAIClient creates a new OpenAI API client
func NewOpenAIClient() *OpenAIClient {
	return &OpenAIClient{
		Client: *NewClient(openAIConfig),
	}
}

type OpenAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []OpenAIChatMessage `json:"messages"`
	Stream   bool                `json:"stream"` // Always true for streaming
}

type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []OpenAIChatChoice `json:"choices"`
	Usage   *OpenAIUsage       `json:"usage,omitempty"` // Usage field, pointer to handle null
}

type OpenAIChatChoice struct {
	Delta        OpenAIChatDelta `json:"delta"`
	FinishReason *string         `json:"finish_reason,omitempty"` // Pointer to handle null
	Index        int             `json:"index"`
}

type OpenAIChatDelta struct {
	Content *string `json:"content,omitempty"` // Pointer to handle null
	Role    *string `json:"role,omitempty"`    // Pointer to handle null
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAIModelsResponse struct {
	Object string        `json:"object"`
	Data   []OpenAIModel `json:"data"`
}

type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// GetModels fetches the models from the OpenAI API
func (c *OpenAIClient) GetModels() ([]OpenAIModel, error) {
	requestURL := c.GetModelsURL()

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch data: " + resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response OpenAIModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	AddOpenAIModelCache(response.Data)

	return response.Data, nil
}

func AddOpenAIModelCache(openAIModels []OpenAIModel) {
	for _, model := range openAIModels {
		// todo tidy up
		//if model.OwnedBy == "openai" {
		CacheModels[model.ID] = "openai"
		//}
	}
}

// Chat makes a chat request to the OpenAI API
func (c *OpenAIClient) Chat(ctx context.Context, req *OpenAIChatRequest, fn func([]byte) error) error {
	return c.stream(ctx, req, fn)
}

func (c *OpenAIClient) stream(ctx context.Context, data *OpenAIChatRequest, fn func([]byte) error) error {
	localLogger := logger.NewLogger("openai stream chat")

	var buf *bytes.Buffer
	if data != nil {
		bts, err := json.Marshal(data)
		if err != nil {
			return err
		}
		buf = bytes.NewBuffer(bts)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.GetChatURL(), buf)
	if err != nil {
		localLogger.Error("Failed to request on ollama chat:", err)
		localLogger.Error("failed request", request)
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		if err := json.NewDecoder(response.Body).Decode(&errResp); err != nil {
			localLogger.Error("Failed to decode error response:", err)
			return fmt.Errorf("received non-200 response: %d, failed to decode error message", response.StatusCode)
		}

		errorMessage := "unknown error"
		if msg, ok := errResp["error"].(map[string]interface{}); ok {
			if message, exists := msg["message"].(string); exists {
				errorMessage = message
			}
		}
		localLogger.Error("Received error response:", errorMessage)
		return fmt.Errorf("received non-200 response: %d, error: %s", response.StatusCode, errorMessage)
	}

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
