package assistant

import (
	"context"
	"errors"
)

// ErrModelUnavailable indicates that the optional model layer is disabled or misconfigured.
var ErrModelUnavailable = errors.New("assistant model is unavailable")

type unavailableModel struct {
	status Status
}

// NewUnavailableModel creates a safe model fallback that never makes network calls.
func NewUnavailableModel(provider, model, reason string) Model {
	if provider == "" {
		provider = "disabled"
	}
	return &unavailableModel{status: Status{
		Available:     false,
		Provider:      provider,
		Model:         model,
		PromptVersion: PromptVersion,
		MaxHistory:    maxHistoryMessages,
		Reason:        reason,
	}}
}

func (m *unavailableModel) Status() Status {
	return m.status
}

func (m *unavailableModel) Respond(context.Context, ModelRequest) (ModelResponse, error) {
	return ModelResponse{}, ErrModelUnavailable
}
