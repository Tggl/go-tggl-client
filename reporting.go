package tggl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
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
	Clients            []ReportClient           `json:"clients,omitempty"`
	ReceivedProperties map[string][]interface{} `json:"receivedProperties,omitempty"`
	ReceivedValues     map[string][][]string    `json:"receivedValues,omitempty"`
}

// Reporting structure to handle reports
type Reporting struct {
	serverAPIKey string
	httpClt      HTTPClient
	clock        Clock
	baseURL      string
	report       *Report
	app          string
	appPrefix    string
}

// NewReporting creates a new instance of Reporting
func NewReporting(serverAPIKey, app, appPrefix string, httpClt HTTPClient, clock Clock) *Reporting {
	r := &Reporting{
		serverAPIKey: serverAPIKey,
		httpClt:      httpClt,
		baseURL:      "https://api.tggl.io",
		report: &Report{
			Clients:            make([]ReportClient, 0),
			ReceivedValues:     make(map[string][][]string),
			ReceivedProperties: make(map[string][]interface{}),
		},
		app:       app,
		appPrefix: appPrefix,
		clock:     clock,
	}
	// _ = r.SendReport()
	return r
}

// SendReport sends the stored report to the Tggl API
func (r *Reporting) SendReport() (err error) {
	var jsonData []byte
	if len(r.report.Clients) != 0 || len(r.report.ReceivedValues) != 0 || len(r.report.ReceivedProperties) != 0 {
		jsonData, err = json.Marshal(r.report)
		if err != nil {
			return fmt.Errorf("error serializing payload: %w", err)
		}
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
	// r.report = &Report{Clients: make([]ReportClient, 0)}

	return nil
}

func (r *Reporting) ReportFlag(slug string, flag *FlagValue) error {
	clientID := ""
	if r.appPrefix != "" && r.app != "" {
		clientID = r.appPrefix + "/" + r.app
	} else if r.app != "" {
		clientID = r.app
	} else if r.appPrefix != "" {
		clientID = r.appPrefix
	}

	clientAlreadyExists := false
	for _, clt := range r.report.Clients {
		if clt.ID == clientID {
			clientAlreadyExists = true
		}
	}
	if !clientAlreadyExists {
		flag.Count += 1
		r.report.Clients = append(r.report.Clients, ReportClient{
			ID: clientID,
			Flags: map[string][]FlagValue{
				slug: {*flag},
			},
		})
		return nil
	}

	for i, clt := range r.report.Clients {
		if clt.ID != clientID {
			continue
		}
		_, flagAlreadyExists := clt.Flags[slug]
		if !flagAlreadyExists {
			flag.Count = 1
			r.report.Clients[i].Flags[slug] = []FlagValue{*flag}
			return nil
		}

		valueAlreadyExists := false
		for idx, v := range r.report.Clients[i].Flags[slug] {
			isValueEqual := false
			_, isMap := v.Value.(interface{})
			if isMap {
				if reflect.DeepEqual(v.Value, flag.Value) {
					isValueEqual = true
				}
			} else {
				if v.Value == flag.Value {
					isValueEqual = true
				}
			}

			if !isValueEqual {
				continue
			}

			isDefaultValueEqual := false
			_, isMap = v.Value.(interface{})
			if isMap {
				if reflect.DeepEqual(v.Default, flag.Default) {
					isDefaultValueEqual = true
				}
			} else {
				if v.Default == flag.Default {
					isDefaultValueEqual = true
				}
			}

			if isDefaultValueEqual {
				r.report.Clients[i].Flags[slug][idx].Count++
				valueAlreadyExists = true
				break
			}
		}
		if !valueAlreadyExists {
			flag.Count = 1
			r.report.Clients[i].Flags[slug] = append(r.report.Clients[i].Flags[slug], *flag)
		}
	}
	return nil
}

func (r *Reporting) ReportContext(slug string, context *Context) error {
	for k, v := range *context {
		_, ok := r.report.ReceivedProperties[k]
		if !ok {
			r.report.ReceivedProperties[k] = []interface{}{r.clock.Now().Unix(), r.clock.Now().Unix()}
		}
		r.report.ReceivedProperties[k][1] = r.clock.Now().Unix()

		valStr, isString := v.(string)
		if !isString || valStr == "" {
			continue
		}
		r.report.ReceivedValues[k] = append(r.report.ReceivedValues[k])
		r.report.ReceivedValues[k] = append(r.report.ReceivedValues[k], []string{valStr})

		// TODO: Implementer le dict de values. les clés qui finissent par ID et Name et ont le meme nom sont catégorisé
		//  dans la même liste
		if strings.HasSuffix(k, "id") {

		}
	}

	return nil
}
