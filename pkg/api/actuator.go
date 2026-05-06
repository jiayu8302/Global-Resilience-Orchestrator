package api

import (
	"context"
	"log/slog"
)

// Actuator is the interface for all cloud providers.
type Actuator interface {
	ApplyRoutingChange(ctx context.Context, update RoutingUpdate) error
	GetCurrentRoutingState(ctx context.Context) (*RoutingUpdate, error)
}

// MockActuator provides a no-op implementation for testing and local dev.
// This MUST be in the pkg/api package to be called as api.MockActuator.
type MockActuator struct{}

func (m *MockActuator) ApplyRoutingChange(ctx context.Context, update RoutingUpdate) error {
	slog.Info("[MOCK] Routing update applied",
		"target", update.TargetRegionID,
		"reason", update.ActionReason)
	return nil
}

func (m *MockActuator) GetCurrentRoutingState(ctx context.Context) (*RoutingUpdate, error) {
	return &RoutingUpdate{TargetRegionID: "mock-region-1", TrafficWeight: 100}, nil
}
