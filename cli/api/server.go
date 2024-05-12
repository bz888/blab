package api

import (
	"bytes"
	"encoding/json"
	"github.com/rivo/tview"
	"io"
	"log"
	"net/http"
)

// ClientRequest Request from client
type ClientRequest struct {
	Text string `json:"text"`
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

var URL = "http://localhost:11434/"

var debugMode bool // Flag to control debug logging
var debugConsoleGlob *tview.TextView

func debugLog(v ...interface{}) {
	if debugMode {
		//fmt.Fprintf(debugConsoleGlob, "DEBUG: %v\n", v)
		log.Println(v)
	}
}

// Handler function that processes text
func processTextHandler(w http.ResponseWriter, r *http.Request) {
	var clientReq ClientRequest

	err := json.NewDecoder(r.Body).Decode(&clientReq)
	debugLog("Processing request:", clientReq)

	if err != nil {
		log.Printf("Error decoding client JSON: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(r.Body)

	// Prepare the request for the external API
	apiReq := APIRequest{
		Model: "llama3", // this should be selectable
		Messages: []Message{
			{
				Role:    "user",
				Content: clientReq.Text,
			},
		},
		Stream: false,
	}

	requestData, err := json.Marshal(apiReq)
	debugLog("Sending data to API:", string(requestData))

	if err != nil {
		log.Printf("Error marshaling API request JSON: %s", err)
		http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
		return
	}

	// Send the request to the external API
	apiURL := "http://localhost:11434/api/chat"
	apiResp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(requestData))
	if err != nil {
		log.Printf("Error calling external API: %s", err)
		http.Error(w, "Error calling external API", http.StatusInternalServerError)
		return
	}
	defer apiResp.Body.Close()

	// Read the response from external API
	var apiResponse APIResponse
	if err := json.NewDecoder(apiResp.Body).Decode(&apiResponse); err != nil {
		log.Printf("Error decoding API response JSON: %s", err)
		http.Error(w, "Error decoding API response JSON", http.StatusInternalServerError)
		return
	}
	debugLog("Received response from API:", apiResponse)

	clientResp := ClientResponse{ProcessedText: apiResponse.Message.Content}
	if err := json.NewEncoder(w).Encode(clientResp); err != nil {
		log.Printf("Error encoding client response JSON: %s", err)
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
	}
}

func StartServer(debugMode bool, debugConsole *tview.TextView) {
	//debugMode = debug // Set the global debug flag based on input
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
	if debugMode {
		//debugConsoleGlob = debugConsole
		log.Println("Server starting on http://localhost:8080/")
		log.Println("Debug mode is enabled")
	}
	log.Fatal(http.ListenAndServe(":8080", nil))
}
