package orchestration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
)

type AnalysisAggregationService struct{}

func NewAnalysisAggregationService() *AnalysisAggregationService {
	return &AnalysisAggregationService{}
}

// AggregateScopeResults turns terminal job snapshots into the versioned
// aggregate response. Each scope is normalized independently before the
// namespaced combined graph is normalized, so a failed scope cannot mutate a
// usable scope and no relationship is inferred between scopes.
func AggregateScopeResults(snapshot ExecutionSnapshot) (AnalysisRun, error) {
	return NewAnalysisAggregationService().AggregateScopeResults(snapshot)
}

func (s *AnalysisAggregationService) AggregateScopeResults(snapshot ExecutionSnapshot) (AnalysisRun, error) {
	plan := cloneJobPlan(snapshot.Plan)
	if plan.PlanVersion == "" {
		plan.PlanVersion = JobPlanSchemaVersion
	}
	if plan.PlanVersion != JobPlanSchemaVersion {
		return AnalysisRun{}, analysis.NewHostError(analysis.ErrInvalidRequest, "job plan version is unsupported", map[string]any{"plan_version": plan.PlanVersion})
	}
	runID := snapshot.RunID
	if runID == "" {
		runID = newRunID()
	}
	jobs := cloneJobs(snapshot.Jobs)
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].ScopeID < jobs[j].ScopeID })

	run := AnalysisRun{
		SchemaVersion: AggregateSchemaVersion,
		RunID:         runID,
		Repository:    RepositoryMetadata{RootLabel: repositoryLabel(plan.RepositoryRoot)},
		JobPlan:       plan,
		Scopes:        []ScopeSummary{},
		Diagnostics:   []ScopedDiagnostic{},
		Summary:       AggregateSummary{JobCount: len(jobs)},
		Events:        append([]JobLifecycleEvent(nil), snapshot.Events...),
		scopeModels:   make(map[string]model.Model),
		scopeResults:  make(map[string]analysis.AnalysisResult),
	}

	usableJobs := []AnalyzerJob{}
	normalizedPartial := false
	for index := range jobs {
		job := jobs[index]
		summary := scopeSummary(job)
		result := job.Result
		usable := result != nil && (job.Status == JobComplete || job.Status == JobPartial) && (result.Status == analysis.StatusComplete || result.Status == analysis.StatusPartial)
		if usable {
			value := cloneAnalysisResult(*result)
			if job.Manifest.ID != "" {
				if err := analysis.ValidateAnalysisResult(value, job.Manifest, job.ProjectRoot); err != nil {
					usable = false
					jobDiagnostic := diagnosticForJob(job, err)
					summary.Diagnostics = mergeDiagnostics(summary.Diagnostics, []analysis.Diagnostic{jobDiagnostic})
					summary.Status = JobFailed
					jobs[index].Status = JobFailed
					jobs[index].Diagnostics = mergeDiagnostics(jobs[index].Diagnostics, []analysis.Diagnostic{jobDiagnostic})
				}
			}
			if usable {
				individual, err := canonical.Normalize(value)
				if err != nil {
					usable = false
					jobDiagnostic := analysis.Diagnostic{
						Code:        "aggregate_normalization_failed",
						Severity:    "error",
						Message:     fmt.Sprintf("scope %s could not be normalized: %v", job.ScopeID, err),
						Subject:     job.LogicalAnalyzerID,
						Path:        job.RelativeProjectRoot,
						Recoverable: true,
					}
					summary.Diagnostics = mergeDiagnostics(summary.Diagnostics, []analysis.Diagnostic{jobDiagnostic})
					summary.Status = JobFailed
					jobs[index].Status = JobFailed
					jobs[index].Diagnostics = mergeDiagnostics(jobs[index].Diagnostics, []analysis.Diagnostic{jobDiagnostic})
				} else {
					run.scopeResults[job.ScopeID] = value
					run.scopeModels[job.ScopeID] = individual
					usableJobs = append(usableJobs, job)
					summary.Summary = value.Summary
					if individual.Status == model.StatusPartial && jobs[index].Status == JobComplete {
						normalizedPartial = true
						jobs[index].Status = JobPartial
						summary.Status = JobPartial
					}
				}
			}
		}
		if summary.Summary == (analysis.AnalysisSummary{}) && result != nil {
			summary.Summary = result.Summary
		}
		run.Scopes = append(run.Scopes, summary)
	}
	sort.Slice(run.Scopes, func(i, j int) bool { return run.Scopes[i].ScopeID < run.Scopes[j].ScopeID })
	run.Status = aggregateStatus(jobs, usableJobs, len(plan.DiscoveryDiagnostics) > 0)
	if normalizedPartial && run.Status == analysis.StatusComplete {
		run.Status = analysis.StatusPartial
	}
	run.JobPlan = plan
	run.JobPlan.Jobs = cloneJobs(jobs)
	for index := range run.JobPlan.Jobs {
		run.JobPlan.Jobs[index].Result = nil
	}
	run.Summary.UsableScopeCount = len(usableJobs)
	for _, scope := range run.Scopes {
		if scope.Status == JobFailed || scope.Status == JobCancelled || scope.Status == JobSkipped {
			run.Summary.FailedScopeCount++
		}
	}

	for _, diagnostic := range plan.DiscoveryDiagnostics {
		run.Diagnostics = append(run.Diagnostics, scopedDiagnostic("", diagnostic))
	}
	for _, job := range jobs {
		for _, diagnostic := range job.Diagnostics {
			run.Diagnostics = append(run.Diagnostics, scopedDiagnostic(job.ScopeID, diagnostic))
		}
	}
	sort.Slice(run.Diagnostics, func(i, j int) bool { return run.Diagnostics[i].ID < run.Diagnostics[j].ID })

	if len(usableJobs) > 0 {
		merged, err := mergeUsableResults(run, usableJobs, jobs, plan.DiscoveryDiagnostics)
		if err != nil {
			return AnalysisRun{}, err
		}
		canonicalModel, err := canonical.Normalize(merged)
		if err != nil {
			details := map[string]any{"cause_code": string(analysis.ErrorCodeOf(err))}
			var hostErr *analysis.HostError
			if errors.As(err, &hostErr) {
				details["cause_message"] = hostErr.Message
				if hostErr.Details != nil {
					details["cause_details"] = hostErr.Details
				}
			}
			return AnalysisRun{}, analysis.WrapHostError(analysis.ErrInvalidModel, "aggregate model could not be normalized", err, details)
		}
		run.combinedCanonical = canonicalModel
		run.Model = aggregateModelFromCanonical(canonicalModel, run.Scopes)
	}
	return run, nil
}

