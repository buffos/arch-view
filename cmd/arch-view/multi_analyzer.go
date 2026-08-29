package main

import (
	"context"
	"io"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/export"
	"github.com/buffo/arch-view/internal/viewer/layout"
)

func runCombinedAnalysis(ctx context.Context, host *analysis.Host, project string, cliOptions map[string]any) (orchestration.AnalysisRun, error) {
	return runCombinedAnalysisWithPolicy(ctx, host, project, cliOptions, orchestration.SourceScopePolicy{})
}

func runCombinedAnalysisWithPolicy(ctx context.Context, host *analysis.Host, project string, cliOptions map[string]any, sourcePolicy orchestration.SourceScopePolicy) (orchestration.AnalysisRun, error) {
	if host == nil {
		return orchestration.AnalysisRun{}, analysis.NewHostError(analysis.ErrHostFailure, "analysis host is not initialized", nil)
	}
	registry := host.RegistrySnapshot()
	planner := orchestration.NewAnalyzerJobPlanner(registry)
	policy := sourcePolicy
	if values, ok := cliOptions["exclude"].([]string); ok {
		policy.Exclude = append(append([]string(nil), policy.Exclude...), values...)
	}
	plan, err := planner.PlanAnalyzerJobs(ctx, orchestration.PlanRequest{
		RepositoryRoot:    project,
		SourceScopePolicy: policy,
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
	snapshot := scheduler.ExecuteAnalyzerPlan(ctx, plan)
	return orchestration.AggregateScopeResults(snapshot)
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

func writeCombinedAnalysisOutput(run orchestration.AnalysisRun, format, output, project, scope, referenceVisibility string, viewPath, referenceScopes []string, overwrite, embedSource bool, ctx context.Context, stdout, stderr io.Writer) int {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "analysis-json" {
		var value any = run
		if scope != "" && !strings.EqualFold(scope, "all") {
			selected, err := run.ScopeResult(scope)
			if err != nil {
				writeError(stderr, err)
				return analysis.ExitCodeForError(err)
			}
			value = selected
		}
		if output == "-" {
			if code := writeJSON(stdout, value); code != 0 {
				return code
			}
		} else if err := writeFileJSON(output, value); err != nil {
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
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
	return writeExport(selected.Model, export.Request{Format: format, OutputPath: output, ViewPath: viewPath, ReferenceVisibility: referenceVisibility, ReferenceScopes: referenceScopes, LayoutProfile: layoutProfile, Overwrite: overwrite, EmbedSource: embedSource, Context: ctx}, stdout, stderr)
}
