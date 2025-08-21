package tggl

import (
	"net/http"
	"time"
)

// Context is the evaluation context for flags
type Context map[string]interface{}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

type MyService struct {
	clock Clock
}

func (s *MyService) DoSomething() time.Time {
	return s.clock.Now()
}
