package tggl

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Variation represents a flag variation with its value and active state
type Variation struct {
	Active bool        `json:"active"`
	Value  interface{} `json:"value"`
}

// Rule represents a rule for flag evaluation
type Rule struct {
	Key      string      `json:"key,omitempty"`
	Operator string      `json:"operator,omitempty"`
	Negate   *bool       `json:"negate,omitempty"`
	Values   []string    `json:"values,omitempty"`
	Value    interface{} `json:"value,omitempty"`
	Version  []int       `json:"version,omitempty"`
	// Fields for percentage operator
	RangeStart *float64 `json:"rangeStart,omitempty"`
	RangeEnd   *float64 `json:"rangeEnd,omitempty"`
	Seed       *int     `json:"seed,omitempty"`
	// Fields for date operator
	Timestamp *int    `json:"timestamp,omitempty"`
	ISO       *string `json:"iso,omitempty"`
}

// Condition represents a condition group with rules and variation
type Condition struct {
	Rules     []Rule    `json:"rules"`
	Variation Variation `json:"variation"`
}

// Flag represents a feature flag configuration
type Flag struct {
	Slug             string      `json:"slug"`
	DefaultVariation Variation   `json:"defaultVariation"`
	Conditions       []Condition `json:"conditions"`
}

// LocalClient represents the Tggl API client for configuration management
type LocalClient struct {
	baseURL      string
	serverAPIKey string
	httpClient   HTTPClient
	flags        []Flag
}

// NewLocalClient creates a new instance of the client for configuration management
func NewLocalClient(serverAPIKey string, httpClt HTTPClient) *LocalClient {
	return &LocalClient{
		baseURL:      "https://api.tggl.io",
		serverAPIKey: serverAPIKey,
		httpClient:   httpClt,
	}
}

// GetConfig retrieves the configuration from the API
func (c *LocalClient) GetConfig() error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/config", c.baseURL), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Tggl-Api-Key", c.serverAPIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make API call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid server API key")
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("API error: %s", errResp.Error)
		}
		return fmt.Errorf("API error: status code %d", resp.StatusCode)
	}

	var flags []Flag
	if err := json.NewDecoder(resp.Body).Decode(&flags); err != nil {
		return fmt.Errorf("failed to deserialize response: %w", err)
	}
	c.flags = flags
	return nil
}

func (c *LocalClient) GetString(context Context, slug, defaultValue string) string {
	typeVal, ok := c.Get(context, slug, defaultValue).(string)
	if !ok {
		return defaultValue
	}
	return typeVal
}

func (c *LocalClient) GetFloat64(context Context, slug string, defaultValue float64) float64 {
	typeVal, ok := c.Get(context, slug, defaultValue).(float64)
	if !ok {
		return defaultValue
	}
	return typeVal
}

func (c *LocalClient) GetInt(context Context, slug string, defaultValue int) int {
	typeVal, ok := c.Get(context, slug, defaultValue).(int)
	if !ok {
		return defaultValue
	}
	return typeVal
}

func (c *LocalClient) GetBool(context Context, slug string, defaultValue bool) bool {
	typeVal, ok := c.Get(context, slug, defaultValue).(bool)
	if !ok {
		return defaultValue
	}
	return typeVal
}

func (c *LocalClient) Get(context Context, slug string, defaultValue any) any {
	var flag Flag
	notFound := true
	for i := 0; i < len(c.flags); i++ {
		f := c.flags[i]
		if f.Slug == slug {
			flag = f
			notFound = false
		}
	}
	if notFound {
		return defaultValue
	}
	return evalFlag(flag, context)
}
