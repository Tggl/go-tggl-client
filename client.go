package tggl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client represents the Tggl API client for context evaluation
type Client struct {
	baseURL    string
	apiKey     string
	httpClient HTTPClient
}

// NewClient creates a new instance of the client for context evaluation
func NewClient(apiKey string, httpClt HTTPClient) *Client {
	return &Client{
		baseURL:    "https://api.tggl.io",
		apiKey:     apiKey,
		httpClient: httpClt,
	}
}

// FlagResponse represents the API response for flags
type FlagResponse map[string]interface{}

// EvaluateContexts evaluates flags for a list of contexts
func (c *Client) EvaluateContexts(contexts []Context) ([]FlagResponse, error) {
	body, err := json.Marshal(contexts)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize context: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/flags", c.baseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tggl-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("API error: status code %d", resp.StatusCode)
	}

	var flags []FlagResponse
	if err := json.NewDecoder(resp.Body).Decode(&flags); err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	return flags, nil
}

// EvaluateContext evaluates flags for a single context
func (c *Client) EvaluateContext(context Context) (*FlagResponse, error) {
	contexts := []Context{context}
	flags, err := c.EvaluateContexts(contexts)
	if err != nil {
		return &FlagResponse{}, err
	}
	return &flags[0], nil
}

// GetBool returns the boolean value for the given key or the default value if the key doesn't exist
func (f *FlagResponse) GetBool(slug string, defaultValue bool) bool {
	val, ok := f.Get(slug, defaultValue).(bool)
	if !ok {
		return defaultValue
	}
	return val
}

// GetString returns the string value for the given key or the default value if the key doesn't exist
func (f *FlagResponse) GetString(slug string, defaultValue string) string {
	val, ok := f.Get(slug, defaultValue).(string)
	if !ok {
		return defaultValue
	}
	return val
}

// GetFloat64 returns the float64 value for the given key or the default value if the key doesn't exist
func (f *FlagResponse) GetFloat64(slug string, defaultValue float64) float64 {
	val, ok := f.Get(slug, defaultValue).(float64)
	if !ok {
		return defaultValue
	}
	return val
}

// GetInt returns the int value for the given key or the default value if the key doesn't exist
func (f *FlagResponse) GetInt(slug string, defaultValue int) int {
	val, ok := f.Get(slug, defaultValue).(int)
	if !ok {
		return defaultValue
	}
	return val
}

func (f *FlagResponse) Get(slug string, defaultValue any) any {
	val, ok := (*f)[slug]
	if !ok {
		return defaultValue
	}
	return val
}
