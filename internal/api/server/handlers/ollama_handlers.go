package handlers

import (
	"encoding/json"
	"github.com/bz888/blab/internal/api/server/client"
	"github.com/bz888/blab/internal/logger"
	"net/http"
)

func (h *Handler) processWithOllamaClient(w http.ResponseWriter, r *http.Request, clientReq client.ChatRequest, chatHistory *[]client.ServerChatMessage) {
	localLogger := logger.NewLogger("Ollama handler")

	*chatHistory = append(*chatHistory, client.ServerChatMessage{
		Role:    client.RoleUser,
		Content: clientReq.Text,
	})

	apiReq := client.ServerChatRequest{
		Model:    clientReq.Model,
		Messages: *chatHistory,
		Stream:   true,
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

	err := h.ollamaClient.Chat(r.Context(), &apiReq, func(bts []byte) error {
		var apiResp client.OllamaAPIResponse
		if err := json.Unmarshal(bts, &apiResp); err != nil {
			localLogger.Error("Failed to unmarshal response:", err)
			localLogger.Error("Raw response data:", string(bts)) // Log the raw response data
			return err
		}

		err := encoder.Encode(client.ChatResponse{ProcessedText: apiResp.Message.Content})
		if !apiResp.Done {
			localLogger.Info("Received response:", apiResp.Message.Content)
		} else {
			localLogger.Info("Completed response", string(bts))
		}

		if err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	})

	if err != nil {
		http.Error(w, "Failed to process request: "+err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) processOllamaModels() []string {
	ollamaModels, err := h.ollamaClient.GetModels()
	// should return just the model name in an array
	if err != nil {
		return []string{}
	}

	modelNames := make([]string, len(ollamaModels))
	for i, model := range ollamaModels {
		modelNames[i] = model.Name
		//h.modelClientMap[model.Name] = "ollama"
	}
	return modelNames
}