// SelectAnalysisScope selects only cached canonical data. It never calls an
// analyzer and therefore remains safe for viewer dropdown changes.
func (r *AnalysisRun) SelectAnalysisScope(scope string) (SelectedScope, error) {
	if r == nil {
		return SelectedScope{}, analysis.NewHostError(analysis.ErrInvalidModel, "analysis run is not initialized", nil)
	}
	scope = strings.TrimSpace(scope)
	if scope == "" || strings.EqualFold(scope, "all") {
		if r.Model == nil || r.combinedCanonical.ModelID == "" {
			return SelectedScope{}, analysis.NewHostError(analysis.ErrInvalidModel, "combined model is unavailable because no usable scope exists", nil)
		}
		projection, err := model.BuildHierarchyProjection(r.combinedCanonical, nil)
		if err != nil {
			return SelectedScope{}, err
		}
		return SelectedScope{RunID: r.RunID, Scope: "all", Model: r.combinedCanonical, Projection: projection}, nil
	}
	value, ok := r.scopeModels[scope]
	if !ok {
		for _, candidate := range r.Scopes {
			if candidate.ScopeID == scope {
				// Failed and cancelled scopes remain selectable for diagnostics,
				// but intentionally do not fabricate a model or projection.
				return SelectedScope{RunID: r.RunID, Scope: scope, Summary: candidate}, nil
			}
		}
		return SelectedScope{}, analysis.NewHostError(analysis.ErrAnalysisScopeNotFound, "analysis scope was not found", map[string]any{"scope_id": scope})
	}
	var summary ScopeSummary
	for _, candidate := range r.Scopes {
		if candidate.ScopeID == scope {
			summary = candidate
			break
		}
	}
	projection, err := model.BuildHierarchyProjection(value, nil)
	if err != nil {
		return SelectedScope{}, err
	}
	return SelectedScope{RunID: r.RunID, Scope: scope, Summary: summary, Model: value, Projection: projection}, nil
}

