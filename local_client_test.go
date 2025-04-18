package tggl

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// MockConfig implements the HTTPClient interface for testing
type MockConfig struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockConfig) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestGetConfig(t *testing.T) {
	// Test the nominal case - valid response with flags
	t.Run("successful config retrieval", func(t *testing.T) {
		mockClient := &MockConfig{
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
		mockClient := &MockConfig{
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

func TestPolling(t *testing.T) {
	t.Run("polling makes periodic calls and stops correctly", func(t *testing.T) {
		// Create a channel to track API calls
		calls := make(chan time.Time, 10)

		mockClient := &MockConfig{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				// Record the time of each call
				calls <- time.Now()
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				}, nil
			},
		}

		// Create client with 100ms polling interval
		client := NewLocalClient("test-key", mockClient, WithPollingInterval(100))

		// Wait for at least 3 calls (should take ~300ms)
		var callTimes []time.Time
		timeout := time.After(500 * time.Millisecond)

		for i := 0; i < 3; i++ {
			select {
			case callTime := <-calls:
				callTimes = append(callTimes, callTime)
			case <-timeout:
				t.Fatalf("Timeout waiting for API calls, got only %d calls", len(callTimes))
			}
		}

		// Verify intervals between calls (should be ~100ms)
		for i := 1; i < len(callTimes); i++ {
			interval := callTimes[i].Sub(callTimes[i-1])
			if interval < 90*time.Millisecond || interval > 110*time.Millisecond {
				t.Errorf("Expected interval around 100ms, got %v", interval)
			}
		}

		// Stop polling
		client.StopPolling()

		// Wait a bit to ensure no more calls are made
		time.Sleep(200 * time.Millisecond)

		// Verify no more calls were made
		select {
		case <-calls:
			t.Error("Received API call after stopping polling")
		default:
			// No calls received, which is expected
		}
	})

	t.Run("StartPolling with invalid interval returns error", func(t *testing.T) {
		client := NewLocalClient("test-key", &MockConfig{})
		client.pollingInterval = 0

		err := client.StartPolling()
		if err == nil {
			t.Error("Expected error for zero polling interval")
		}
	})

	t.Run("StartPolling when already polling does nothing", func(t *testing.T) {
		calls := 0
		mockClient := &MockConfig{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				}, nil
			},
		}

		client := NewLocalClient("test-key", mockClient, WithPollingInterval(100))

		// Try to start polling again
		err := client.StartPolling()
		if err != nil {
			t.Errorf("Expected no error when starting polling again, got %v", err)
		}

		// Wait a bit and check call count
		time.Sleep(250 * time.Millisecond)
		expectedCalls := 3 // Initial call + 2 polling calls
		if calls > expectedCalls {
			t.Errorf("Expected around %d calls, got %d (multiple polling routines?)", expectedCalls, calls)
		}

		client.StopPolling()
	})
}
