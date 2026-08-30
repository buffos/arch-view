package main

import (
	"context"
	"io"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	analysisconfig "github.com/buffo/arch-view/internal/analysis/config"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/export"
	"github.com/buffo/arch-view/internal/quality"
	"github.com/buffo/arch-view/internal/viewer/layout"
)

func runCombinedAnalysis(ctx context.Context, host *analysis.Host, project string, cliOptions map[string]any) (orchestration.AnalysisRun, error) {
	return runCombinedAnalysisWithPolicy(ctx, host, project, cliOptions, orchestration.SourceScopePolicy{})
}

func runCombinedAnalysisWithPolicy(ctx context.Context, host *analysis.Host, project string, cliOptions map[string]any, sourcePolicy orchestration.SourceScopePolicy) (orchestration.AnalysisRun, error) {
	return runCombinedAnalysisWithPolicyAndCache(ctx, host, project, cliOptions, sourcePolicy, nil)
}

func runCombinedAnalysisWithPolicyAndCache(ctx context.Context, host *analysis.Host, project string, cliOptions map[string]any, sourcePolicy orchestration.SourceScopePolicy, cache *orchestration.SessionCache) (orchestration.AnalysisRun, error) {
	if host == nil {
		return orchestration.AnalysisRun{}, analysis.NewHostError(analysis.ErrHostFailure, "analysis host is not initialized", nil)
	}
	registry := host.RegistrySnapshot()
	planner := orchestration.NewAnalyzerJobPlanner(registry)
	invocationRoot := sourcePolicy.InvocationRoot
	if strings.TrimSpace(invocationRoot) == "" {
		invocationRoot = "."
	}
	configuration, configurationErr := analysisconfig.LoadNearestAt(project, invocationRoot, registry)
	if configurationErr != nil {
		return orchestration.AnalysisRun{}, configurationErr
	}
	configuredPolicy, policyErr := configuration.SourceScopePolicy(invocationRoot)
	if policyErr != nil {
		return orchestration.AnalysisRun{}, policyErr
	}
	policy, policyErr := mergeCombinedSourcePolicies(configuredPolicy, sourcePolicy, invocationRoot)
	if policyErr != nil {
		return orchestration.AnalysisRun{}, policyErr
	}
	if values, ok := cliOptions["exclude"].([]string); ok {
		policy.Exclude = append(append([]string(nil), policy.Exclude...), values...)
		policy, policyErr = orchestration.NormalizeSourceScopePolicy(policy, invocationRoot)
		if policyErr != nil {
			return orchestration.AnalysisRun{}, policyErr
		}
	}
	assignments := []orchestration.AnalyzerAssignment{}
	if configuration.Analysis != nil {
		assignments = append(assignments, configuration.Analysis.Assignments...)
	}
	plan, err := planner.PlanAnalyzerJobs(ctx, orchestration.PlanRequest{
		RepositoryRoot:    project,
		InvocationRoot:    invocationRoot,
		SourceScopePolicy: policy,
		Assignments:       assignments,
		CLIOptionsByID:    splitOptionsByAnalyzer(registry, cliOptions),
		Runtime:           host.Runtime(),
	})
	if err != nil {
		return orchestration.AnalysisRun{}, err
	}
	scheduler := orchestration.NewAnalyzerJobScheduler(func(runContext context.Context, job orchestration.AnalyzerJob) (analysis.AnalysisResult, error) {
		return host.RunPlanned(runContext, analysis.PlannedRunRequest{
			ProjectRoot: job.ProjectRoot,
			AnalyzerID:  job.LogicalAnalyzerID,
			Selection:   job.Selection,
			Options:     job.Options,
			SourceScope: &job.EffectiveSourceScope,
		})
	}, orchestration.SchedulerOptions{})
	snapshot := orchestration.ExecuteAnalyzerPlanWithCache(ctx, scheduler, plan, cache)
	return orchestration.AggregateScopeResults(snapshot)
}