func (r *AnalysisRun) CombinedCanonicalModel() (model.Model, bool) {
	if r == nil || r.combinedCanonical.ModelID == "" {
		return model.Model{}, false
	}
	return r.combinedCanonical, true
}

// ScopeResult returns the cached analyzer result for a scope. It exists for
// CLI/CI consumers that request a single cached scope after a combined run;
// it never invokes an analyzer.
func (r *AnalysisRun) ScopeResult(scope string) (analysis.AnalysisResult, error) {
	if r == nil {
		return analysis.AnalysisResult{}, analysis.NewHostError(analysis.ErrInvalidModel, "analysis run is not initialized", nil)
	}
	if strings.TrimSpace(scope) == "" || strings.EqualFold(strings.TrimSpace(scope), "all") {
		return analysis.AnalysisResult{}, analysis.NewHostError(analysis.ErrInvalidRequest, "a concrete scope id is required for a cached scope result", nil)
	}
	value, ok := r.scopeResults[scope]
	if !ok {
		return analysis.AnalysisResult{}, analysis.NewHostError(analysis.ErrAnalysisScopeNotFound, "analysis scope was not found", map[string]any{"scope_id": scope})
	}
	return cloneAnalysisResult(value), nil
}

func scopeSummary(job AnalyzerJob) ScopeSummary {
	info := analysis.AnalyzerInfo{
		ID:              job.LogicalAnalyzerID,
		Version:         job.AnalyzerVersion,
		Language:        job.Language,
		APIVersion:      analysis.AnalyzerAPIVersion,
		RuntimeMode:     job.RuntimeMode,
		RuntimeSource:   job.RuntimeSource,
		RuntimePlatform: job.RuntimePlatform,
	}
	if job.Manifest.ID != "" {
		info.ID = job.Manifest.ID
		info.Version = job.Manifest.Version
		info.Language = job.Manifest.Language
		info.APIVersion = job.Manifest.APIVersion
	}
	if job.Result != nil && job.Result.Analyzer.ID != "" {
		info = job.Result.Analyzer
	}
	return ScopeSummary{
		ScopeID:       job.ScopeID,
		ProjectRoot:   job.RelativeProjectRoot,
		Analyzer:      info,
		RuntimeSource: job.RuntimeSource,
		SourceScope: ScopeSourceSummary{
			PolicyFingerprint:           job.EffectiveSourceScope.PolicyFingerprint,
			MatchedSourceSetFingerprint: job.MatchedSourceSetFingerprint,
		},
		Status:                      job.Status,
		SelectionSource:             job.SelectionSource,
		AssignmentPath:              job.AssignmentPath,
		EffectiveOptionsFingerprint: job.EffectiveOptionsFingerprint,
		CacheKey:                    job.CacheKey,
		CacheHit:                    job.CacheHit,
		InvalidationReason:          job.InvalidationReason,
		Summary:                     resultSummary(job.Result),
		Diagnostics:                 append([]analysis.Diagnostic(nil), job.Diagnostics...),
	}
}

func resultSummary(result *analysis.AnalysisResult) analysis.AnalysisSummary {
	if result == nil {
		return analysis.AnalysisSummary{}
	}
	if result.Summary != (analysis.AnalysisSummary{}) {
		return result.Summary
	}
	return analysis.ComputeSummary(*result)
}

