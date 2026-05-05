package observability

import (
	"sync"
	"sync/atomic"
)

// Metrics holds the counters for the orchestration lifecycle.
type Metrics struct {
	TotalChecks      uint64
	TotalFailovers   uint64
	PanicEvents      uint64
	ActiveRegionLoad sync.Map // Stores current load per Region ID
}

var (
	// Global instance for simple access across packages
	DefaultMetrics = &Metrics{}
)

func (m *Metrics) RecordCheck()                       { atomic.AddUint64(&m.TotalChecks, 1) }
func (m *Metrics) RecordFailover()                    { atomic.AddUint64(&m.TotalFailovers, 1) }
func (m *Metrics) RecordPanic()                       { atomic.AddUint64(&m.PanicEvents, 1) }
func (m *Metrics) UpdateLoad(id string, load float64) { m.ActiveRegionLoad.Store(id, load) }
