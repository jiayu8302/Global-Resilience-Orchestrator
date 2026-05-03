package strategy

import (
	"context"
	"errors"
	"math"

	"github.com/jiayu8302/global-resilience-orchestrator/pkg/api"
)

// WeightedLatencyStrategy calculates the optimal region using latency and capacity metrics.
type WeightedLatencyStrategy struct{}

func NewWeightedLatencyStrategy() *WeightedLatencyStrategy {
	return &WeightedLatencyStrategy{}
}

// SelectOptimalRegion finds the healthiest region with the lowest 'cost' score.
func (s *WeightedLatencyStrategy) SelectOptimalRegion(ctx context.Context, regions []api.Region) (*api.Region, error) {
	if len(regions) == 0 {
		return nil, errors.New("no regions available for selection")
	}

	var feasibleRegions []api.Region
	const capacityHeadroomLimit = 0.85 // Avoid regions > 85% load

	// Filter for healthy regions with available headroom
	for _, r := range regions {
		if r.Status == api.StatusHealthy && r.CurrentLoad < capacityHeadroomLimit {
			feasibleRegions = append(feasibleRegions, r)
		}
	}

	if len(feasibleRegions) == 0 {
		return nil, errors.New("all regions are either unhealthy or at maximum capacity")
	}

	var bestRegion *api.Region
	minScore := math.MaxFloat64

	for _, r := range feasibleRegions {
		// Scoring logic: Lower score is better.
		// Formula: Latency weighted by current saturation.
		score := float64(r.LatencyMs) * (1.0 + r.CurrentLoad)

		if score < minScore {
			minScore = score
			target := r
			bestRegion = &target
		}
	}

	return bestRegion, nil
}
