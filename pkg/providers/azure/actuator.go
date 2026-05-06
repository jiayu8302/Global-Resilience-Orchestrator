package azure

import (
	"context"
	"log/slog"

	"github.com/jiayu8302/global-resilience-orchestrator/pkg/api"
)

type Actuator struct {
	ResourceGroup string
	ProfileName   string
}

func (a *Actuator) ApplyRoutingChange(ctx context.Context, update api.RoutingUpdate) error {
	slog.Info("[Azure] Shifting traffic via Front Door",
		"profile", a.ProfileName,
		"target", update.TargetRegionID)

	// Real-world: Use azure-sdk-for-go/sdk/resourcemanager/frontdoor/armfrontdoor
	return nil
}

func (a *Actuator) GetCurrentRoutingState(ctx context.Context) (*api.RoutingUpdate, error) {
	return &api.RoutingUpdate{TargetRegionID: "azure-us-east", TrafficWeight: 100}, nil
}
