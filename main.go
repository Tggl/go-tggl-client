package tggl

import "net/http"

// Context is the evaluation context for flags
type Context map[string]interface{}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
