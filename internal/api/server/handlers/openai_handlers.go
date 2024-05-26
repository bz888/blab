package handlers

import (
	"encoding/json"
	"github.com/bz888/blab/internal/api/server/client"
	"net/http"
	"os"
)

func (h *Handler) processWithOpenAIClient(w http.ResponseWriter, r *http.Request, clientReq client.ChatRequest) {
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
	w.Header().Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	encoder := json.NewEncoder(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	err := h.openAIClient.Chat(r.Context(), &apiReq, func(bts []byte) error {
		var apiResp client.OpenAIChatResponse
		if err := json.Unmarshal(bts, &apiResp); err != nil {
			return err
		}

		// TODO support selectable choices
		err := encoder.Encode(client.ChatResponse{ProcessedText: apiResp.Choices[0].Message.Content})
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

func (h *Handler) processOpenAiModels() []string {
	//var localLogger = logger.NewLogger("openai handler")

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
