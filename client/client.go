// Package client provides a Go SDK for external agents to register
// with a Troupe server via HTTP. Agents call Register to join the
// World, Heartbeat to stay alive, and Unregister to leave.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// ErrRegistrationFailed is returned when the server rejects registration.
var ErrRegistrationFailed = errors.New("client: registration failed")

// Client connects an external agent to a Troupe server.
type Client struct {
	serverURL   string
	callbackURL string
	entityID    uint64
	httpClient  *http.Client
}

// New creates a client targeting a Troupe server.
func New(serverURL string) *Client {
	return &Client{
		serverURL:  serverURL,
		httpClient: &http.Client{},
	}
}

// RegisterRequest is sent to POST /admission.
type RegisterRequest struct {
	Role        string   `json:"role"`
	CallbackURL string   `json:"callback_url"`
	Skills      []string `json:"skills,omitempty"`
}

// RegisterResponse is returned from POST /admission.
type RegisterResponse struct {
	EntityID uint64 `json:"entity_id"`
	Status   string `json:"status"`
}

// Register admits the agent into the Troupe World. The callbackURL is
// the agent's A2A endpoint where Troupe will send messages.
func (c *Client) Register(ctx context.Context, role, callbackURL string) (uint64, error) {
	body, err := json.Marshal(RegisterRequest{
		Role:        role,
		CallbackURL: callbackURL,
	})
	if err != nil {
		return 0, fmt.Errorf("client register marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL+"/admission", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("client register request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("client register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: status %d", ErrRegistrationFailed, resp.StatusCode)
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("client register decode: %w", err)
	}

	c.entityID = result.EntityID
	c.callbackURL = callbackURL
	return result.EntityID, nil
}

// EntityID returns the assigned entity ID after registration.
func (c *Client) EntityID() uint64 { return c.entityID }
