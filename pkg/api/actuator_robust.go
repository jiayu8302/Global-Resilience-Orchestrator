package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// RobustActuator wraps a base Actuator with retry logic and telemetry.
type RobustActuator struct {
	Base       Actuator
	MaxRetries int
	RetryDelay time.Duration
}

func (ra *RobustActuator) ApplyRoutingChange(ctx context.Context, update RoutingUpdate) error {
	var lastErr error

	for i := 0; i < ra.MaxRetries; i++ {
		err := ra.Base.ApplyRoutingChange(ctx, update)
		if err == nil {
			if i > 0 {
				slog.Info("Actuation succeeded after retry", "attempt", i+1)
			}
			return nil
		}

		lastErr = err
		slog.Warn("Actuation failed, retrying...", "attempt", i+1, "error", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(ra.RetryDelay):
			continue
		}
	}

	return fmt.Errorf("actuation failed after %d attempts: %w", ra.MaxRetries, lastErr)
}

func (ra *RobustActuator) GetCurrentRoutingState(ctx context.Context) (*RoutingUpdate, error) {
	return ra.Base.GetCurrentRoutingState(ctx)
}
