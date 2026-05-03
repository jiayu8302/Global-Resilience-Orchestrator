package monitor

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jiayu8302/global-resilience-orchestrator/pkg/api"
)

// MonitoringEngine manages the state of the global fleet telemetry.
type MonitoringEngine struct {
	checker     *HealthChecker
	stateMutex  sync.RWMutex
	latestState []api.Region
}

func NewMonitoringEngine() *MonitoringEngine {
	return &MonitoringEngine{
		checker: NewHealthChecker(5 * time.Second),
	}
}

// RunBackgroundMonitor periodically refreshes the health status of all tracked regions.
func (e *MonitoringEngine) RunBackgroundMonitor(ctx context.Context, regions []api.Region, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			updated := e.checker.CheckAllRegions(ctx, regions)
			e.updateState(updated)
			log.Printf("[MONITOR] Health state refreshed. Active regions tracked: %d", len(updated))
		}
	}
}

func (e *MonitoringEngine) GetLatestState() []api.Region {
	e.stateMutex.RLock()
	defer e.stateMutex.RUnlock()

	cp := make([]api.Region, len(e.latestState))
	copy(cp, e.latestState)
	return cp
}

func (e *MonitoringEngine) updateState(regions []api.Region) {
	e.stateMutex.Lock()
	defer e.stateMutex.Unlock()
	e.latestState = regions
}
