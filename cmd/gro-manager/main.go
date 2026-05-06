package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jiayu8302/global-resilience-orchestrator/pkg/api"
	"github.com/jiayu8302/global-resilience-orchestrator/pkg/monitor"
	"github.com/jiayu8302/global-resilience-orchestrator/pkg/observability"
	"github.com/jiayu8302/global-resilience-orchestrator/pkg/providers/aws"
	"github.com/jiayu8302/global-resilience-orchestrator/pkg/providers/azure"
	"github.com/jiayu8302/global-resilience-orchestrator/pkg/strategy"
	"gopkg.in/yaml.v3"
)

// Config matches the structure of config.yaml
type Config struct {
	CloudProvider      string       `yaml:"cloud_provider"`
	PanicThreshold     float64      `yaml:"panic_threshold"`
	CheckInterval      string       `yaml:"check_interval"`
	EvaluationInterval string       `yaml:"evaluation_interval"`
	Regions            []api.Region `yaml:"regions"`
}

// NewActuator is the Factory that selects the cloud implementation at runtime.
func NewActuator(providerType string) (api.Actuator, error) {
	switch providerType {
	case "azure":
		return &azure.Actuator{ResourceGroup: "gro-production-rg", ProfileName: "global-fd"}, nil
	case "aws":
		return &aws.Actuator{HostedZoneID: "Z0987654321"}, nil
	case "mock":
		return &api.MockActuator{}, nil
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", providerType)
	}
}

// loadConfig handles YAML file ingestion with error wrapping.
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}
	return &cfg, nil
}

// startInternalAPI launches the Health and Metrics endpoints.
func startInternalAPI() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("UP"))
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(observability.DefaultMetrics)
	})

	go func() {
		slog.Info("Internal API listening on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil && err != http.ErrServerClosed {
			slog.Error("Metrics API failed", "error", err)
			os.Exit(1)
		}
	}()
}

func main() {
	// 1. Initialization & Config
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	slog.Info("🚀 GRO: Global Resilience Orchestrator Starting...")

	cfg, err := loadConfig("config.yaml")
	if err != nil {
		slog.Error("Bootstrap failed", "error", err)
		os.Exit(1)
	}

	startInternalAPI()

	// 2. Dependency Injection & Timing
	checkDur, _ := time.ParseDuration(cfg.CheckInterval)
	evalDur, _ := time.ParseDuration(cfg.EvaluationInterval)

	// Load Provider via Factory
	baseActuator, err := NewActuator(cfg.CloudProvider)
	if err != nil {
		slog.Error("Provider initialization failed", "error", err)
		os.Exit(1)
	}

	// Wrap in Robust Decorator for retries and fault tolerance
	actuator := &api.RobustActuator{
		Base:       baseActuator,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
	}

	monEngine := monitor.NewMonitoringEngine()
	steeringStrategy := strategy.NewWeightedLatencyStrategy()
	panicProtector := &strategy.PanicProtector{
		Config: strategy.PanicThresholdConfig{MaxFailurePercentage: cfg.PanicThreshold},
	}

	// 3. Orchestration Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background monitoring of global fleet
	go monEngine.RunBackgroundMonitor(ctx, cfg.Regions, checkDur)

	// Graceful Shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(evalDur)
	defer ticker.Stop()

	var lastActiveRegionID string
	slog.Info("✅ GRO Active", "mode", cfg.CloudProvider, "regions_monitored", len(cfg.Regions))

	for {
		select {
		case <-sigChan:
			slog.Warn("Shutting down GRO gracefully...")
			return
		case <-ticker.C:
			observability.DefaultMetrics.RecordCheck()

			currentRegions := monEngine.GetLatestState()
			if len(currentRegions) == 0 {
				continue
			}

			// Safety Gate (Panic Threshold)
			if err := panicProtector.ValidateSystemHealth(currentRegions); err != nil {
				observability.DefaultMetrics.RecordPanic()
				slog.Error("CRITICAL: Global Panic Threshold Breached", "details", err)
				continue
			}

			// Select Optimal Target
			best, err := steeringStrategy.SelectOptimalRegion(ctx, currentRegions)
			if err != nil {
				slog.Warn("No healthy target found", "error", err)
				continue
			}

			observability.DefaultMetrics.UpdateLoad(best.ID, best.CurrentLoad)

			// Only act if a change is actually required (Stability)
			if best.ID == lastActiveRegionID {
				continue
			}

			update := api.RoutingUpdate{
				TargetRegionID: best.ID,
				TrafficWeight:  100,
				ActionReason:   "Failover: superior target identified",
			}

			if err := actuator.ApplyRoutingChange(ctx, update); err == nil {
				lastActiveRegionID = best.ID
				observability.DefaultMetrics.RecordFailover()
			} else {
				slog.Error("Routing update failed", "target", best.ID, "error", err)
			}
		}
	}
}
