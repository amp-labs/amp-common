// Package webhook holds shared types describing inbound provider webhook deliveries.
//
// These types are shared across Ampersand repositories: the server populates them when a
// provider webhook arrives (directly or via the Hookdeck gateway), and downstream libraries
// (e.g. the connectors subscribe configs) consume them to build provider-specific verification
// parameters without depending on server-internal types.
package webhook

import "net/http"

// PubsubPayload is the normalized inbound webhook delivery handed to provider-specific webhook
// handling (verification-params builders, event unwrapping). It is a verbatim copy of the
// server's messenger.PubsubPayload — same name, field names, and JSON tags — so payloads
// serialize identically across repositories.
type PubsubPayload struct {
	Message        []byte      `json:"message"`        // Raw event message from provider
	ProjectId      string      `json:"projectId"`      // projectId for the message
	IntegrationId  string      `json:"integrationId"`  // integrationId for the message
	Headers        http.Header `json:"headers"`        // webhook request headers from provider
	URL            string      `json:"url"`            // webhook request URL from provider
	Method         string      `json:"method"`         // webhook request method from provider
	InstallationId string      `json:"installationId"` // installationId for the message
}
