package strategy

import (
	"errors"
	"fmt"

	"github.com/jiayu8302/global-resilience-orchestrator/pkg/api"
)

// PanicThresholdConfig defines the safety boundaries for automated failover.
type PanicThresholdConfig struct {
	MaxFailurePercentage float64 // e.g., 0.5 for 50%
}

// PanicProtector evaluates if the system is in a "Panic" state.
type PanicProtector struct {
	Config PanicThresholdConfig
}

// ValidateSystemHealth checks if too many regions are down simultaneously.
func (p *PanicProtector) ValidateSystemHealth(regions []api.Region) error {
	if len(regions) == 0 {
		return errors.New("infrastructure registry is empty")
	}

	unhealthyCount := 0
	for _, r := range regions {
		if r.Status != api.StatusHealthy {
			unhealthyCount++
		}
	}

	failureRate := float64(unhealthyCount) / float64(len(regions))

	if failureRate >= p.Config.MaxFailurePercentage {
		// This is the "Safety Brake" mentioned in the README.
		return fmt.Errorf("PANIC MODE TRIGGERED: %.2f%% regions are unhealthy (threshold: %.2f%%). Halting automated failover to prevent cascading collapse",
			failureRate*100, p.Config.MaxFailurePercentage*100)
	}

	return nil
}
