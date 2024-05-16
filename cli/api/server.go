package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
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

var host = "localhost:11434"
var port = 8080

// Handler function that processes text
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
		base: &url.URL{Scheme: "http", Host: host},
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

		//if !apiResp.Done {
		//	localLogger.Info("Received response:", apiResp.Message.Content)
		//} else {
		//	localLogger.Info("Completed response", string(bts))
		//}

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

	address := ":" + strconv.Itoa(port)
	log.Println("Debug mode is enabled")
	log.Println("Server started on http://localhost" + address + "/")

	// Start the server
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("Error starting server: ", err)
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
