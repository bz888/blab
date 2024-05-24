package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func newClient(host string) *Client {
	return &Client{
		base: &url.URL{Scheme: "http", Host: host},
		http: &http.Client{},
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
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

	return response.Models, nil
}
