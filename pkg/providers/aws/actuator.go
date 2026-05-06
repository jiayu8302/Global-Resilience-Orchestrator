package aws

import (
	"context"
	"log/slog"

	"github.com/jiayu8302/global-resilience-orchestrator/pkg/api"
)

type Actuator struct {
	HostedZoneID string
}

func (a *Actuator) ApplyRoutingChange(ctx context.Context, update api.RoutingUpdate) error {
	slog.Info("[AWS] Modifying Route53 Weighted Record Sets",
		"zone", a.HostedZoneID,
		"target", update.TargetRegionID)

	// Real-world: Use aws-sdk-go-v2/service/route53
	return nil
}

func (a *Actuator) GetCurrentRoutingState(ctx context.Context) (*api.RoutingUpdate, error) {
	return &api.RoutingUpdate{TargetRegionID: "aws-us-west-2", TrafficWeight: 100}, nil
}
