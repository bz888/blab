package client

// ChatRequest ClientRequest Request from client
type ChatRequest struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

// ChatResponse ClientResponse Response to client
type ChatResponse struct {
	// add role to it
	ProcessedText string `json:"processedText"`
}
