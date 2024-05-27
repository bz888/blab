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
	"time"
)

// OllamaClient represents a client for the Ollama API
type OllamaClient struct {
	Client
}

type OllamaClientInterface interface {
	GetModels() ([]OllamaModel, error)
	Chat(ctx context.Context, req *OllamaChatRequest, fn func([]byte) error) error
}

var ollamaConfig = ClientConfig{
	Scheme:     "http",
	Host:       "localhost:11434",
	ModelsPath: "/api/tags",
	ChatPath:   "/api/chat",
}

// NewOllamaClient creates a new Ollama API client
func NewOllamaClient() *OllamaClient {
	return &OllamaClient{
		Client: *NewClient(ollamaConfig),
	}
}

type OllamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaMessageResponse struct {
	Model              string        `json:"model"`
	CreatedAt          string        `json:"created_at"`
	Message            OllamaMessage `json:"message"`
	Done               bool          `json:"done"`
	TotalDuration      int64         `json:"total_duration"`
	LoadDuration       int64         `json:"load_duration"`
	PromptEvalCount    int           `json:"prompt_eval_count"`
	PromptEvalDuration int64         `json:"prompt_eval_duration"`
	EvalCount          int           `json:"eval_count"`
	EvalDuration       int64         `json:"eval_duration"`
}

type OllamaAPIResponse struct {
	Message OllamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

type ModelsResponse struct {
	Models []OllamaModel `json:"models"`
}

type OllamaModel struct {
	Name       string       `json:"name"`
	ModifiedAt time.Time    `json:"modified_at"`
	Size       int64        `json:"size"`
	Digest     string       `json:"digest"`
	Details    ModelDetails `json:"details"`
}

type Families []string

// ModelDetails Details represents the details of a model.
type ModelDetails struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          Families `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

func (c *OllamaClient) GetModels() ([]OllamaModel, error) {
	requestURL := c.GetModelsURL()

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
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

	var response ModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	AddOllamaModelCache(response.Models)

	return response.Models, nil
}

func AddOllamaModelCache(ollamaModels []OllamaModel) {
	for _, model := range ollamaModels {
		CacheModels[model.Name] = "ollama"
	}
}

func (c *OllamaClient) Chat(ctx context.Context, req *OllamaChatRequest, fn func([]byte) error) error {
	return c.stream(ctx, req, fn)
}

func (c *OllamaClient) stream(ctx context.Context, data *OllamaChatRequest, fn func([]byte) error) error {
	localLogger := logger.NewLogger("ollama stream chat")
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

// UnmarshalJSON handles the custom unmarshalling for Families.
func (f *Families) UnmarshalJSON(data []byte) error {
	// If the JSON data is "null", return an empty Families slice.
	if string(data) == "null" {
		*f = Families{}
		return nil
	}

	// Otherwise, unmarshal the data as a regular slice of strings.
	var families []string
	if err := json.Unmarshal(data, &families); err != nil {
		return err
	}
	*f = Families(families)
	return nil
}
