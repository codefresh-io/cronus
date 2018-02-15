package types

import (
	"errors"
	"fmt"
	"strings"
)

type (
	// Event extended cron event
	Event struct {
		// cron expression
		Expression string `json:"expression"`
		// event message
		Message string `json:"message"`
		// event secret
		Secret string `json:"secret"`
		// Description human readable text
		Description string `json:"description,omitempty"`
		// Status current event handler status (active, error, not active)
		Status string `json:"status,omitempty"`
		// Help test
		Help string `json:"help,omitempty"`
	}

	// EventStore job manager interface to add/remove running jobs
	EventStore interface {
		StoreEvent(event Event) error
		DeleteEvent(uri string) error
		GetEvent(uri string) (*Event, error)
		GetAllEvents() ([]Event, error)
	}
)

// GetURI get cron event unique key for store, in form {cron-expression}:{message}
func GetURI(e Event) string {
	return fmt.Sprintf("cron:codefresh:%s:%s", e.Expression, e.Message)
}

// ConstructEvent convert construct event from store key and description
func ConstructEvent(uri string, secret string, description string) (*Event, error) {
	s := strings.Split(uri, ":")
	if len(s) != 4 {
		return nil, errors.New("bad cron event uri")
	}
	if s[0] != "cron" && s[1] != "codefresh" {
		return nil, errors.New("bad cron event uri, wrong type or kind")
	}
	return &Event{
		Expression:  s[2],
		Message:     s[3],
		Secret:      secret,
		Description: description,
	}, nil
}
