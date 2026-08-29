package orchestration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
)

var runSequence atomic.Uint64

// NewAnalyzerJobScheduler creates a bounded scheduler. A worker count above
// the hard cap is reduced to the cap; it can never expand the process or
// goroutine budget beyond the public contract.
func NewAnalyzerJobScheduler(executor AnalyzerExecutor, options SchedulerOptions) *AnalyzerJobScheduler {
	workers := options.WorkerCount
	if workers <= 0 {
		workers = DefaultWorkerCount
	}
	if workers > HardMaxWorkerCount {
		workers = HardMaxWorkerCount
	}
	return &AnalyzerJobScheduler{Executor: executor, WorkerCount: workers}
}

// newRunID creates a fresh execution identity. Scope and job identities stay
// stable across retries, while each run remains independently addressable in
// the aggregate cache.
func newRunID() string {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		fallback := fmt.Sprintf("%d:%d", time.Now().UnixNano(), runSequence.Add(1))
		sum := sha256.Sum256([]byte(fallback))
		copy(value, sum[:len(value)])
	}
	return "run-" + hex.EncodeToString(value)
}

// ExecuteAnalyzerPlan runs each planned job at most once. The plan supplied by
// the caller is never mutated; all lifecycle state lives in the returned
// snapshot.
func (s *AnalyzerJobScheduler) ExecuteAnalyzerPlan(ctx context.Context, plan JobPlan) ExecutionSnapshot {
	if ctx == nil {
		ctx = context.Background()
	}
	runID := newRunID()
	jobs := cloneJobs(plan.Jobs)
	for index := range jobs {
		cached := jobs[index].CacheHit && isTerminal(jobs[index].Status)
		if !cached {
			jobs[index].Status = JobPlanned
			jobs[index].Result = nil
		}
		jobs[index].StartedAt = nil
		jobs[index].FinishedAt = nil
		jobs[index].Diagnostics = append([]analysis.Diagnostic(nil), jobs[index].Diagnostics...)
	}
	planCopy := cloneJobPlan(plan)
	planCopy.Jobs = cloneJobs(jobs)

	var mu sync.Mutex
	events := make([]JobLifecycleEvent, 0, len(jobs)*3+2)
	sequence := 0
	completed := 0
	emit := func(event JobLifecycleEvent) {
		sequence++
		event.SchemaVersion = JobEventSchemaVersion
		event.Sequence = sequence
		event.RunID = runID
		event.CompletedJobs = completed
		event.TotalJobs = len(jobs)
		events = append(events, event)
	}
	emit(JobLifecycleEvent{Type: "run.started"})
	for _, job := range jobs {
		emit(JobLifecycleEvent{Type: "job.planned", JobID: job.JobID, ScopeID: job.ScopeID, Status: JobPlanned})
	}
	for index := range jobs {
		if !jobs[index].CacheHit || !isTerminal(jobs[index].Status) {
			continue
		}
		completed++
		emit(JobLifecycleEvent{Type: "job.cache_hit", JobID: jobs[index].JobID, ScopeID: jobs[index].ScopeID, Status: jobs[index].Status})
	}

	workerCount := DefaultWorkerCount
	if s != nil && s.WorkerCount > 0 {
		workerCount = s.WorkerCount
	}
	if workerCount > HardMaxWorkerCount {
		workerCount = HardMaxWorkerCount
	}
	if workerCount > len(jobs) && len(jobs) > 0 {
		workerCount = len(jobs)
	}
	if workerCount == 0 {
		workerCount = 1
	}

	runContext, cancel := context.WithCancel(ctx)
	if s != nil {
		s.setCancel(cancel)
		defer s.clearCancel(cancel)
	}
	defer cancel()

	for index := range jobs {
		if !isTerminal(jobs[index].Status) {
			jobs[index].Status = JobQueued
		}
	}
	work := make(chan int)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for worker := 0; worker < workerCount; worker++ {
		go func() {
			defer workers.Done()
			for {
				select {
				case <-runContext.Done():
					return
				case index, ok := <-work:
					if !ok {
						return
					}
					s.executeOne(runContext, index, jobs, &mu, &completed, emit)
				}
			}
		}()
	}

	dispatchDone := make(chan struct{})
	go func() {
		defer close(dispatchDone)
		for index := range jobs {
			select {
			case work <- index:
			case <-runContext.Done():
				return
			}
		}
		close(work)
	}()
	<-dispatchDone
	workers.Wait()

	mu.Lock()
	for index := range jobs {
		if isTerminal(jobs[index].Status) {
			continue
		}
		if runContext.Err() != nil || ctx.Err() != nil {
			s.finishCancelledLocked(index, jobs, &completed, emit, "analysis run was cancelled before the job started")
			continue
		}
		// A missing executor is a stable, scoped failure rather than a panic.
		s.finishFailedLocked(index, jobs, &completed, emit, analysis.NewHostError(analysis.ErrHostFailure, "analyzer job executor is unavailable", nil))
	}
	runStatus := executionStatus(jobs, ctx.Err() != nil || runContext.Err() != nil)
	emit(JobLifecycleEvent{Type: "run.completed", Status: runStatus})
	mu.Unlock()

	snapshotJobs := cloneJobs(jobs)
	sort.Slice(snapshotJobs, func(i, j int) bool { return snapshotJobs[i].ScopeID < snapshotJobs[j].ScopeID })
	return ExecutionSnapshot{RunID: runID, Plan: planCopy, Jobs: snapshotJobs, Events: append([]JobLifecycleEvent(nil), events...)}
}

