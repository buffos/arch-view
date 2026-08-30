package live

import (
	"context"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/quality"
	qualityadapter "github.com/buffo/arch-view/internal/quality/adapter"
)

// MultiAnalyzerScanner is the default scanner for a live session. It uses the
// existing planner, bounded scheduler, and aggregate normalizer; it contains
// no language-specific branch.
type MultiAnalyzerScanner struct {
	Host           *analysis.Host
	QualityCatalog *quality.Catalog
	Profiles       QualityProfileResolver
	Cache          *orchestration.SessionCache
}

func NewMultiAnalyzerScanner(host *analysis.Host, catalog *quality.Catalog, profiles QualityProfileResolver) *MultiAnalyzerScanner {
	return &MultiAnalyzerScanner{Host: host, QualityCatalog: catalog, Profiles: profiles, Cache: orchestration.NewSessionCache()}
}

func (scanner *MultiAnalyzerScanner) Scan(ctx context.Context, request ScanRequest) (ScanResult, error) {
	if scanner == nil || scanner.Host == nil {
		return ScanResult{}, newLiveError(ErrorSnapshotBuildFailed, "multi-analyzer scanner is not initialized", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	registry := scanner.Host.RegistrySnapshot()
	policy := orchestration.SourceScopePolicy{PolicyVersion: orchestration.SourceScopePolicyVersion, InvocationRoot: ".", Exclude: []string{}, Include: []orchestration.AnalyzerIncludeRule{}}
	planner := orchestration.NewAnalyzerJobPlanner(registry)
	plan, err := planner.PlanAnalyzerJobs(ctx, orchestration.PlanRequest{
		RepositoryRoot:    request.RepositoryRoot,
		InvocationRoot:    ".",
		SourceScopePolicy: policy,
		Runtime:           scanner.Host.Runtime(),
	})
	if err != nil {
		return ScanResult{}, err
	}
	if len(request.Config.AnalyzerIDs) > 0 {
		allowed := make(map[string]struct{}, len(request.Config.AnalyzerIDs))
		for _, id := range request.Config.AnalyzerIDs {
			allowed[id] = struct{}{}
		}
		filtered := make([]orchestration.AnalyzerJob, 0, len(plan.Jobs))
		for _, job := range plan.Jobs {
			if _, ok := allowed[job.LogicalAnalyzerID]; ok {
				filtered = append(filtered, job)
			}
		}
		plan.Jobs = filtered
	}
	workerCount := request.Config.WatchPolicy.MaxParallelScopes
	if workerCount < 1 {
		workerCount = DefaultMaxParallelScopes
	}
	scheduler := orchestration.NewAnalyzerJobScheduler(func(runContext context.Context, job orchestration.AnalyzerJob) (analysis.AnalysisResult, error) {
		sourceIndexRequest := analysis.SourceIndexRequest{
			Enabled:      request.Config.SourceIndexRequest.Enabled,
			Capabilities: append([]string(nil), request.Config.SourceIndexRequest.Capabilities...),
		}
		return scanner.Host.RunPlanned(runContext, analysis.PlannedRunRequest{
			ProjectRoot:        job.ProjectRoot,
			AnalyzerID:         job.LogicalAnalyzerID,
			Selection:          job.Selection,
			Options:            job.Options,
			SourceScope:        &job.EffectiveSourceScope,
			SourceIndexRequest: &sourceIndexRequest,
		})
	}, orchestration.SchedulerOptions{WorkerCount: workerCount})
	// A live scan is authoritative for the fingerprint captured immediately
	// before it. The existing cache is safe only behind the planner's explicit
	// source-set identity and a stable before/after fingerprint. A conservative
	// full rescan resets it first so a changed scope cannot leave stale
	// cross-scope relationships in an apparently unchanged cache entry.
	if scanner.Cache != nil && request.Invalidation.Mode == "full_rescan" {
		scanner.Cache.Reset()
	}
	var execution orchestration.ExecutionSnapshot
	if scanner.Cache != nil {
		execution = scanner.Cache.Execute(ctx, scheduler, plan)
	} else {
		execution = scheduler.ExecuteAnalyzerPlan(ctx, plan)
	}
	run, err := orchestration.AggregateScopeResults(execution)
	if err != nil {
		return ScanResult{}, err
	}
	if run.Model == nil {
		return ScanResult{}, newLiveError(ErrorSnapshotBuildFailed, "multi-analyzer scan produced no usable model", map[string]any{"status": run.Status, "scopes": len(run.Scopes)})
	}
	canonicalModel, ok := run.CombinedCanonicalModel()
	if !ok {
		return ScanResult{}, newLiveError(ErrorSnapshotBuildFailed, "multi-analyzer scan produced no canonical model", nil)
	}
	result := ScanResult{Run: run, Model: canonicalModel, Diagnostics: []LiveDiagnostic{}}
	for _, diagnostic := range run.Diagnostics {
		result.Diagnostics = append(result.Diagnostics, LiveDiagnostic{Code: diagnostic.Code, Message: diagnostic.Message, Severity: diagnostic.Severity, Path: diagnostic.Path, Recoverable: diagnostic.Recoverable, Details: diagnostic.Metadata})
	}
	if request.Config.QualityRequest != nil {
		if scanner.Profiles == nil || scanner.QualityCatalog == nil {
			return ScanResult{}, newLiveError(ErrorSnapshotBuildFailed, "quality profile evaluation was requested but its catalog or resolver is unavailable", nil)
		}
		profile, profileErr := scanner.Profiles.ResolveProfile(ctx, request.Config.QualityRequest.ProfileID, request.Config.QualityRequest.ProfileVersion)
		if profileErr != nil {
			return ScanResult{}, newLiveError(ErrorSnapshotBuildFailed, "quality profile could not be resolved for the initial scan", map[string]any{"error": profileErr.Error()})
		}
		canonicalModel, ok = run.CombinedCanonicalModel()
		if !ok {
			return ScanResult{}, newLiveError(ErrorSnapshotBuildFailed, "quality evaluation has no canonical aggregate model", nil)
		}
		report, qualityErr := qualityadapter.EvaluateModel(profile, canonicalModel, scanner.QualityCatalog)
		if qualityErr != nil {
			return ScanResult{}, newLiveError(ErrorSnapshotBuildFailed, "quality evaluation failed during the initial scan", map[string]any{"error": qualityErr.Error()})
		}
		if err := run.AttachQualityReport(report); err != nil {
			return ScanResult{}, newLiveError(ErrorSnapshotValidation, "quality report could not be attached to the aggregate model", map[string]any{"error": err.Error()})
		}
		result.Run = run
		canonicalModel, ok := run.CombinedCanonicalModel()
		if ok {
			result.Model = canonicalModel
		}
		result.QualityReport = &report
	}
	if !request.Config.SourceIndexRequest.Enabled {
		result.Model.SourceIndex = nil
		result.Model.QualityReport = result.QualityReport
	}
	return result, nil
}

// StaticScanner is a test and adapter seam for callers that already have an
// authoritative scan result. It still returns a defensive copy to preserve
// the unpublished-candidate boundary.
type StaticScanner struct {
	Result ScanResult
	Err    error
}

func (scanner StaticScanner) Scan(context.Context, ScanRequest) (ScanResult, error) {
	if scanner.Err != nil {
		return ScanResult{}, scanner.Err
	}
	result := scanner.Result
	result.Diagnostics = append([]LiveDiagnostic(nil), result.Diagnostics...)
	if result.Model.SchemaVersion == "" {
		if canonicalModel, ok := result.Run.CombinedCanonicalModel(); ok {
			result.Model = canonicalModel
		}
	}
	return result, nil
}

func normalizeScannerLanguage(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
