package server

import (
	"github.com/bz888/blab/internal/api/server/handlers"
	"net/http"
)

func registerRoutes(handler *handlers.Handler) {
	http.HandleFunc("/chat", handler.ProcessTextHandler)
	http.HandleFunc("/models", handler.ModelHandler)
}