func aggregateStatus(jobs, usable []AnalyzerJob, discoveryHadDiagnostics bool) analysis.AnalysisStatus {
	if len(usable) > 0 {
		if len(usable) == len(jobs) && !discoveryHadDiagnostics {
			allComplete := true
			for _, job := range usable {
				if job.Status != JobComplete {
					allComplete = false
					break
				}
			}
			if allComplete {
				return analysis.StatusComplete
			}
		}
		return analysis.StatusPartial
	}
	if len(jobs) > 0 {
		for _, job := range jobs {
			if job.Status == JobFailed || job.Status == JobSkipped {
				return analysis.StatusFailed
			}
		}
		return analysis.StatusCancelled
	}
	return analysis.StatusFailed
}

func mergeUsableResults(run AnalysisRun, usableJobs, allJobs []AnalyzerJob, diagnostics []analysis.Diagnostic) (analysis.AnalysisResult, error) {
	manifest := aggregateManifest()
	merged := analysis.AnalysisResult{
		RunID:    run.RunID,
		Status:   run.Status,
		Analyzer: analysis.AnalyzerInfo{ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion},
		Project:  analysis.ProjectInfo{RootLabel: run.Repository.RootLabel, Boundary: "aggregate"},
		Modules:  []analysis.ModuleObservation{}, Relationships: []analysis.RelationshipObservation{}, References: []analysis.Reference{}, SourceReferences: []analysis.SourceReference{}, Diagnostics: []analysis.Diagnostic{},
	}
	for _, job := range usableJobs {
		result, ok := run.scopeResults[job.ScopeID]
		if !ok {
			continue
		}
		value := namespaceResult(job, result)
		merged.Modules = append(merged.Modules, value.Modules...)
		merged.Relationships = append(merged.Relationships, value.Relationships...)
		merged.References = append(merged.References, value.References...)
		merged.SourceReferences = append(merged.SourceReferences, value.SourceReferences...)
		merged.Diagnostics = append(merged.Diagnostics, value.Diagnostics...)
	}
	for _, job := range allJobs {
		result, hasResult := run.scopeResults[job.ScopeID]
		for _, diagnostic := range job.Diagnostics {
			if hasResult && containsDiagnostic(result.Diagnostics, diagnostic) {
				continue
			}
			merged.Diagnostics = append(merged.Diagnostics, namespaceDiagnostic(job, diagnostic))
		}
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "" {
			continue
		}
		// Scope diagnostics are already included above. Discovery and aggregate
		// failures need one recoverable bridge diagnostic in a partial model.
		if run.Status == analysis.StatusPartial && !containsDiagnostic(merged.Diagnostics, diagnostic) {
			merged.Diagnostics = append(merged.Diagnostics, diagnostic)
		}
	}
	if run.Status == analysis.StatusPartial && len(merged.Diagnostics) == 0 {
		merged.Diagnostics = append(merged.Diagnostics, analysis.Diagnostic{Code: "aggregate_partial", Severity: "warning", Message: "One or more analyzer scopes did not produce a complete result.", Recoverable: true})
	}
	return merged, nil
}