// runConfiguredSingleAnalysis keeps explicit CLI selection on the same
// configuration, source-scope, and planning boundary as combined analysis.
// The CLI selection has precedence over persisted assignments, while the
// persisted filters still constrain the effective source scope.
func runConfiguredSingleAnalysis(ctx context.Context, host *analysis.Host, project, analyzerID, language string, cliOptions map[string]any) (analysis.AnalysisResult, error) {
	if host == nil {
		return analysis.AnalysisResult{}, analysis.NewHostError(analysis.ErrHostFailure, "analysis host is not initialized", nil)
	}
	registry := host.RegistrySnapshot()
	configuration, err := analysisconfig.LoadNearest(project, registry)
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	// A missing analysis document, and a v1 layout-only document, must retain
	// the established explicit CLI result shape. There is no configured scope
	// to resolve in either case, so the host path is the compatibility boundary
	// while v2 configuration continues through the shared planner below.
	if configuration.Analysis == nil {
		return host.Run(ctx, analysis.RunRequest{
			ProjectRoot:    project,
			AnalyzerID:     analyzerID,
			Language:       language,
			CLIOptions:     cliOptions,
			ProjectOptions: map[string]any{},
		})
	}
	configuredPolicy, err := configuration.SourceScopePolicy(".")
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	policy, err := mergeCombinedSourcePolicies(configuredPolicy, orchestration.SourceScopePolicy{}, ".")
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	if values, ok := cliOptions["exclude"].([]string); ok {
		policy.Exclude = append(append([]string(nil), policy.Exclude...), values...)
		policy, err = orchestration.NormalizeSourceScopePolicy(policy, ".")
		if err != nil {
			return analysis.AnalysisResult{}, err
		}
	}
	assignments := []orchestration.AnalyzerAssignment{}
	if configuration.Analysis != nil {
		assignments = append(assignments, configuration.Analysis.Assignments...)
	}
	plan, err := orchestration.PlanAnalyzerJobs(ctx, registry, orchestration.PlanRequest{
		RepositoryRoot:    project,
		InvocationRoot:    ".",
		SourceScopePolicy: policy,
		Assignments:       assignments,
		CLISelection: &orchestration.ExplicitSelection{
			ProjectRoot: ".",
			AnalyzerID:  analyzerID,
			Language:    language,
		},
		CLISelectionOnly: true,
		CLIOptionsByID:   splitOptionsByAnalyzer(registry, cliOptions),
		Runtime:          host.Runtime(),
	})
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	scheduler := orchestration.NewAnalyzerJobScheduler(func(runContext context.Context, job orchestration.AnalyzerJob) (analysis.AnalysisResult, error) {
		return host.RunPlanned(runContext, analysis.PlannedRunRequest{
			ProjectRoot: job.ProjectRoot,
			AnalyzerID:  job.LogicalAnalyzerID,
			Selection:   job.Selection,
			Options:     job.Options,
			SourceScope: &job.EffectiveSourceScope,
		})
	}, orchestration.SchedulerOptions{})
	run, err := orchestration.AggregateScopeResults(scheduler.ExecuteAnalyzerPlan(ctx, plan))
	if err != nil {
		return analysis.AnalysisResult{}, err
	}
	for _, scope := range run.Scopes {
		if scope.ProjectRoot == "." || strings.EqualFold(scope.ProjectRoot, "") {
			result, resultErr := run.ScopeResult(scope.ScopeID)
			if resultErr == nil {
				if result.Modules == nil {
					result.Modules = []analysis.ModuleObservation{}
				}
				if result.Relationships == nil {
					result.Relationships = []analysis.RelationshipObservation{}
				}
				if result.References == nil {
					result.References = []analysis.Reference{}
				}
				if result.SourceReferences == nil {
					result.SourceReferences = []analysis.SourceReference{}
				}
				if result.Diagnostics == nil {
					result.Diagnostics = []analysis.Diagnostic{}
				}
				return result, nil
			}
			for _, diagnostic := range scope.Diagnostics {
				if diagnostic.Code == string(analysis.ErrAssignmentAnalyzerUnavailable) {
					return analysis.AnalysisResult{}, analysis.NewHostError(analysis.ErrNoAnalyzer, "the explicitly selected analyzer is unavailable", map[string]any{"analyzer_id": analyzerID})
				}
			}
			return analysis.AnalysisResult{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "the explicitly selected analyzer did not detect the project", map[string]any{
				"analyzer_id": analyzerID,
				"language":    language,
			})
		}
	}
	return analysis.AnalysisResult{}, analysis.NewHostError(analysis.ErrNoAnalyzer, "explicit analyzer selection produced no usable scope", map[string]any{
		"analyzer_id": analyzerID,
		"language":    language,
	})
}

