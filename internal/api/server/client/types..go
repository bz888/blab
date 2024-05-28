package client

// ChatRequest ClientRequest Request from client
type ChatRequest struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

// ChatResponse ClientResponse Response to client
type ChatResponse struct {
	ProcessedText string `json:"processedText"`
}

type ServerChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ServerChatRequest struct {
	Model    string              `json:"model"`
	Messages []ServerChatMessage `json:"messages"`
	Stream   bool                `json:"stream"` // Always true for streaming
}
