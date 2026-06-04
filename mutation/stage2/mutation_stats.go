package stage2

import (
	"sort"
	"sync"
)

// MutationStats: tracks historical effectiveness of each mutation type.
// Used for coverage-guided mutation ordering (P3-2).
type MutationStats struct {
	mu    sync.Mutex
	stats map[string]*mutationStat
}

type mutationStat struct {
	Name       string
	TotalRuns  int
	BugsFound  int
	ExecErrors int
}

// GlobalMutationStats: shared statistics across all tasks in a session.
var GlobalMutationStats = NewMutationStats()

// NewMutationStats: create a new mutation statistics tracker.
func NewMutationStats() *MutationStats {
	return &MutationStats{
		stats: make(map[string]*mutationStat),
	}
}

// RecordResult: record the outcome of a mutation execution.
func (ms *MutationStats) RecordResult(mutationName string, foundBug bool, execError bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	s, ok := ms.stats[mutationName]
	if !ok {
		s = &mutationStat{Name: mutationName}
		ms.stats[mutationName] = s
	}
	s.TotalRuns++
	if foundBug {
		s.BugsFound++
	}
	if execError {
		s.ExecErrors++
	}
}

// HitRate: returns the bug discovery rate for a mutation type (bugs / totalRuns).
func (ms *MutationStats) HitRate(mutationName string) float64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	s, ok := ms.stats[mutationName]
	if !ok || s.TotalRuns == 0 {
		return 0.0
	}
	return float64(s.BugsFound) / float64(s.TotalRuns)
}

// PrioritizeUnits: sort MutateUnits by historical hit rate (highest first).
// Units with unknown mutation types (no stats) are placed first (exploration priority).
func (ms *MutationStats) PrioritizeUnits(units []*MutateUnit) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	sort.SliceStable(units, func(i, j int) bool {
		// Skip errored units
		if units[i].Err != nil && units[j].Err == nil {
			return false
		}
		if units[i].Err == nil && units[j].Err != nil {
			return true
		}

		si, okI := ms.stats[units[i].Name]
		sj, okJ := ms.stats[units[j].Name]

		// Unknown mutation types go first (exploration)
		if !okI && okJ {
			return true
		}
		if okI && !okJ {
			return false
		}
		if !okI && !okJ {
			return false
		}

		// Known types: sort by hit rate descending
		rateI := float64(si.BugsFound) / float64(max(si.TotalRuns, 1))
		rateJ := float64(sj.BugsFound) / float64(max(sj.TotalRuns, 1))
		if rateI != rateJ {
			return rateI > rateJ
		}

		// Tie-break: fewer exec errors first
		return si.ExecErrors < sj.ExecErrors
	})
}

// PrioritizePgUnits: sort PgMutateUnits by historical hit rate (highest first).
func (ms *MutationStats) PrioritizePgUnits(units []*PgMutateUnit) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	sort.SliceStable(units, func(i, j int) bool {
		if units[i].Err != nil && units[j].Err == nil {
			return false
		}
		if units[i].Err == nil && units[j].Err != nil {
			return true
		}

		si, okI := ms.stats[units[i].Name]
		sj, okJ := ms.stats[units[j].Name]

		if !okI && okJ {
			return true
		}
		if okI && !okJ {
			return false
		}
		if !okI && !okJ {
			return false
		}

		rateI := float64(si.BugsFound) / float64(max(si.TotalRuns, 1))
		rateJ := float64(sj.BugsFound) / float64(max(sj.TotalRuns, 1))
		if rateI != rateJ {
			return rateI > rateJ
		}

		return si.ExecErrors < sj.ExecErrors
	})
}

// GetStats: return a snapshot of all mutation statistics.
func (ms *MutationStats) GetStats() map[string]*mutationStat {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	result := make(map[string]*mutationStat, len(ms.stats))
	for k, v := range ms.stats {
		cp := *v
		result[k] = &cp
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