func namespaceResult(job AnalyzerJob, result analysis.AnalysisResult) analysis.AnalysisResult {
	value := cloneAnalysisResult(result)
	value.RunID = job.ScopeID + "::" + result.RunID
	value.Project.RootLabel = job.RelativeProjectRoot
	value.Project.Boundary = result.Project.Boundary
	for index := range value.Modules {
		localID := value.Modules[index].ID
		value.Modules[index].ID = namespacedID(job.ScopeID, localID)
		value.Modules[index].SourceReferenceIDs = namespaceIDs(job.ScopeID, value.Modules[index].SourceReferenceIDs)
		value.Modules[index].Metadata = addScopeMetadata(value.Modules[index].Metadata, job.ScopeID)
	}
	for index := range value.References {
		value.References[index].ID = namespacedID(job.ScopeID, value.References[index].ID)
		value.References[index].Metadata = addScopeMetadata(value.References[index].Metadata, job.ScopeID)
	}
	for index := range value.SourceReferences {
		value.SourceReferences[index].ID = namespacedID(job.ScopeID, value.SourceReferences[index].ID)
		value.SourceReferences[index].Path = repositoryPath(job.RelativeProjectRoot, value.SourceReferences[index].Path)
	}
	for index := range value.Relationships {
		value.Relationships[index].ID = namespacedID(job.ScopeID, value.Relationships[index].ID)
		value.Relationships[index].FromModuleID = namespacedID(job.ScopeID, value.Relationships[index].FromModuleID)
		if value.Relationships[index].ToModuleID != "" {
			value.Relationships[index].ToModuleID = namespacedID(job.ScopeID, value.Relationships[index].ToModuleID)
		}
		if value.Relationships[index].ToReferenceID != "" {
			value.Relationships[index].ToReferenceID = namespacedID(job.ScopeID, value.Relationships[index].ToReferenceID)
		}
		value.Relationships[index].SourceReferenceIDs = namespaceIDs(job.ScopeID, value.Relationships[index].SourceReferenceIDs)
		value.Relationships[index].Metadata = addScopeMetadata(value.Relationships[index].Metadata, job.ScopeID)
	}
	for index := range value.Diagnostics {
		value.Diagnostics[index] = namespaceDiagnostic(job, value.Diagnostics[index])
	}
	return value
}

func namespaceDiagnostic(job AnalyzerJob, diagnostic analysis.Diagnostic) analysis.Diagnostic {
	value := diagnostic
	originalSubject := value.Subject
	value.Subject = job.ScopeID + "::" + originalSubject
	if value.Path != "" {
		pathValue := normalizeRelativePath(value.Path)
		root := normalizeRelativePath(job.RelativeProjectRoot)
		if root != "." && pathValue != root && !strings.HasPrefix(pathValue, root+"/") {
			pathValue = repositoryPath(root, pathValue)
		}
		value.Path = pathValue
	}
	metadata := addScopeMetadata(value.Metadata, job.ScopeID)
	metadata["aggregate_original_subject"] = originalSubject
	value.Metadata = metadata
	return value
}

func aggregateManifest() analysis.Manifest {
	return analysis.Manifest{ID: "org.archview.aggregate", Version: "1.0.0", Language: "mixed", APIVersion: analysis.AnalyzerAPIVersion, DetectionMarkers: []analysis.DetectionMarker{{Kind: "file", Value: "aggregate", Weight: 1}}}
}

func aggregateModelFromCanonical(value model.Model, scopes []ScopeSummary) *AggregateModel {
	result := &AggregateModel{
		SchemaVersion:    AggregateModelSchemaVersion,
		Status:           value.Status,
		Project:          value.Project,
		Analyzer:         value.Analyzer,
		Scopes:           append([]ScopeSummary(nil), scopes...),
		Modules:          value.Modules,
		References:       value.References,
		SourceReferences: value.SourceReferences,
		Relationships:    value.Relationships,
		Diagnostics:      value.Diagnostics,
		Derived:          value.Derived,
	}
	result.ModelID = aggregateModelID(*result)
	return result
}

func aggregateModelID(value AggregateModel) string {
	value.ModelID = ""
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return "aggregate-model-" + hex.EncodeToString(sum[:12])
}

func namespaceIDs(scope string, values []string) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = namespacedID(scope, value)
	}
	sort.Strings(result)
	return result
}

func addScopeMetadata(values map[string]any, scope string) map[string]any {
	result := make(map[string]any, len(values)+1)
	for key, value := range values {
		result[key] = value
	}
	result["scope_id"] = scope
	return result
}

func repositoryPath(root, local string) string {
	local = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(local)), "./")
	if local == "." || local == "" {
		return normalizeRelativePath(root)
	}
	if root == "." || root == "" {
		return normalizeRelativePath(local)
	}
	return normalizeRelativePath(filepath.ToSlash(filepath.Join(filepath.FromSlash(root), filepath.FromSlash(local))))
}

