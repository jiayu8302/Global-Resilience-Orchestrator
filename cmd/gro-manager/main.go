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
	"gopkg.in/yaml.v3"
)

// Config represents the root configuration structure
type Config struct {
	PanicThreshold     float64      `yaml:"panic_threshold"`
	CheckInterval      string       `yaml:"check_interval"` // Parsed as string then converted to time.Duration
	EvaluationInterval string       `yaml:"evaluation_interval"`
	Regions            []api.Region `yaml:"regions"`
}

type MockActuator struct{}

func (m *MockActuator) ApplyRoutingChange(ctx context.Context, update api.RoutingUpdate) error {
	slog.Info("Executing traffic shift",
		"target_region", update.TargetRegionID,
		"reason", update.ActionReason)
	return nil
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Load configuration from YAML
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Parse durations from strings
	checkDur, _ := time.ParseDuration(cfg.CheckInterval)
	evalDur, _ := time.ParseDuration(cfg.EvaluationInterval)

	slog.Info("GRO Control Plane initialized from config",
		"regions_loaded", len(cfg.Regions),
		"panic_threshold", cfg.PanicThreshold)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize orchestration components
	monEngine := monitor.NewMonitoringEngine()
	steeringStrategy := strategy.NewWeightedLatencyStrategy()
	panicProtector := &strategy.PanicProtector{
		Config: strategy.PanicThresholdConfig{MaxFailurePercentage: cfg.PanicThreshold},
	}
	actuator := &MockActuator{}

	// 3. Start Observe Phase (Monitoring)
	go monEngine.RunBackgroundMonitor(ctx, cfg.Regions, checkDur)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(evalDur)
	defer ticker.Stop()

	var lastActiveRegionID string

	for {
		select {
		case <-sigChan:
			slog.Warn("Termination signal received")
			return
		case <-ticker.C:
			currentRegions := monEngine.GetLatestState()
			if len(currentRegions) == 0 {
				continue
			}

			// Safety Gate
			if err := panicProtector.ValidateSystemHealth(currentRegions); err != nil {
				slog.Error("SYSTEM PANIC", "details", err.Error())
				continue
			}

			// Intelligence Phase
			best, err := steeringStrategy.SelectOptimalRegion(ctx, currentRegions)
			if err != nil {
				slog.Warn("Strategy evaluation failed", "error", err)
				continue
			}

			// State Check
			if best.ID == lastActiveRegionID {
				continue
			}

			// Act Phase
			update := api.RoutingUpdate{
				TargetRegionID: best.ID,
				TrafficWeight:  100,
				ActionReason:   "Optimal resilience target identified via config-driven topology",
			}

			if err := actuator.ApplyRoutingChange(ctx, update); err == nil {
				lastActiveRegionID = best.ID
				slog.Info("Failover target reached", "region", best.ID)
			}
		}
	}
}
