package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Stream   bool                `json:"stream,omitempty"`
}

type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Choices []OpenAIChatChoice `json:"choices"`
	Usage   OpenAIUsage        `json:"usage"`
}

type OpenAIChatChoice struct {
	Index        int               `json:"index"`
	Message      OpenAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
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
		if model.OwnedBy == "openai" {
			CacheModels[model.ID] = "openai"
		}
	}
}

// Chat makes a chat request to the OpenAI API
func (c *OpenAIClient) Chat(ctx context.Context, req *OpenAIChatRequest, fn func([]byte) error) error {
	return c.stream(ctx, http.MethodPost, req, fn)
}

func (c *OpenAIClient) stream(ctx context.Context, method string, data any, fn func([]byte) error) error {
	var buf *bytes.Buffer
	if data != nil {
		bts, err := json.Marshal(data)
		if err != nil {
			return err
		}
		buf = bytes.NewBuffer(bts)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.GetChatURL(), buf)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

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
