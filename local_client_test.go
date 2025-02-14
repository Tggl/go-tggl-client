package tggl

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// MockHTTPClient implements the HTTPClient interface for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestGetConfig(t *testing.T) {
	// Test the nominal case - valid response with flags
	t.Run("successful config retrieval", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("X-Tggl-Api-Key") != "test-key" {
					t.Error("API key header not set correctly")
				}

				// Simulate a valid response with flags
				response := `[{"slug":"test-flag","defaultVariation":{"active":true,"value":true}}]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(response)),
				}, nil
			},
		}

		client := NewLocalClient("test-key", mockClient)
		err := client.GetConfig()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(client.flags) != 1 {
			t.Errorf("Expected 1 flag, got %d", len(client.flags))
		}
	})

	t.Run("unauthorized error", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}, nil
			},
		}

		client := NewLocalClient("invalid-key", mockClient)
		err := client.GetConfig()

		if err == nil || err.Error() != "invalid server API key" {
			t.Errorf("Expected invalid server API key error, got %v", err)
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("matching condition returns variation value", func(t *testing.T) {
		client := &LocalClient{
			flags: []Flag{
				{
					Slug: "test-flag",
					Conditions: []Condition{
						{
							Rules: []Rule{{
								Key:      "userId",
								Operator: "STR_EQUAL",
								Values:   []string{"123"},
							}},
							Variation: Variation{Active: true, Value: "test-value"},
						},
					},
				},
			},
		}

		result := client.Get(Context{"userId": "123"}, "test-flag", "default")
		if result != "test-value" {
			t.Errorf("Expected test-value, got %v", result)
		}
	})

	t.Run("non-existent flag returns default value", func(t *testing.T) {
		client := &LocalClient{flags: []Flag{}}
		result := client.Get(Context{}, "unknown-flag", "default")
		if result != "default" {
			t.Errorf("Expected default, got %v", result)
		}
	})

	t.Run("no matching conditions returns default variation", func(t *testing.T) {
		client := &LocalClient{
			flags: []Flag{
				{
					Slug:             "test-flag",
					DefaultVariation: Variation{Active: true, Value: "default-variation"},
					Conditions: []Condition{
						{
							Rules: []Rule{{
								Key:      "userId",
								Operator: "STR_EQUAL",
								Values:   []string{"123"},
							}},
							Variation: Variation{Active: true, Value: "test-value"},
						},
					},
				},
			},
		}

		result := client.Get(Context{"userId": "456"}, "test-flag", "default")
		if result != "default-variation" {
			t.Errorf("Expected default-variation, got %v", result)
		}
	})
}
