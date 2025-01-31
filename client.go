package tggl

import "net/http"

// Client represents the Tggl API client for context evaluation
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// LocalClient represents the Tggl API client for configuration management
type LocalClient struct {
	baseURL      string
	serverAPIKey string
	httpClient   *http.Client
	flags        []Flag
}

// NewClient creates a new instance of the client for context evaluation
func NewClient(apiKey string) *Client {
	return &Client{
		baseURL:    "https://api.tggl.io",
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// NewLocalClient creates a new instance of the client for configuration management
func NewLocalClient(serverAPIKey string) *LocalClient {
	return &LocalClient{
		baseURL:      "https://api.tggl.io",
		serverAPIKey: serverAPIKey,
		httpClient:   &http.Client{},
	}
}
