package main

import (
    "encoding/json"
    "log"
    "net/http"
)

// Define a struct to parse incoming JSON data
type Request struct {
    Text string `json:"text"`
}

// Define a struct to format outgoing JSON data
type Response struct {
    ProcessedText string `json:"processedText"`
}

// Handler function that processes text
func processTextHandler(w http.ResponseWriter, r *http.Request) {
    var req Request

    // Log the incoming request
    log.Println("Received a new request")

    // Decode JSON body from the request
    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        log.Printf("Error decoding JSON: %s", err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer r.Body.Close()
    log.Printf("Decoded text: %s", req.Text)

    // Process the text (e.g., modify, analyze, etc.)
    processedText := "Processed: " + req.Text
    log.Printf("Processing completed: %s", processedText)

    // Create a response struct
    resp := Response{ProcessedText: processedText}

    // Encode and send the response as JSON
    if err := json.NewEncoder(w).Encode(resp); err != nil {
        log.Printf("Error encoding JSON: %s", err)
        http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
    }
}

// Main function to set up the routes and start the server
func main() {
    // Set up URL endpoint and handler function
    http.HandleFunc("/process_text", processTextHandler)
    log.Println("Server starting on http://localhost:8080/")

    // Start the HTTP server on port 8080
    log.Fatal(http.ListenAndServe(":8080", nil))
}
