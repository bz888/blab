package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/bz888/blab/internal/api/server/client"
	"github.com/bz888/blab/internal/logger"
	"net/http"
)

func (h *Handler) processWithOpenAIClient(w http.ResponseWriter, r *http.Request, clientReq client.ChatRequest) {
	localLogger := logger.NewLogger("openai handler")
	apiReq := client.OpenAIChatRequest{
		Model: clientReq.Model,
		Messages: []client.OpenAIChatMessage{
			{
				Role:    client.RoleUser,
				Content: clientReq.Text,
			},
		},
		Stream: true,
	}

	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Accept", "text/event-stream")

	encoder := json.NewEncoder(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	respCh := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		defer close(respCh)

		err := h.openAIClient.Chat(r.Context(), &apiReq, func(bts []byte) error {
			localLogger.Info("Received data chunk:", string(bts))

			cleanData := bytes.TrimPrefix(bts, []byte("data: "))
			cleanData = bytes.TrimSpace(cleanData)

			if len(cleanData) == 0 {
				localLogger.Warn("Received empty data chunk")
				return nil
			}

			if string(cleanData) == "[DONE]" {
				return nil
			}

			var apiResp client.OpenAIChatResponse
			if err := json.Unmarshal(cleanData, &apiResp); err != nil {
				localLogger.Error("Failed to unmarshal response:", err)
				localLogger.Error("Raw response data:", string(bts))
				return err
			}

			if apiResp.Choices != nil && len(apiResp.Choices) > 0 && apiResp.Choices[0].Delta.Content != nil {
				content := *apiResp.Choices[0].Delta.Content
				if content != "" {
					localLogger.Info("msg content", content)
					respCh <- content
				}
			} else {
				localLogger.Warn("No content in response choice")
			}

			return nil
		})
		if err != nil {
			localLogger.Error("Error from Chat function:", err)
			errCh <- err
		}
	}()

	for {
		select {
		case <-r.Context().Done():
			return
		case err := <-errCh:
			http.Error(w, "Failed to process request: "+err.Error(), http.StatusInternalServerError)
			return
		case message, ok := <-respCh:
			if !ok {
				return
			}
			if err := encoder.Encode(client.ChatResponse{ProcessedText: message}); err != nil {
				http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
				return
			}
			flusher.Flush()
		}
	}
}

func (h *Handler) processOpenAiModels() []string {

	if len(client.CacheModels) > 0 {
		var cachedModelNames []string
		for key, value := range client.CacheModels {
			if value == "openai" {
				cachedModelNames = append(cachedModelNames, key)
			}
		}
		if len(cachedModelNames) > 0 {
			return cachedModelNames
		}
	}

	openAIModels, err := h.openAIClient.GetModels()
	if err != nil {
		return []string{}
	}
	var modelNames = make([]string, 0)
	for _, model := range openAIModels {
		if model.OwnedBy == "openai" && model.ID != "" {
			modelNames = append(modelNames, model.ID)
		}
	}

	return modelNames
}
