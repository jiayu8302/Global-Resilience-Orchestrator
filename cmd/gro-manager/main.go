package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jiayu8302/global-resilience-orchestrator/pkg/api"
	"github.com/jiayu8302/global-resilience-orchestrator/pkg/monitor"
	"github.com/jiayu8302/global-resilience-orchestrator/pkg/strategy"
)

type MockActuator struct{}

func (m *MockActuator) ApplyRoutingChange(ctx context.Context, update api.RoutingUpdate) error {
	log.Printf("[ACTUATOR] ⚙️ Applying traffic shift: Steering 100%% traffic to [%s]. Reason: %s",
		update.TargetRegionID, update.ActionReason)
	return nil
}

func (m *MockActuator) GetCurrentRoutingState(ctx context.Context) (*api.RoutingUpdate, error) {
}

func main() {
	log.Println("🚀 Initializing Global Resilience Orchestrator (GRO) Control Plane...")

	// 1. Define initial infrastructure topology
	initialRegions := []api.Region{
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize orchestration components
	monEngine := monitor.NewMonitoringEngine()
	steeringStrategy := strategy.NewWeightedLatencyStrategy()
	panicProtector := &strategy.PanicProtector{
		Config: strategy.PanicThresholdConfig{MaxFailurePercentage: 0.6},
	}
	actuator := &MockActuator{}

	go monEngine.RunBackgroundMonitor(ctx, initialRegions, 10*time.Second)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 5. Main Control Loop (Analyze & Act)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	log.Println("✅ GRO is active and monitoring global resilience...")

	for {
		select {
		case <-sigChan:
			log.Println("Terminating GRO gracefully...")
			return
		case <-ticker.C:
			currentRegions := monEngine.GetLatestState()
			if len(currentRegions) == 0 {
				continue
			}

			if err := panicProtector.ValidateSystemHealth(currentRegions); err != nil {
				log.Printf("❌ Critical Alert: %v. Automated steering suspended.", err)
				continue
			}

			best, err := steeringStrategy.SelectOptimalRegion(ctx, currentRegions)
			if err != nil {
				log.Printf("⚠️ Steering Error: %v", err)
				continue
			}

			update := api.RoutingUpdate{
				TargetRegionID: best.ID,
				TrafficWeight:  100,
			}
			actuator.ApplyRoutingChange(ctx, update)
		}
	}
}
