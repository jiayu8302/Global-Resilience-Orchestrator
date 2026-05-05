package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jiayu8302/global-resilience-orchestrator/pkg/api"
	"github.com/jiayu8302/global-resilience-orchestrator/pkg/monitor"
	"github.com/jiayu8302/global-resilience-orchestrator/pkg/strategy"
)

// MockActuator simulates an external infrastructure controller (e.g., Azure Traffic Manager API).
type MockActuator struct{}

// ApplyRoutingChange updates the traffic distribution rules on the simulated cloud provider.
func (m *MockActuator) ApplyRoutingChange(ctx context.Context, update api.RoutingUpdate) error {
	slog.Info("Executing traffic shift",
		"target_region", update.TargetRegionID,
		"reason", update.ActionReason)
	return nil
}

func main() {
	// Initialize structured JSON logging (Industry standard for cloud-native observability)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Initializing GRO Control Plane", "pid", os.Getpid())

	// 1. Define initial infrastructure topology.
	// In production, this would be loaded from a configuration file or a database.
	initialRegions := []api.Region{
		{
			ID:          "azure-us-east",
			Provider:    "Azure",
			Status:      api.StatusHealthy,
			LatencyMs:   42,
			CurrentLoad: 0.45,
		},
		{
			ID:          "azure-us-west",
			Provider:    "Azure",
			Status:      api.StatusHealthy,
			LatencyMs:   85,
			CurrentLoad: 0.25,
		},
		{
			ID:          "aws-eu-central",
			Provider:    "AWS",
			Status:      api.StatusHealthy,
			LatencyMs:   150,
			CurrentLoad: 0.10,
		},
	}

	// 2. Setup context and signal handling for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Initialize core orchestration components.
	monEngine := monitor.NewMonitoringEngine()
	steeringStrategy := strategy.NewWeightedLatencyStrategy()
	panicProtector := &strategy.PanicProtector{
		Config: strategy.PanicThresholdConfig{MaxFailurePercentage: 0.6}, // Halt if > 60% failure
	}
	actuator := &MockActuator{}

	// 4. Start the Observe Phase (Asynchronous health monitoring).
	go monEngine.RunBackgroundMonitor(ctx, initialRegions, 10*time.Second)

	// Listen for termination signals (Ctrl+C or Kill).
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 5. Initialize the Control Loop Ticker.
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Track the last applied state to prevent redundant API calls (Flapping prevention).
	var lastActiveRegionID string

	slog.Info("GRO Control Plane is active. Global fleet monitoring engaged.")

	// 6. Main Control Loop: Analyze & Act
	for {
		select {
		case <-sigChan:
			slog.Warn("Termination signal received. Cleaning up resources...")
			return
		case <-ticker.C:
			// Fetch the latest telemetry snapshot from the monitor engine.
			currentRegions := monEngine.GetLatestState()
			if len(currentRegions) == 0 {
				slog.Debug("Waiting for monitor telemetry to populate...")
				continue
			}

			// A. Safety Check (Analyze Phase)
			// Check if the current global failure rate exceeds the safety threshold.
			if err := panicProtector.ValidateSystemHealth(currentRegions); err != nil {
				slog.Error("SYSTEM PANIC DETECTED", "details", err.Error())
				continue
			}

			// B. Strategy Selection (Analyze Phase)
			// Calculate the optimal region based on real-time latency and capacity.
			best, err := steeringStrategy.SelectOptimalRegion(ctx, currentRegions)
			if err != nil {
				slog.Warn("Strategy evaluation failed", "error", err)
				continue
			}

			// C. State Comparison (Analyze Phase)
			// Only trigger the actuator if the optimal region has changed.
			if best.ID == lastActiveRegionID {
				slog.Debug("Infrastructure state stable", "current_leader", best.ID)
				continue
			}

			// D. Execution (Act Phase)
			update := api.RoutingUpdate{
				TargetRegionID: best.ID,
				TrafficWeight:  100,
				ActionReason:   "Automatic failover: identified superior resilience target",
			}

			if err := actuator.ApplyRoutingChange(ctx, update); err == nil {
				// Successfully updated the data plane; store state.
				lastActiveRegionID = best.ID
				slog.Info("Resilience goal achieved", "active_region", best.ID)
			} else {
				slog.Error("Failed to apply routing change", "error", err)
			}
		}
	}
}
