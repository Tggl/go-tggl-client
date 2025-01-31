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

// GetConfig retrieves the configuration from the API
func (c *LocalClient) GetConfig() ([]Flag, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/config", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Tggl-Api-Key", c.serverAPIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid server API key")
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

	var flags []Flag
	if err := json.NewDecoder(resp.Body).Decode(&flags); err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	return flags, nil
}

func (c *LocalClient) GetString(context map[string]interface{}, slug, defaultValue string) string {
	res, ok := getFromConfig[string](c.flags, context, slug)
	if !ok {
		return defaultValue
	}
	return res
}

func (c *LocalClient) GetFloat64(context map[string]interface{}, slug string, defaultValue float64) float64 {
	res, ok := getFromConfig[float64](c.flags, context, slug)
	if !ok {
		return defaultValue
	}
	return res
}

func (c *LocalClient) GetInt(context map[string]interface{}, slug string, defaultValue int) int {
	res, ok := getFromConfig[int](c.flags, context, slug)
	if !ok {
		return defaultValue
	}
	return res
}

func (c *LocalClient) GetBool(context map[string]interface{}, slug string, defaultValue bool) bool {
	res, ok := getFromConfig[bool](c.flags, context, slug)
	if !ok {
		return defaultValue
	}
	return res
}

func getFromConfig[T any](flags []Flag, context map[string]interface{}, slug string) (T, bool) {
	var flag Flag
	var res T
	notFound := true
	for i := 0; i < len(flags); i++ {
		f := flags[i]
		if f.Slug == slug {
			flag = f
			notFound = false
		}
	}
	if notFound {
		return res, false
	}
	typeVal, ok := evalFlag(flag, context).(T)
	if !ok {
		return res, false
	}
	return typeVal, true
}