func (s *AnalyzerJobScheduler) executeOne(ctx context.Context, index int, jobs []AnalyzerJob, mu *sync.Mutex, completed *int, emit func(JobLifecycleEvent)) {
	mu.Lock()
	if isTerminal(jobs[index].Status) {
		mu.Unlock()
		return
	}
	if ctx.Err() != nil {
		s.finishCancelledLocked(index, jobs, completed, emit, "analysis run was cancelled before the job started")
		mu.Unlock()
		return
	}
	now := time.Now().UTC()
	jobs[index].Status = JobRunning
	jobs[index].StartedAt = &now
	job := cloneJob(jobs[index])
	emit(JobLifecycleEvent{Type: "job.started", JobID: job.JobID, ScopeID: job.ScopeID, Status: JobRunning})
	if diagnostic := firstBlockingDiagnostic(job.Diagnostics); diagnostic != nil {
		s.finishDiagnosticFailedLocked(index, jobs, completed, emit, *diagnostic)
		mu.Unlock()
		return
	}
	mu.Unlock()

	var result analysis.AnalysisResult
	var err error
	if s == nil || s.Executor == nil {
		err = analysis.NewHostError(analysis.ErrHostFailure, "analyzer job executor is unavailable", nil)
	} else {
		err = func() (callErr error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					callErr = analysis.NewHostError(analysis.ErrAnalyzerFailed, "analyzer job executor panicked", map[string]any{"panic": fmt.Sprintf("%T: %v", recovered, recovered)})
				}
			}()
			result, callErr = s.Executor(ctx, job)
			return callErr
		}()
	}

	mu.Lock()
	defer mu.Unlock()
	if isTerminal(jobs[index].Status) {
		return
	}
	if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || analysis.ErrorCodeOf(err) == analysis.ErrCancelled {
		s.finishCancelledLocked(index, jobs, completed, emit, cancellationMessage(err))
		return
	}
	if err != nil {
		s.finishFailedLocked(index, jobs, completed, emit, err)
		return
	}
	result = filterResultBySourceScope(result, jobs[index].EffectiveSourceScope)
	if jobs[index].Manifest.ID != "" {
		if validationErr := analysis.ValidateAnalysisResult(result, jobs[index].Manifest, jobs[index].ProjectRoot); validationErr != nil {
			s.finishFailedLocked(index, jobs, completed, emit, validationErr)
			return
		}
	}
	resultCopy := result
	jobs[index].Result = &resultCopy
	jobs[index].Diagnostics = mergeDiagnostics(jobs[index].Diagnostics, result.Diagnostics)
	switch result.Status {
	case analysis.StatusComplete:
		jobs[index].Status = JobComplete
	case analysis.StatusPartial:
		jobs[index].Status = JobPartial
	case analysis.StatusCancelled:
		s.finishCancelledLocked(index, jobs, completed, emit, "analyzer returned a cancelled result")
		return
	case analysis.StatusFailed:
		s.finishFailedLocked(index, jobs, completed, emit, analysis.NewHostError(analysis.ErrAnalyzerFailed, "analyzer returned a failed result", nil))
		return
	default:
		s.finishFailedLocked(index, jobs, completed, emit, analysis.NewHostError(analysis.ErrResultInvalid, "analyzer returned an invalid status", map[string]any{"status": result.Status}))
		return
	}
	finishedAt := time.Now().UTC()
	jobs[index].FinishedAt = &finishedAt
	*completed++
	emit(JobLifecycleEvent{Type: "job.completed", JobID: jobs[index].JobID, ScopeID: jobs[index].ScopeID, Status: jobs[index].Status})
}

func firstBlockingDiagnostic(values []analysis.Diagnostic) *analysis.Diagnostic {
	for index := range values {
		if strings.EqualFold(strings.TrimSpace(values[index].Severity), "error") {
			value := values[index]
			return &value
		}
	}
	return nil
}

