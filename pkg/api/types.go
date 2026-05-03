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
	ID           string       `json:"id"`
	Provider     string       `json:"provider"` // e.g., "Azure", "AWS"
	Status       HealthStatus `json:"status"`
	LatencyMs    int64        `json:"latency_ms"`    // Real-time network latency
	CurrentLoad  float64      `json:"current_load"`  // 0.0 to 1.0 (1.0 = 100% capacity)
	CapacityUsed float64      `json:"capacity_used"` // Actual throughput vs limit
	LastSeen     time.Time    `json:"last_seen"`     // Last successful health check
}

// Endpoint represents a specific service entry point within a region.
type Endpoint struct {
	URL      string `json:"url"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // http, https, tcp
}