func diagnosticForJob(job AnalyzerJob, err error) analysis.Diagnostic {
	return analysis.Diagnostic{Code: string(analysis.ErrorCodeOf(err)), Severity: "error", Message: errorMessage(err), Subject: job.LogicalAnalyzerID, Path: job.RelativeProjectRoot, Recoverable: true, Metadata: map[string]any{"scope_id": job.ScopeID, "job_id": job.JobID}}
}

func scopedDiagnostic(scope string, diagnostic analysis.Diagnostic) ScopedDiagnostic {
	metadata := cloneAnyMap(diagnostic.Metadata)
	if scope != "" {
		metadata["scope_id"] = scope
	}
	local := diagnostic.Code + "\x00" + diagnostic.Subject + "\x00" + diagnostic.Path + "\x00" + diagnostic.Message
	idScope := scope
	if idScope == "" {
		idScope = "aggregate"
	}
	return ScopedDiagnostic{ID: namespacedID(idScope, local), ScopeID: scope, Code: diagnostic.Code, Severity: diagnostic.Severity, Message: diagnostic.Message, Subject: diagnostic.Subject, Path: diagnostic.Path, Location: diagnostic.Location, Recoverable: diagnostic.Recoverable, Metadata: metadata}
}

func containsDiagnostic(values []analysis.Diagnostic, target analysis.Diagnostic) bool {
	for _, value := range values {
		left, _ := json.Marshal(value)
		right, _ := json.Marshal(target)
		if string(left) == string(right) {
			return true
		}
	}
	return false
}

func cloneAnyMap(values map[string]any) map[string]any {
	result := make(map[string]any, len(values)+1)
	for key, value := range values {
		result[key] = cloneAnyValue(value)
	}
	return result
}

func cloneAnyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case map[string]string:
		result := make(map[string]string, len(typed))
		for key, item := range typed {
			result[key] = item
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = cloneAnyValue(item)
		}
		return result
	case []string:
		return append([]string(nil), typed...)
	default:
		return value
	}
}

func cloneAnalysisResult(value analysis.AnalysisResult) analysis.AnalysisResult {
	value.Modules = append([]analysis.ModuleObservation(nil), value.Modules...)
	for index := range value.Modules {
		value.Modules[index].Hierarchy = append([]string(nil), value.Modules[index].Hierarchy...)
		value.Modules[index].SourceReferenceIDs = append([]string(nil), value.Modules[index].SourceReferenceIDs...)
		value.Modules[index].Tags = append([]string(nil), value.Modules[index].Tags...)
		value.Modules[index].Metadata = cloneAnyMap(value.Modules[index].Metadata)
	}
	value.References = append([]analysis.Reference(nil), value.References...)
	for index := range value.References {
		value.References[index].Metadata = cloneAnyMap(value.References[index].Metadata)
	}
	value.SourceReferences = append([]analysis.SourceReference(nil), value.SourceReferences...)
	for index := range value.SourceReferences {
		value.SourceReferences[index].Start = clonePosition(value.SourceReferences[index].Start)
		value.SourceReferences[index].End = clonePosition(value.SourceReferences[index].End)
	}
	value.Relationships = append([]analysis.RelationshipObservation(nil), value.Relationships...)
	for index := range value.Relationships {
		value.Relationships[index].SourceReferenceIDs = append([]string(nil), value.Relationships[index].SourceReferenceIDs...)
		value.Relationships[index].Metadata = cloneAnyMap(value.Relationships[index].Metadata)
		if value.Relationships[index].Confidence != nil {
			confidence := *value.Relationships[index].Confidence
			value.Relationships[index].Confidence = &confidence
		}
	}
	value.Diagnostics = append([]analysis.Diagnostic(nil), value.Diagnostics...)
	for index := range value.Diagnostics {
		value.Diagnostics[index].Location = clonePosition(value.Diagnostics[index].Location)
		value.Diagnostics[index].Metadata = cloneAnyMap(value.Diagnostics[index].Metadata)
	}
	return value
}

func clonePosition(value *analysis.Position) *analysis.Position {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func repositoryLabel(root string) string {
	if strings.TrimSpace(root) == "" {
		return "repository"
	}
	return filepath.Base(filepath.Clean(root))
}
