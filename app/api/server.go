package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	logger "github.com/bz888/blab/utils"
	"github.com/rivo/tview"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Model represents a single model with its details.
type Model struct {
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

// ModelsResponse Response represents the response structure.
type ModelsResponse struct {
	Models []Model `json:"models"`
}

type Client struct {
	base *url.URL
	http *http.Client
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

// ClientRequest Request from client
type ClientRequest struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

// ClientResponse Response to client
type ClientResponse struct {
	ProcessedText string `json:"processedText"`
}

var ollamaHost = "localhost:11434"
var port = 8080

var localLogger *logger.DebugLogger

func InitService(debugConsole *tview.TextView, dev bool, logPath string) {
	localLogger = logger.NewLogger(debugConsole, dev, "server", logPath)
}

func newClient(host string) *Client {
	return &Client{
		base: &url.URL{Scheme: "http", Host: host},
		http: &http.Client{},
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

	client := newClient(ollamaHost)

	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")

	encoder := json.NewEncoder(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	err = client.chat(r.Context(), &apiReq, func(bts []byte) error {
		var apiResp APIResponse
		if err := json.Unmarshal(bts, &apiResp); err != nil {
			return err
		}

		err := encoder.Encode(ClientResponse{ProcessedText: apiResp.Message.Content})
		if !apiResp.Done {
			localLogger.Info("Received response:", apiResp.Message.Content)
		} else {
			localLogger.Info("Completed response", string(bts))
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

func modelHandler(w http.ResponseWriter, r *http.Request) {
	client := newClient(ollamaHost)

	models, err := client.getLocalModels()
	if err != nil {
		http.Error(w, "Failed to fetch data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

func Run() {
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
	http.HandleFunc("/models", modelHandler)

	address := ":" + strconv.Itoa(port)
	localLogger.Info("Debug mode is enabled")
	localLogger.Info("Server started on http://localhost" + address + "/")

	// Start the server
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}

}

func (c *Client) getLocalModels() ([]Model, error) {
	requestURL := c.base.ResolveReference(&url.URL{Path: "/api/tags"})

	req, err := http.NewRequest(http.MethodGet, requestURL.String(), nil)
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

	log.Println(response)
	return response.Models, nil
}

func (c *Client) chat(ctx context.Context, req *APIRequest, fn func([]byte) error) error {
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