func (s *AnalyzerJobScheduler) finishDiagnosticFailedLocked(index int, jobs []AnalyzerJob, completed *int, emit func(JobLifecycleEvent), diagnostic analysis.Diagnostic) {
	if isTerminal(jobs[index].Status) {
		return
	}
	jobs[index].Status = JobFailed
	now := time.Now().UTC()
	jobs[index].FinishedAt = &now
	*completed++
	emit(JobLifecycleEvent{Type: "job.failed", JobID: jobs[index].JobID, ScopeID: jobs[index].ScopeID, Status: JobFailed, Diagnostic: &diagnostic})
}

func (s *AnalyzerJobScheduler) finishFailedLocked(index int, jobs []AnalyzerJob, completed *int, emit func(JobLifecycleEvent), err error) {
	if isTerminal(jobs[index].Status) {
		return
	}
	code := analysis.ErrorCodeOf(err)
	if code == analysis.ErrCancelled {
		s.finishCancelledLocked(index, jobs, completed, emit, cancellationMessage(err))
		return
	}
	diagnostic := analysis.Diagnostic{
		Code:        string(code),
		Severity:    "error",
		Message:     errorMessage(err),
		Subject:     jobs[index].LogicalAnalyzerID,
		Path:        jobs[index].RelativeProjectRoot,
		Recoverable: true,
		Metadata: map[string]any{
			"job_id":   jobs[index].JobID,
			"scope_id": jobs[index].ScopeID,
		},
	}
	jobs[index].Diagnostics = mergeDiagnostics(jobs[index].Diagnostics, []analysis.Diagnostic{diagnostic})
	jobs[index].Status = JobFailed
	now := time.Now().UTC()
	jobs[index].FinishedAt = &now
	*completed++
	emit(JobLifecycleEvent{Type: "job.failed", JobID: jobs[index].JobID, ScopeID: jobs[index].ScopeID, Status: JobFailed, Diagnostic: &diagnostic})
}

func (s *AnalyzerJobScheduler) finishCancelledLocked(index int, jobs []AnalyzerJob, completed *int, emit func(JobLifecycleEvent), message string) {
	if isTerminal(jobs[index].Status) {
		return
	}
	diagnostic := analysis.Diagnostic{
		Code:        string(analysis.ErrCancelled),
		Severity:    "warning",
		Message:     message,
		Subject:     jobs[index].LogicalAnalyzerID,
		Path:        jobs[index].RelativeProjectRoot,
		Recoverable: true,
		Metadata: map[string]any{
			"job_id":   jobs[index].JobID,
			"scope_id": jobs[index].ScopeID,
		},
	}
	jobs[index].Diagnostics = mergeDiagnostics(jobs[index].Diagnostics, []analysis.Diagnostic{diagnostic})
	jobs[index].Status = JobCancelled
	now := time.Now().UTC()
	jobs[index].FinishedAt = &now
	*completed++
	emit(JobLifecycleEvent{Type: "job.cancelled", JobID: jobs[index].JobID, ScopeID: jobs[index].ScopeID, Status: JobCancelled, Diagnostic: &diagnostic})
}

func (s *AnalyzerJobScheduler) setCancel(cancel context.CancelFunc) {
	if s == nil {
		return
	}
	// The scheduler is normally used for one active run. Protecting the hook
	// makes an HTTP caller's cancellation safe even if a status reader races it.
	s.cancelMu.Lock()
	s.cancel = cancel
	s.cancelMu.Unlock()
}

func (s *AnalyzerJobScheduler) clearCancel(cancel context.CancelFunc) {
	if s == nil {
		return
	}
	s.cancelMu.Lock()
	if s.cancel != nil {
		s.cancel = nil
	}
	s.cancelMu.Unlock()
}

