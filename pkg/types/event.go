package types

import (
	"errors"
	"fmt"
	"strings"

	"github.com/codefresh-io/cronus/pkg/cronexp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/robfig/cron.v2"
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
		GetDBStats() (int, error)
	}
)

// ErrEventNotFound error when cron event not found
var ErrEventNotFound = errors.New("cron event not found")

var commonHelp = `Cronus cron event provider triggers Codefresh pipeline execution, following cron expression.
Supported cron expression syntax:
https://github.com/codefresh-io/cronus/docs/blob/master/expression.md`

// GetURI get cron event unique key for store, in form {cron-expression}:{message}
func GetURI(e Event) string {
	return fmt.Sprintf("cron:codefresh:%s:%s", e.Expression, e.Message)
}

// ConstructEvent convert construct event from store key
func ConstructEvent(uri string, secret string, cronguru cronexp.Service) (*Event, error) {
	s := strings.Split(uri, ":")
	if len(s) != 4 {
		return nil, errors.New("bad cron event uri")
	}
	if s[0] != "cron" || s[1] != "codefresh" {
		return nil, errors.New("bad cron event uri, wrong type or kind")
	}
	// validate expression
	expression := s[2]
	if _, err := cron.Parse(expression); err != nil {
		return nil, err
	}
	// get message
	message := s[3]
	// get cron expression descriptor
	description, err := cronguru.DescribeCronExpression(expression)
	if err != nil {
		log.WithError(err).Warn("failed to get cron expression description")
		description = "failed to get cron description"
	}
	// set status to active
	status := "active"
	// set help string
	help := commonHelp
	return &Event{
		Expression:  expression,
		Message:     message,
		Secret:      secret,
		Description: description,
		Status:      status,
		Help:        help,
	}, nil
}
