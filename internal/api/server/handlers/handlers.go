package handlers

import (
	"encoding/json"
	"github.com/bz888/blab/internal/api/server/client"
	"net/http"
	"sync"
)

type Handler struct {
	openAIClient client.OpenAIClientInterface
	ollamaClient client.OllamaClientInterface
}

func NewHandler(openAIClient client.OpenAIClientInterface, ollamaClient client.OllamaClientInterface, modelClientMap map[string]string) *Handler {
	return &Handler{
		openAIClient: openAIClient,
		ollamaClient: ollamaClient,
	}
}

func (h *Handler) ProcessTextHandler(w http.ResponseWriter, r *http.Request) {
	var clientReq client.ChatRequest
	err := json.NewDecoder(r.Body).Decode(&clientReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	clientType, ok := client.CacheModels[clientReq.Model]

	if !ok {
		http.Error(w, "Model not found", http.StatusBadRequest)
		return
	}

	if clientType == "openai" {
		h.processWithOpenAIClient(w, r, clientReq)
	} else if clientType == "ollama" {
		h.processWithOllamaClient(w, r, clientReq)
	} else {
		http.Error(w, "Unknown client type", http.StatusInternalServerError)
	}
}

func (h *Handler) ModelHandler(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	models := make([]string, 0)
	modelsChan := make(chan []string, 2)
	errChan := make(chan error, 2)

	if h.ollamaClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ollamaModels := h.processOllamaModels()
			modelsChan <- ollamaModels
		}()
	}

	if h.openAIClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			openAIModels := h.processOpenAiModels()
			modelsChan <- openAIModels
		}()
	}

	go func() {
		wg.Wait()
		close(modelsChan)
		close(errChan)
	}()

	for modelList := range modelsChan {
		models = append(models, modelList...)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}
