package api

import (
	"context"
)

// RoutingUpdate encapsulated the decision made by the Strategy Engine.
type RoutingUpdate struct {
	TargetRegionID string `json:"target_region_id"`
	TrafficWeight  int    `json:"traffic_weight"` // 0-100
	ActionReason   string `json:"action_reason"`  // e.g., "Primary Region Overloaded"
}

// Actuator is the abstraction layer for applying traffic steering rules.
// By defining this as an interface, GRO remains vendor-agnostic.
type Actuator interface {
	// ApplyRoutingChange pushes the new traffic configuration to the data plane.
	ApplyRoutingChange(ctx context.Context, update RoutingUpdate) error

	// GetCurrentRoutingState retrieves what's currently active in the cloud/DNS.
	GetCurrentRoutingState(ctx context.Context) (*RoutingUpdate, error)
}
