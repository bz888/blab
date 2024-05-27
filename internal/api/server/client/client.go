package client

import (
	"net/http"
	"net/url"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

var CacheModels = make(map[string]string)

// Client represents a client for the API
type Client struct {
	base      *url.URL
	http      *http.Client
	modelsUrl *url.URL
	chatUrl   *url.URL
}

// ClientConfig holds the configuration for the client
type ClientConfig struct {
	Scheme     string
	Host       string
	ModelsPath string
	ChatPath   string
}

// NewClient creates a new API client with configurable base URL and endpoints
func NewClient(config ClientConfig) *Client {
	baseURL := &url.URL{Scheme: config.Scheme, Host: config.Host}
	return &Client{
		base:      baseURL,
		http:      &http.Client{},
		modelsUrl: baseURL.ResolveReference(&url.URL{Path: config.ModelsPath}),
		chatUrl:   baseURL.ResolveReference(&url.URL{Path: config.ChatPath}),
	}
}

func (c *Client) GetModelsURL() string {
	return c.modelsUrl.String()
}

func (c *Client) GetChatURL() string {
	return c.chatUrl.String()
}
