package tggl

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
	"time"
)

//go:embed reporting_tests.json
var reportingTests []byte

type MockReporting struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockReporting) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

type FakeClock struct {
	fixedTime time.Time
}

func (f FakeClock) Now() time.Time { return f.fixedTime }

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
	Expected *Report `json:"result"`
}

func TestReporting(t *testing.T) {
	// Decode test cases
	var testCases []ReportingTestCase
	if err := json.Unmarshal(reportingTests, &testCases); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	timestampMs := int64(123456789000) // en millisecondes
	sec := timestampMs / 1000
	nsec := (timestampMs % 1000) * int64(time.Millisecond)

	fixedTime := time.Unix(sec, nsec)

	fakeClock := &FakeClock{
		fixedTime: fixedTime,
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			callsCount := 0
			mockClient := &MockReporting{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					callsCount++
					require.Equal(t, "API_KEY", req.Header.Get("X-Tggl-Api-Key"))
					require.Equal(t, "api.tggl.io", req.URL.Host)
					require.Equal(t, "/report", req.URL.Path)
					require.Equal(t, "POST", req.Method)

					var body *Report
					if req.Body != http.NoBody {
						err := json.NewDecoder(req.Body).Decode(&body)
						require.NoError(t, err)
					}

					require.Equal(t, tc.Expected, body)

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				},
			}

			reporting := NewReporting("API_KEY", tc.App, tc.AppPrefix, mockClient, fakeClock)

			for _, call := range tc.Calls {
				if call.Type == "flag" {
					err := reporting.ReportFlag(call.Slug, &FlagValue{
						Default: call.DefaultValue,
						Value:   call.Value,
					})
					require.NoError(t, err)
					continue
				}

				if call.Type == "context" {
					err := reporting.ReportContext(call.Slug, &call.Context)
					require.NoError(t, err)
					continue
				}
				t.Errorf("Unexpected call type: %v", call.Type)
			}

			err := reporting.SendReport()
			require.NoError(t, err)

			if callsCount != 1 {
				t.Errorf("Expected 1 call, got %d", callsCount)
			}
		})
	}
}
