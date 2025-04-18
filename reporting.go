package tggl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// FlagValue represents a value for a specific flag with its count
type FlagValue struct {
	Value   interface{} `json:"value"`
	Default interface{} `json:"default"`
	Count   int         `json:"count"`
}

// ReportClient represents a client with its flags data
type ReportClient struct {
	ID    string                 `json:"id"`
	Flags map[string][]FlagValue `json:"flags"`
}

// Report represents the structure of the report request body
type Report struct {
	Clients []ReportClient `json:"clients"`
}

// Reporting structure to handle reports
type Reporting struct {
	serverAPIKey string
	httpClt      HTTPClient
	baseURL      string
	report       Report
	app          string
	appPrefix    string
}

// NewReporting creates a new instance of Reporting
func NewReporting(serverAPIKey, app, appPrefix string, httpClt HTTPClient) *Reporting {
	r := &Reporting{
		serverAPIKey: serverAPIKey,
		httpClt:      httpClt,
		baseURL:      "https://api.tggl.io",
		report:       Report{Clients: make([]ReportClient, 0)},
		app:          app,
		appPrefix:    appPrefix,
	}
	_ = r.SendReport()
	return r
}

// SendReport sends the stored report to the Tggl API
func (r *Reporting) SendReport() error {
	jsonData, err := json.Marshal(r.report)
	if err != nil {
		return fmt.Errorf("error serializing payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, r.baseURL+"/report", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("X-Tggl-Api-Key", r.serverAPIKey)

	resp, err := r.httpClt.Do(req)
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

	// Clear the report after successful send
	r.report = Report{Clients: make([]ReportClient, 0)}

	return nil
}

func (r *Reporting) ReportFlag(slug string, flag *FlagValue) error {
	// TODO: to implement
	return nil
}

func (r *Reporting) ReportContext(slug string, context *Context) error {
	// TODO: to implement
	return nil
}
