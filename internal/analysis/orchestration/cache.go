package orchestration

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/buffo/arch-view/internal/analysis"
)

// SessionCache retains immutable terminal scope snapshots for one viewer or
// command session. It is intentionally in-memory; persisted cache storage is
// outside the bounded capability.
type SessionCache struct {
	mu      sync.RWMutex
	entries map[string]AnalyzerJob
}

// NewSessionCache creates an empty session-scoped cache.
func NewSessionCache() *SessionCache {
	return &SessionCache{entries: make(map[string]AnalyzerJob)}
}

// Reset drops every reusable scope snapshot. Live coordinators use this when
// an invalidation plan cannot prove dependency impact is selective; retaining
// an apparently unchanged scope in that case could preserve stale
// cross-scope relationships.
func (cache *SessionCache) Reset() {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	cache.entries = make(map[string]AnalyzerJob)
	cache.mu.Unlock()
}

// ExecuteAnalyzerPlanWithCache applies reusable terminal scope snapshots,
// executes only cache misses, and records the new terminal snapshots.
func ExecuteAnalyzerPlanWithCache(ctx context.Context, scheduler *AnalyzerJobScheduler, plan JobPlan, cache *SessionCache) ExecutionSnapshot {
	plan = cloneJobPlan(plan)
	if cache != nil {
		cache.prepare(&plan)
	}
	var snapshot ExecutionSnapshot
	if scheduler == nil {
		snapshot = (&AnalyzerJobScheduler{}).ExecuteAnalyzerPlan(ctx, plan)
	} else {
		snapshot = scheduler.ExecuteAnalyzerPlan(ctx, plan)
	}
	if cache != nil {
		cache.store(snapshot)
	}
	return snapshot
}

// Execute is the method form used by callers that keep a cache with their
// viewer session.
func (cache *SessionCache) Execute(ctx context.Context, scheduler *AnalyzerJobScheduler, plan JobPlan) ExecutionSnapshot {
	return ExecuteAnalyzerPlanWithCache(ctx, scheduler, plan, cache)
}

func (cache *SessionCache) prepare(plan *JobPlan) {
	if cache == nil || plan == nil {
		return
	}
	cache.mu.RLock()
	deferred := make(map[string]AnalyzerJob, len(cache.entries))
	for key, value := range cache.entries {
		deferred[key] = cloneJob(value)
	}
	cache.mu.RUnlock()

	for index := range plan.Jobs {
		job := &plan.Jobs[index]
		job.CacheHit = false
		job.InvalidationReason = ""
		job.Result = nil
		job.Status = JobPlanned
		if previous, ok := previousCacheEntry(deferred, *job); ok {
			if previous.CacheKey != "" && previous.CacheKey == job.CacheKey && isReusableCacheStatus(previous.Status) {
				job.CacheHit = true
				job.Status = previous.Status
				job.Result = cloneResultPointer(previous.Result)
				job.Diagnostics = append([]analysis.Diagnostic(nil), previous.Diagnostics...)
				for diagnosticIndex := range job.Diagnostics {
					job.Diagnostics[diagnosticIndex] = cloneDiagnostic(job.Diagnostics[diagnosticIndex])
				}
				continue
			}
			job.InvalidationReason = cacheInvalidationReason(previous, *job)
		} else {
			job.InvalidationReason = "cache_miss"
		}
	}
}

func previousCacheEntry(entries map[string]AnalyzerJob, current AnalyzerJob) (AnalyzerJob, bool) {
	if previous, ok := entries[current.ScopeID]; ok {
		return previous, true
	}
	// Changing an assignment's analyzer also changes ScopeID. Match the prior
	// job by project root so the new scope reports assignment_changed instead
	// of losing the causal invalidation behind a generic cache miss.
	if current.SelectionSource != SelectionAssignment {
		hasPriorAssignment := false
		for _, previous := range entries {
			if previous.RelativeProjectRoot == current.RelativeProjectRoot && previous.SelectionSource == SelectionAssignment {
				hasPriorAssignment = true
				break
			}
		}
		if !hasPriorAssignment {
			return AnalyzerJob{}, false
		}
	}
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		previous := entries[key]
		if previous.RelativeProjectRoot != current.RelativeProjectRoot {
			continue
		}
		if previous.SelectionSource == SelectionAssignment || current.SelectionSource == SelectionAssignment {
			return previous, true
		}
	}
	return AnalyzerJob{}, false
}

func (cache *SessionCache) store(snapshot ExecutionSnapshot) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.entries == nil {
		cache.entries = make(map[string]AnalyzerJob)
	}
	for _, job := range snapshot.Jobs {
		if job.ScopeID == "" || job.CacheKey == "" || !isReusableCacheStatus(job.Status) {
			continue
		}
		stored := cloneJob(job)
		stored.CacheHit = false
		stored.InvalidationReason = ""
		cache.entries[job.ScopeID] = stored
	}
}

func isReusableCacheStatus(status JobStatus) bool {
	return status == JobComplete || status == JobPartial || status == JobFailed || status == JobSkipped
}

func cloneResultPointer(value *analysis.AnalysisResult) *analysis.AnalysisResult {
	if value == nil {
		return nil
	}
	result := cloneAnalysisResult(*value)
	return &result
}

func cacheInvalidationReason(previous, current AnalyzerJob) string {
	if previous.SelectionSource != current.SelectionSource || previous.AssignmentPath != current.AssignmentPath {
		if previous.SelectionSource == SelectionAssignment || current.SelectionSource == SelectionAssignment || previous.AssignmentPath != "" || current.AssignmentPath != "" {
			return "assignment_changed"
		}
	}
	if previous.LogicalAnalyzerID != current.LogicalAnalyzerID && (previous.SelectionSource == SelectionAssignment || current.SelectionSource == SelectionAssignment) {
		return "assignment_changed"
	}
	if previous.EffectiveOptionsFingerprint != current.EffectiveOptionsFingerprint {
		return "options_changed"
	}
	if previous.MatchedSourceSetFingerprint != current.MatchedSourceSetFingerprint {
		if sameStrings(previous.EffectiveSourceScope.MatchedPaths, current.EffectiveSourceScope.MatchedPaths) {
			return "source_content_changed"
		}
		return "filter_changed"
	}
	if previous.SourceScopeFingerprint != current.SourceScopeFingerprint {
		return "filter_changed"
	}
	if previous.LogicalAnalyzerID != current.LogicalAnalyzerID || previous.AnalyzerVersion != current.AnalyzerVersion || previous.Manifest.APIVersion != current.Manifest.APIVersion || previous.Manifest.RuntimeIdentity != current.Manifest.RuntimeIdentity {
		return "analyzer_changed"
	}
	if previous.RuntimeMode != current.RuntimeMode || previous.RuntimeSource != current.RuntimeSource || previous.RuntimePlatform != current.RuntimePlatform {
		return "runtime_changed"
	}
	if previous.RelativeProjectRoot != current.RelativeProjectRoot || previous.ProjectRoot != current.ProjectRoot {
		return "project_root_changed"
	}
	return "inputs_changed"
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if strings.TrimSpace(left[index]) != strings.TrimSpace(right[index]) {
			return false
		}
	}
	return true
}