// CancelAnalyzerPlan requests cancellation of the active plan. Active
// process-backed analyzers receive the cancellation through their context and
// existing process boundary; queued work is terminally marked by the runner.
func (s *AnalyzerJobScheduler) CancelAnalyzerPlan() {
	if s == nil {
		return
	}
	s.cancelMu.Lock()
	cancel := s.cancel
	s.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// PublishJobLifecycle returns a defensive copy suitable for transport.
func PublishJobLifecycle(snapshot ExecutionSnapshot) []JobLifecycleEvent {
	result := append([]JobLifecycleEvent(nil), snapshot.Events...)
	for index := range result {
		if result[index].Diagnostic != nil {
			diagnostic := cloneDiagnostic(*result[index].Diagnostic)
			result[index].Diagnostic = &diagnostic
		}
	}
	return result
}

func executionStatus(jobs []AnalyzerJob, cancelled bool) JobStatus {
	usable := 0
	partial := false
	failed := false
	for _, job := range jobs {
		switch job.Status {
		case JobComplete:
			usable++
		case JobPartial:
			usable++
			partial = true
		case JobFailed, JobSkipped:
			failed = true
		case JobCancelled:
			cancelled = true
		}
	}
	if usable > 0 {
		if partial || failed || cancelled || usable != len(jobs) {
			return JobPartial
		}
		return JobComplete
	}
	if cancelled {
		return JobCancelled
	}
	return JobFailed
}

func isTerminal(status JobStatus) bool {
	return status == JobComplete || status == JobPartial || status == JobFailed || status == JobCancelled || status == JobSkipped
}

func cancellationMessage(err error) string {
	if err == nil {
		return "analysis run was cancelled"
	}
	return "analysis job was cancelled: " + err.Error()
}

func errorMessage(err error) string {
	if err == nil {
		return "analyzer job failed"
	}
	return err.Error()
}

func mergeDiagnostics(left, right []analysis.Diagnostic) []analysis.Diagnostic {
	all := append(append([]analysis.Diagnostic(nil), left...), right...)
	unique := make(map[string]analysis.Diagnostic, len(all))
	for _, diagnostic := range all {
		data, _ := json.Marshal(diagnostic)
		unique[string(data)] = diagnostic
	}
	result := make([]analysis.Diagnostic, 0, len(unique))
	for _, diagnostic := range unique {
		result = append(result, diagnostic)
	}
	sortDiagnostics(result)
	return result
}

func cloneJobs(values []AnalyzerJob) []AnalyzerJob {
	result := make([]AnalyzerJob, len(values))
	for index, value := range values {
		result[index] = cloneJob(value)
	}
	return result
}

func cloneJobPlan(value JobPlan) JobPlan {
	value.SourceScopePolicy.Exclude = append([]string(nil), value.SourceScopePolicy.Exclude...)
	value.SourceScopePolicy.Include = append([]AnalyzerIncludeRule(nil), value.SourceScopePolicy.Include...)
	for index := range value.SourceScopePolicy.Include {
		value.SourceScopePolicy.Include[index].Globs = append([]string(nil), value.SourceScopePolicy.Include[index].Globs...)
	}
	value.Jobs = cloneJobs(value.Jobs)
	value.DiscoveryDiagnostics = append([]analysis.Diagnostic(nil), value.DiscoveryDiagnostics...)
	for index := range value.DiscoveryDiagnostics {
		value.DiscoveryDiagnostics[index] = cloneDiagnostic(value.DiscoveryDiagnostics[index])
	}
	return value
}

func cloneDiagnostic(value analysis.Diagnostic) analysis.Diagnostic {
	value.Location = clonePosition(value.Location)
	value.Metadata = cloneAnyMap(value.Metadata)
	return value
}

func cloneJob(value AnalyzerJob) AnalyzerJob {
	value.NestedRootExclusions = append([]string(nil), value.NestedRootExclusions...)
	value.Selection.MatchedMarkers = append([]string(nil), value.Selection.MatchedMarkers...)
	value.Options = cloneEffectiveOptions(value.Options)
	value.EffectiveSourceScope.NestedRootExclusions = append([]string(nil), value.EffectiveSourceScope.NestedRootExclusions...)
	value.EffectiveSourceScope.IncludeGlobs = append([]string(nil), value.EffectiveSourceScope.IncludeGlobs...)
	value.EffectiveSourceScope.ExcludeGlobs = append([]string(nil), value.EffectiveSourceScope.ExcludeGlobs...)
	value.EffectiveSourceScope.MatchedPaths = append([]string(nil), value.EffectiveSourceScope.MatchedPaths...)
	value.EffectiveSourceScope.MatchedLocalPaths = append([]string(nil), value.EffectiveSourceScope.MatchedLocalPaths...)
	value.EffectiveSourceScope.ExcludedPaths = append([]string(nil), value.EffectiveSourceScope.ExcludedPaths...)
	value.Diagnostics = append([]analysis.Diagnostic(nil), value.Diagnostics...)
	for index := range value.Diagnostics {
		value.Diagnostics[index] = cloneDiagnostic(value.Diagnostics[index])
	}
	value.Manifest = cloneManifest(value.Manifest)
	if value.Result != nil {
		result := cloneAnalysisResult(*value.Result)
		value.Result = &result
	}
	return value
}

func cloneManifest(value analysis.Manifest) analysis.Manifest {
	value.DetectionMarkers = append([]analysis.DetectionMarker(nil), value.DetectionMarkers...)
	value.Capabilities = append([]string(nil), value.Capabilities...)
	value.Options = append([]analysis.OptionDescriptor(nil), value.Options...)
	for index := range value.Options {
		value.Options[index].Default = cloneOptionValue(value.Options[index].Default)
		value.Options[index].AllowedValues = append([]string(nil), value.Options[index].AllowedValues...)
	}
	return value
}