func mergeCombinedSourcePolicies(configured, requested orchestration.SourceScopePolicy, invocationRoot string) (orchestration.SourceScopePolicy, error) {
	policy := configured
	policy.InvocationRoot = invocationRoot
	policy.Exclude = append(append([]string(nil), configured.Exclude...), requested.Exclude...)
	policy.Include = make([]orchestration.AnalyzerIncludeRule, len(configured.Include))
	for index, rule := range configured.Include {
		policy.Include[index] = orchestration.AnalyzerIncludeRule{AnalyzerID: rule.AnalyzerID, Globs: append([]string(nil), rule.Globs...)}
	}
	for _, requestedRule := range requested.Include {
		configuredAnalyzer := false
		for _, configuredRule := range configured.Include {
			if configuredRule.AnalyzerID == requestedRule.AnalyzerID {
				configuredAnalyzer = true
				break
			}
		}
		if !configuredAnalyzer {
			policy.Include = append(policy.Include, orchestration.AnalyzerIncludeRule{AnalyzerID: requestedRule.AnalyzerID, Globs: append([]string(nil), requestedRule.Globs...)})
		}
	}
	return orchestration.NormalizeSourceScopePolicy(policy, invocationRoot)
}

func splitOptionsByAnalyzer(registry *analysis.Registry, values map[string]any) map[string]map[string]any {
	result := make(map[string]map[string]any)
	if registry == nil {
		return result
	}
	for _, manifest := range registry.ListManifests() {
		allowed := make(map[string]struct{}, len(manifest.Options))
		for _, option := range manifest.Options {
			allowed[option.Name] = struct{}{}
		}
		selected := make(map[string]any)
		for name, value := range values {
			if _, ok := allowed[name]; ok {
				selected[name] = value
			}
		}
		result[manifest.ID] = selected
	}
	return result
}

func writeCombinedAnalysisOutput(run orchestration.AnalysisRun, format, output, project, scope, referenceVisibility string, viewPath, referenceScopes []string, overwrite, embedSource bool, policy quality.ExitPolicy, ctx context.Context, stdout, stderr io.Writer) int {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "analysis-json" {
		var value any = run
		var report *quality.QualityEvaluation
		if scope != "" && !strings.EqualFold(scope, "all") {
			selected, err := run.ScopeResult(scope)
			if err != nil {
				writeError(stderr, err)
				return analysis.ExitCodeForError(err)
			}
			value = selected
			report = selected.QualityReport
		} else if run.Model != nil {
			report = run.Model.QualityReport
		}
		if output == "-" {
			return writeQualityJSON(stdout, value, report, policy)
		} else if err := writeFileJSON(output, value); err != nil {
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
		if code, err := qualityExitCode(report, policy); err != nil {
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		} else if code != 0 {
			return code
		}
		return analysis.ExitCodeForStatus(run.Status)
	}
	var selected orchestration.SelectedScope
	var err error
	if scope == "" {
		selectedModel, ok := run.CombinedCanonicalModel()
		if !ok {
			err = analysis.NewHostError(analysis.ErrInvalidModel, "combined export is unavailable because no usable scope exists", nil)
		} else {
			selected = orchestration.SelectedScope{RunID: run.RunID, Scope: "all", Model: selectedModel}
		}
	} else {
		selected, err = run.SelectAnalysisScope(scope)
	}
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	var layoutProfile *layout.LayoutProfile
	if format == export.FormatHTML {
		profile := layout.NewSession(project).Response().Layout
		layoutProfile = &profile
	}
	return writeExportWithQualityPolicy(selected.Model, export.Request{Format: format, OutputPath: output, ViewPath: viewPath, ReferenceVisibility: referenceVisibility, ReferenceScopes: referenceScopes, LayoutProfile: layoutProfile, Overwrite: overwrite, EmbedSource: embedSource, Context: ctx}, policy, stdout, stderr)
}
