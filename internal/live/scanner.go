package live

import (
	"context"
	"path"
	"sort"
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
	Host                *analysis.Host
	QualityCatalog      *quality.Catalog
	Profiles            QualityProfileResolver
	Cache               *orchestration.SessionCache
	AnalyzerOptionsByID map[string]map[string]any
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
	if request.InputFingerprint.ContentFingerprint.Value != "" && len(request.InputFingerprint.Files) == 0 {
		return ScanResult{}, newLiveError(ErrorSnapshotBuildFailed, "configured live roots contain no eligible input files", nil)
	}
	registry := scanner.Host.RegistrySnapshot()
	policy := liveSourceScopePolicy(request.Config, registry, request.InputFingerprint)
	planner := orchestration.NewAnalyzerJobPlanner(registry)
	plan, err := planner.PlanAnalyzerJobs(ctx, orchestration.PlanRequest{
		RepositoryRoot:    request.RepositoryRoot,
		InvocationRoot:    ".",
		SourceScopePolicy: policy,
		CLIOptionsByID:    cloneAnalyzerOptions(scanner.AnalyzerOptionsByID),
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
	// A detected project with no files beneath the configured live roots is
	// outside this session. Keeping it would attach model facts that the
	// session fingerprint can never invalidate.
	filteredJobs := make([]orchestration.AnalyzerJob, 0, len(plan.Jobs))
	for _, job := range plan.Jobs {
		if len(job.EffectiveSourceScope.MatchedLocalPaths) > 0 {
			filteredJobs = append(filteredJobs, job)
		}
	}
	plan.Jobs = filteredJobs
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
		// Policy writes can update a profile while this long-lived session is
		// running. Prefer the persisted document for every re-scan so a later
		// startup or source-triggered rebuild observes an appended baseline or
		// profile change instead of the resolver's startup copy.
		if request.PolicyService != nil {
			if persisted, persistedErr := request.PolicyService.ResolveProfile(ctx, request.Config.QualityRequest.ProfileID, request.Config.QualityRequest.ProfileVersion); persistedErr == nil {
				profile = persisted
			}
		}
		canonicalModel, ok = run.CombinedCanonicalModel()
		if !ok {
			return ScanResult{}, newLiveError(ErrorSnapshotBuildFailed, "quality evaluation has no canonical aggregate model", nil)
		}
		baselineResolution, baselineErr := resolveQualityBaseline(ctx, request.PolicyService, profile, BaselineModeProfile, nil)
		if baselineErr != nil {
			return ScanResult{}, newLiveError(ErrorSnapshotBuildFailed, "quality baseline could not be resolved during the initial scan", map[string]any{"error": baselineErr.Error()})
		}
		if baselineResolution.Warning != "" {
			result.Diagnostics = append(result.Diagnostics, LiveDiagnostic{Code: ErrorBaselineNotFound, Message: baselineResolution.Warning, Severity: "warning", Recoverable: true})
		}
		report, qualityErr := qualityadapter.EvaluateModelWithBaseline(baselineResolution.Profile, canonicalModel, scanner.QualityCatalog, baselineResolution.Baseline)
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

func cloneAnalyzerOptions(values map[string]map[string]any) map[string]map[string]any {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]map[string]any, len(values))
	for analyzerID, options := range values {
		if options == nil {
			result[analyzerID] = nil
			continue
		}
		result[analyzerID] = make(map[string]any, len(options))
		for name, value := range options {
			result[analyzerID][name] = value
		}
	}
	return result
}

func liveSourceScopePolicy(config LiveSessionConfig, registry *analysis.Registry, input InputFingerprint) orchestration.SourceScopePolicy {
	globs := make([]string, 0, len(config.WatchRoots))
	for _, root := range config.WatchRoots {
		value := root.Path
		if root.Recursive {
			if value == "" || value == "." {
				value = "**"
			} else {
				value = path.Join(value, "**")
			}
			globs = append(globs, value)
			continue
		}
		// Source-scope globs intentionally treat a matched directory as a
		// recursive prefix. Enumerate authoritative direct files for a
		// non-recursive watch root so analyzer and fingerprint semantics stay
		// identical.
		rootPath := strings.Trim(path.Clean(value), "/")
		if rootPath == "" {
			rootPath = "."
		}
		for _, file := range input.Files {
			directory := path.Dir(file.Path)
			if directory == rootPath {
				globs = append(globs, file.Path)
			}
		}
	}
	globs = uniqueStrings(globs)
	ids := append([]string(nil), config.AnalyzerIDs...)
	if len(ids) == 0 && registry != nil {
		for _, manifest := range registry.ListManifests() {
			ids = append(ids, manifest.ID)
		}
	}
	sort.Strings(ids)
	include := make([]orchestration.AnalyzerIncludeRule, 0, len(ids))
	for _, id := range ids {
		include = append(include, orchestration.AnalyzerIncludeRule{AnalyzerID: id, Globs: append([]string(nil), globs...)})
	}
	return orchestration.SourceScopePolicy{PolicyVersion: orchestration.SourceScopePolicyVersion, InvocationRoot: ".", Exclude: []string{}, Include: include}
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
