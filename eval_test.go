package tggl

import (
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed standard_tests.json
var standardTests []byte

// EvalTestCase represents the structure of a test case from the JSON file
type EvalTestCase struct {
	Name     string                 `json:"name"`
	Flag     Flag                   `json:"flag"`
	Context  map[string]interface{} `json:"context"`
	Expected struct {
		Active bool        `json:"active"`
		Value  interface{} `json:"value"`
	} `json:"expected"`
}

func TestEvalFlag(t *testing.T) {
	// Decode test cases
	var testCases []EvalTestCase
	if err := json.Unmarshal(standardTests, &testCases); err != nil {
		t.Fatalf("Error decoding JSON: %v", err)
	}

	// Run each test case
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evalFlag(tc.Flag, tc.Context)

			// Check active state
			if tc.Expected.Active && result != tc.Expected.Value {
				t.Errorf("Value: got %v, want %v", result, tc.Expected.Value)
			}
			if !tc.Expected.Active && result != nil {
				t.Errorf("Value: got %v, want %v", result, nil)
			}
		})
	}
}
