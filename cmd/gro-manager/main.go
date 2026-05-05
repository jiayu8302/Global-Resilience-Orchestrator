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
	"github.com/jiayu8302/global-resilience-orchestrator/pkg/strategy"
	"gopkg.in/yaml.v3"
)

// ... [Keep Config and MockActuator structs as they were] ...

type Config struct {
	PanicThreshold     float64      `yaml:"panic_threshold"`
	CheckInterval      string       `yaml:"check_interval"`
	EvaluationInterval string       `yaml:"evaluation_interval"`
	Regions            []api.Region `yaml:"regions"`
}

type MockActuator struct{}

func (m *MockActuator) ApplyRoutingChange(ctx context.Context, update api.RoutingUpdate) error {
	slog.Info("Executing traffic shift", "target", update.TargetRegionID)
	return nil
}

func (m *MockActuator) GetCurrentRoutingState(ctx context.Context) (*api.RoutingUpdate, error) {
	return &api.RoutingUpdate{TargetRegionID: "azure-us-east", TrafficWeight: 100}, nil
}

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

func startInternalAPI() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("UP")) // Ignoring byte count error is standard here
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// FIX 1: Handled JSON encoding error
		if err := json.NewEncoder(w).Encode(observability.DefaultMetrics); err != nil {
			slog.Error("Failed to encode metrics", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	slog.Info("Internal API listening on :8080")
	// FIX 2: Handled HTTP server startup error (it returns error if it fails to bind)
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil && err != http.ErrServerClosed {
			slog.Error("Internal API server failed", "error", err)
			os.Exit(1)
		}
	}()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := loadConfig("config.yaml")
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	startInternalAPI()

	// FIX 3: Handled time duration parsing errors (invalid strings in YAML)
	checkDur, err := time.ParseDuration(cfg.CheckInterval)
	if err != nil {
		slog.Error("Invalid check_interval in config", "val", cfg.CheckInterval, "error", err)
		os.Exit(1)
	}

	evalDur, err := time.ParseDuration(cfg.EvaluationInterval)
	if err != nil {
		slog.Error("Invalid evaluation_interval in config", "val", cfg.EvaluationInterval, "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monEngine := monitor.NewMonitoringEngine()
	steeringStrategy := strategy.NewWeightedLatencyStrategy()
	panicProtector := &strategy.PanicProtector{
		Config: strategy.PanicThresholdConfig{MaxFailurePercentage: cfg.PanicThreshold},
	}

	actuator := &api.RobustActuator{
		Base:       &MockActuator{},
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
	}

	go monEngine.RunBackgroundMonitor(ctx, cfg.Regions, checkDur)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(evalDur)
	defer ticker.Stop()

	var lastActiveRegionID string
	slog.Info("✅ GRO is active and monitoring global resilience...")

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

			if err := panicProtector.ValidateSystemHealth(currentRegions); err != nil {
				observability.DefaultMetrics.RecordPanic()
				slog.Error("CRITICAL: Automation suspended", "details", err)
				continue
			}

			best, err := steeringStrategy.SelectOptimalRegion(ctx, currentRegions)
			if err != nil {
				slog.Warn("Strategy evaluation failed", "error", err)
				continue
			}

			observability.DefaultMetrics.UpdateLoad(best.ID, best.CurrentLoad)

			if best.ID == lastActiveRegionID {
				continue
			}

			update := api.RoutingUpdate{
				TargetRegionID: best.ID,
				TrafficWeight:  100,
				ActionReason:   "Automatic failover: identified superior target",
			}

			if err := actuator.ApplyRoutingChange(ctx, update); err != nil {
				slog.Error("Failed to apply routing change", "error", err)
			} else {
				lastActiveRegionID = best.ID
				observability.DefaultMetrics.RecordFailover()
			}
		}
	}
}
