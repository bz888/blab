package main

import (
    "bytes"
    "encoding/json"
    "log"
    "net/http"
)

// Request from client
type ClientRequest struct {
    Text string `json:"text"`
}

// Request to external API
type APIRequest struct {
    Model    string   `json:"model"`
    Messages []Message `json:"messages"`
    Stream   bool     `json:"stream"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type APIResponse struct {
    Model            string   `json:"model"`
    CreatedAt        string   `json:"created_at"`
    Message          Message  `json:"message"`
    Done             bool     `json:"done"`
    TotalDuration    int64    `json:"total_duration"`
    LoadDuration     int64    `json:"load_duration"`
    PromptEvalCount  int      `json:"prompt_eval_count"`
    PromptEvalDuration int64  `json:"prompt_eval_duration"`
    EvalCount        int      `json:"eval_count"`
    EvalDuration     int64    `json:"eval_duration"`
}

// Response to client
type ClientResponse struct {
    ProcessedText string `json:"processedText"`
}

var URL = "http://localhost:11434/"

// Handler function that processes text
func processTextHandler(w http.ResponseWriter, r *http.Request) {
    var clientReq ClientRequest

    err := json.NewDecoder(r.Body).Decode(&clientReq)
    log.Println(r.Body)

    if err != nil {
        log.Printf("Error decoding client JSON: %s", err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

    // Prepare the request for the external API
    apiReq := APIRequest{
        Model: "llama3",
        Messages: []Message{
            {
                Role:    "user",
                Content: clientReq.Text,
            },
        },
        Stream: false,
    }

    requestData, err := json.Marshal(apiReq)
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

    log.Printf("Received response from other service: %s", apiResponse.Message.Content)

    // Send the response back to the client
    clientResp := ClientResponse{ProcessedText: apiResponse.Message.Content}
    if err := json.NewEncoder(w).Encode(clientResp); err != nil {
        log.Printf("Error encoding client response JSON: %s", err)
        http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
    }
}

func main() {
    http.HandleFunc("/process_text", processTextHandler)
    log.Println("Server starting on http://localhost:8080/")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
