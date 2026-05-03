package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/jiayu8302/global-resilience-orchestrator/pkg/api"
)

type HealthChecker struct {
	Timeout time.Duration
}

func NewHealthChecker(timeout time.Duration) *HealthChecker {
	return &HealthChecker{Timeout: timeout}
}

// CheckAllRegions runs health probes in parallel using a controlled worker pool.
func (hc *HealthChecker) CheckAllRegions(ctx context.Context, regions []api.Region) []api.Region {
	numRegions := len(regions)
	results := make([]api.Region, numRegions)
	jobs := make(chan int, numRegions)
	var wg sync.WaitGroup

	// Control concurrency to prevent resource exhaustion
	const maxWorkers = 10
	workerCount := maxWorkers
	if numRegions < workerCount {
		workerCount = numRegions
	}

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				// Simulate active probing logic
				status := hc.probeEndpoint(ctx, &regions[i])
				regions[i].Status = status
				regions[i].LastSeen = time.Now()
				results[i] = regions[i]
			}
		}()
	}

	for i := 0; i < numRegions; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return results
}

func (hc *HealthChecker) probeEndpoint(ctx context.Context, r *api.Region) api.HealthStatus {
	// Logic for HTTP/TCP health checks would go here.
	// For now, we assume success for the simulation.
	return api.StatusHealthy
}
