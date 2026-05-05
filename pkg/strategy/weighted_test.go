package strategy

import (
	"context"
	"testing"

	"github.com/jiayu8302/global-resilience-orchestrator/pkg/api"
)

func TestWeightedLatencyStrategy_SelectOptimalRegion(t *testing.T) {
	s := NewWeightedLatencyStrategy()
	ctx := context.Background()

	t.Run("Should pick lowest score (low latency, low load)", func(t *testing.T) {
		regions := []api.Region{
			{ID: "high-latency", Status: api.StatusHealthy, LatencyMs: 200, CurrentLoad: 0.1}, // Score: 220
			{ID: "optimal", Status: api.StatusHealthy, LatencyMs: 50, CurrentLoad: 0.2},       // Score: 60
			{ID: "high-load", Status: api.StatusHealthy, LatencyMs: 40, CurrentLoad: 0.9},     // Should be filtered (load > 0.85)
		}

		best, err := s.SelectOptimalRegion(ctx, regions)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if best.ID != "optimal" {
			t.Errorf("Expected 'optimal', got %s", best.ID)
		}
	})

	t.Run("Should return error if all regions are overloaded", func(t *testing.T) {
		regions := []api.Region{
			{ID: "overloaded", Status: api.StatusHealthy, LatencyMs: 10, CurrentLoad: 0.95},
		}

		_, err := s.SelectOptimalRegion(ctx, regions)
		if err == nil {
			t.Error("Expected an error due to capacity limits, but got none")
		}
	})
}
