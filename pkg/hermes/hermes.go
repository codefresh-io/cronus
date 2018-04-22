package hermes

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/dghubble/sling"
	log "github.com/sirupsen/logrus"
)

type (
	// Service Codefresh Service
	Service interface {
		TriggerEvent(eventURI string, event *NormalizedEvent) error
	}

	// APIEndpoint Hermes API endpoint
	APIEndpoint struct {
		endpoint *sling.Sling
	}

	// NormalizedEvent normalized event: {event-uri, original-payload, secret, variables-map}
	NormalizedEvent struct {
		Original  string            `json:"original,omitempty"`
		Secret    string            `json:"secret,omitempty"`
		Variables map[string]string `json:"variables,omitempty"`
	}
)

// NewNormalizedEvent init NormalizedEvent struct
func NewNormalizedEvent() *NormalizedEvent {
	var event NormalizedEvent
	event.Variables = make(map[string]string)
	return &event
}

// NewHermesEndpoint create new Hermes API endpoint from url and API token
func NewHermesEndpoint(url, token string) Service {
	log.WithField("hermes url", url).Debug("binding to Hermes service")
	endpoint := sling.New().Base(url).Set("Authorization", token)
	return &APIEndpoint{endpoint}
}

// TriggerEvent send normalized event to Hermes trigger-manager server
func (api *APIEndpoint) TriggerEvent(eventURI string, event *NormalizedEvent) error {
	log.WithField("event-uri", eventURI).Debug("Triggering event")
	// runs response
	type PipelineRun struct {
		ID    string `json:"id"`
		Error error  `json:"error,omitempty"`
	}
	// hermes error response
	type HermesError struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	// runs placeholder (on successful call)
	runs := new([]PipelineRun)
	// errors placeholder (for failures)
	var hermesErr HermesError

	// invoke hermes trigger
	log.WithFields(log.Fields{
		"secret":   event.Secret,
		"vars":     event.Variables,
		"original": event.Original,
	}).Debug("sending normalized event payload")
	resp, err := api.endpoint.New().Post(fmt.Sprint("run/", url.QueryEscape(eventURI))).BodyJSON(event).Receive(&runs, &hermesErr)
	// ignore EOF JSON parsing error
	if err != nil && err != io.EOF {
		log.WithError(err).WithField("api", "POST /run/").Error("failed to invoke Hermes REST API")
		return err
	}
	if resp.StatusCode >= 400 {
		log.WithField("hermes error", hermesErr).WithField("api", "POST /run/").Error("failed to invoke Hermes REST API")
		return fmt.Errorf("%s: error triggering event '%s'", resp.Status, eventURI)
	}
	// if no triggers - no pipeline links
	if resp.StatusCode == http.StatusNoContent {
		log.WithField("event-uri", eventURI).Debug("no pipeline linked to the event")
	} else {
		log.WithField("event-uri", eventURI).Debug("event successfully triggered")
		log.WithField("runs", runs).Debug("running following pipelines")
	}
	return nil
}
