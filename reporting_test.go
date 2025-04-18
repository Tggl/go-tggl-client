package tggl

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

//go:embed reporting_tests.json
var reportingTests []byte

type MockReporting struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockReporting) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

type ReportingTestCase struct {
	Name      string `json:"name"`
	App       string `json:"app"`
	AppPrefix string `json:"appPrefix"`
	Calls     []struct {
		Type         string      `json:"type"`
		Value        interface{} `json:"value"`
		DefaultValue interface{} `json:"defaultValue"`
		Context      Context     `json:"context"`
		Slug         string      `json:"slug"`
	}
	Result interface{} `json:"result"`
}

func TestReporting(t *testing.T) {
	// Decode test cases
	var testCases []ReportingTestCase
	if err := json.Unmarshal(reportingTests, &testCases); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			callsCount := 0
			mockClient := &MockReporting{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					callsCount++
					if req.Header.Get("X-Tggl-Api-Key") != "API_KEY" {
						t.Errorf("Expected X-Tggl-Api-Key header to be %q, got %q", "API_KEY", req.Header.Get("X-Tggl-Api-Key"))
					}

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				},
			}

			reporting := NewReporting("API_KEY", tc.App, tc.AppPrefix, mockClient)

			if callsCount != 1 {
				t.Errorf("Expected 1 call, got %d", callsCount)
			}

			for _, call := range tc.Calls {
				if call.Type == "flag" {
					if err := reporting.ReportFlag(call.Slug, &FlagValue{
						Default: call.DefaultValue,
						Value:   call.Value,
					}); err != nil {
						t.Errorf("Error reporting flag: %v", err)
					}
					continue
				}

				if call.Type == "context" {
					if err := reporting.ReportContext(call.Slug, &call.Context); err != nil {
						t.Errorf("Error reporting context: %v", err)
					}
					continue
				}
				t.Errorf("Unexpected call type: %v", call.Type)
			}
		})
	}
}
