package server

import (
	"net/http"
)

func registerRoutes() {
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/process_text", processTextHandler)
	http.HandleFunc("/models", modelHandler)
}
