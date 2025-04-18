package tggl

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestEvaluateContexts(t *testing.T) {
	t.Run("successful contexts evaluation", func(t *testing.T) {
		mockClient := &MockConfig{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				// Vérifie les headers essentiels
				if req.Header.Get("X-Tggl-Api-Key") != "test-key" {
					t.Error("API key header not set correctly")
				}
				if req.Header.Get("Content-Type") != "application/json" {
					t.Error("Content-Type header not set correctly")
				}

				// Simule une réponse valide avec deux contextes
				response := `[{"flag1":true},{"flag2":"value"}]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(response)),
				}, nil
			},
		}

		client := NewClient("test-key", mockClient)
		contexts := []Context{
			{"userId": "123"},
			{"userId": "456"},
		}

		flags, err := client.EvaluateContexts(contexts)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(flags) != 2 {
			t.Errorf("Expected 2 flag responses, got %d", len(flags))
		}
		if flags[0]["flag1"] != true {
			t.Errorf("Expected flag1 to be true")
		}
		if flags[1]["flag2"] != "value" {
			t.Errorf("Expected flag2 to be 'value'")
		}
	})

	t.Run("unauthorized error", func(t *testing.T) {
		mockClient := &MockConfig{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}, nil
			},
		}

		client := NewClient("invalid-key", mockClient)
		_, err := client.EvaluateContexts([]Context{{"userId": "123"}})

		if err == nil || err.Error() != "invalid API key" {
			t.Errorf("Expected invalid API key error, got %v", err)
		}
	})

	t.Run("server error with message", func(t *testing.T) {
		mockClient := &MockConfig{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error":"invalid context format"}`)),
				}, nil
			},
		}

		client := NewClient("test-key", mockClient)
		_, err := client.EvaluateContexts([]Context{{"userId": "123"}})

		if err == nil || err.Error() != "API error: invalid context format" {
			t.Errorf("Expected API error with message, got %v", err)
		}
	})

	t.Run("server error without message", func(t *testing.T) {
		mockClient := &MockConfig{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}, nil
			},
		}

		client := NewClient("test-key", mockClient)
		_, err := client.EvaluateContexts([]Context{{"userId": "123"}})

		if err == nil || err.Error() != "API error: status code 500" {
			t.Errorf("Expected API error with status code, got %v", err)
		}
	})
}

func TestEvaluateContext(t *testing.T) {
	// Un seul test car EvaluateContext est un wrapper autour de EvaluateContexts
	// Les cas d'erreur sont déjà couverts par les tests de EvaluateContexts
	t.Run("evaluates single context", func(t *testing.T) {
		mockClient := &MockConfig{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				response := `[{"flag1":42}]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(response)),
				}, nil
			},
		}

		client := NewClient("test-key", mockClient)
		flags, err := client.EvaluateContext(Context{"userId": "123"})

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if (*flags)["flag1"] != float64(42) {
			t.Errorf("Expected flag1 to be 42")
		}
	})

	t.Run("unauthorized error", func(t *testing.T) {
		mockClient := &MockConfig{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(bytes.NewReader([]byte{})),
				}, nil
			},
		}

		client := NewClient("invalid-key", mockClient)
		_, err := client.EvaluateContext(Context{"userId": "123"})

		if err == nil || err.Error() != "invalid API key" {
			t.Errorf("Expected invalid API key error, got %v", err)
		}
	})
}

func TestFlagResponseGet(t *testing.T) {
	// Test with string value - most common case for feature flags
	t.Run("get string value", func(t *testing.T) {
		response := FlagResponse{"feature": "enabled"}

		if v := response.Get("feature", "default"); v != "enabled" {
			t.Errorf("Expected 'enabled', got %v", v)
		}
	})

	// Test with boolean value - important for simple toggles
	t.Run("get boolean value", func(t *testing.T) {
		response := FlagResponse{"feature": true}

		if v := response.Get("feature", false); v != true {
			t.Errorf("Expected true, got %v", v)
		}
	})

	// Test with numeric value - useful for gradual rollouts
	t.Run("get numeric value", func(t *testing.T) {
		response := FlagResponse{"feature": 42.0}

		if v := response.Get("feature", 0.0); v != 42.0 {
			t.Errorf("Expected 42.0, got %v", v)
		}
	})

	// Test with non-existent key - verifies default behavior
	t.Run("get non-existent key returns default", func(t *testing.T) {
		response := FlagResponse{}

		if v := response.Get("unknown", "default"); v != "default" {
			t.Errorf("Expected 'default', got %v", v)
		}
	})

	// Test with nil value - important for handling edge cases
	t.Run("get nil value", func(t *testing.T) {
		response := FlagResponse{"feature": nil}

		if v := response.Get("feature", "default"); v != nil {
			t.Errorf("Expected 'default', got %v", v)
		}
	})
}
