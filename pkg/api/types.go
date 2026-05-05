package api

import (
	"time"
)

// HealthStatus defines the operational state of a cloud region.
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "HEALTHY"
	StatusUnhealthy HealthStatus = "UNHEALTHY"
	StatusDegraded  HealthStatus = "DEGRADED" // Partial failure or high latency
)

// Region represents a geographic cloud deployment (e.g., Azure East US).
type Region struct {
	ID           string       `json:"id" yaml:"id"`
	Provider     string       `json:"provider" yaml:"provider"`
	Status       HealthStatus `json:"status" yaml:"status"`
	LatencyMs    int64        `json:"latency_ms" yaml:"latency_ms"`
	CurrentLoad  float64      `json:"current_load" yaml:"current_load"`
	CapacityUsed float64      `json:"capacity_used" yaml:"capacity_used"`
	LastSeen     time.Time    `json:"last_seen" yaml:"last_seen"`
}

// Endpoint represents a specific service entry point within a region.
type Endpoint struct {
	URL      string `json:"url"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // http, https, tcp
}
