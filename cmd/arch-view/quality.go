package main

import (
	"encoding/json"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/quality"
	"github.com/buffo/arch-view/internal/quality/adapter"
)

type qualityCLIConfig struct {
	Profile  quality.QualityProfile
	Baseline *quality.Baseline
	Policy   quality.ExitPolicy
	Enabled  bool
}

func loadQualityCLIConfig(profilePath, baselinePath, exitOn string, exitStatuses []string) (qualityCLIConfig, error) {
	profilePath = strings.TrimSpace(profilePath)
	baselinePath = strings.TrimSpace(baselinePath)
	if profilePath == "" && baselinePath != "" {
		return qualityCLIConfig{}, analysis.NewHostError(analysis.ErrInvalidRequest, "--quality-baseline requires --quality-profile", nil)
	}
	config := qualityCLIConfig{}
	if profilePath != "" {
		data, err := os.ReadFile(profilePath)
		if err != nil {
			return config, analysis.WrapHostError(analysis.ErrInvalidOptions, "quality profile could not be read", err, map[string]any{"input": profilePath})
		}
		if err := json.Unmarshal(data, &config.Profile); err != nil {
			return config, analysis.WrapHostError(analysis.ErrInvalidOptions, "quality profile JSON is invalid", err, map[string]any{"input": profilePath})
		}
		config.Enabled = true
	}
	if baselinePath != "" {
		data, err := os.ReadFile(baselinePath)
		if err != nil {
			return config, analysis.WrapHostError(analysis.ErrInvalidOptions, "quality baseline could not be read", err, map[string]any{"input": baselinePath})
		}
		var baseline quality.Baseline
		if err := json.Unmarshal(data, &baseline); err != nil {
			return config, analysis.WrapHostError(analysis.ErrInvalidOptions, "quality baseline JSON is invalid", err, map[string]any{"input": baselinePath})
		}
		config.Baseline = &baseline
	}
	policy, err := parseQualityExitPolicy(exitOn, exitStatuses)
	if err != nil {
		return config, err
	}
	config.Policy = policy
	return config, nil
}

func parseQualityExitPolicy(exitOn string, statuses []string) (quality.ExitPolicy, error) {
	exitOn = strings.ToLower(strings.TrimSpace(exitOn))
	if exitOn == "none" {
		exitOn = ""
	}
	if exitOn == "any" {
		exitOn = quality.SeverityInfo
	}
	policy := quality.ExitPolicy{MinimumSeverity: exitOn, Statuses: []string{}}
	for _, value := range statuses {
		for _, status := range strings.Split(value, ",") {
			status = strings.ToLower(strings.TrimSpace(status))
			if status != "" {
				policy.Statuses = append(policy.Statuses, status)
			}
		}
	}
	if exitOn != "" && len(policy.Statuses) == 0 {
		policy.Statuses = []string{quality.StatusActive}
	}
	policy.Statuses = uniqueSortedStrings(policy.Statuses)
	if err := quality.ValidateExitPolicy(policy); err != nil {
		return quality.ExitPolicy{}, analysis.NewHostError(analysis.ErrInvalidOptions, err.Error(), nil)
	}
	return policy, nil
}

func evaluateAnalysisResultQuality(result analysis.AnalysisResult, config qualityCLIConfig) (analysis.AnalysisResult, error) {
	if !config.Enabled {
		return result, nil
	}
	value, err := canonical.Normalize(result)
	if err != nil {
		return result, analysis.WrapHostError(analysis.ErrInvalidModel, "quality evaluation could not normalize the analysis result", err, nil)
	}
	_, report, err := evaluateModelQuality(value, config)
	if err != nil {
		return result, err
	}
	result.QualityReport = report
	return result, nil
}

func evaluateModelQuality(value model.Model, config qualityCLIConfig) (model.Model, *quality.QualityEvaluation, error) {
	if !config.Enabled {
		return value, nil, nil
	}
	input, err := adapter.EvaluationInputFromModel(value)
	if err != nil {
		return value, nil, err
	}
	input.Baseline = config.Baseline
	report, err := quality.EvaluateQualityProfile(config.Profile, input, quality.NewDefaultCatalog())
	if err != nil {
		return value, nil, err
	}
	withReport, err := canonical.WithQualityReport(value, &report)
	if err != nil {
		return value, nil, err
	}
	return withReport, &report, nil
}

func attachQualityToRun(run *orchestration.AnalysisRun, config qualityCLIConfig, scope string) (*quality.QualityEvaluation, error) {
	if !config.Enabled {
		return nil, nil
	}
	if run == nil {
		return nil, analysis.NewHostError(analysis.ErrInvalidModel, "quality evaluation requires an analysis run", nil)
	}
	if strings.TrimSpace(scope) != "" && !strings.EqualFold(strings.TrimSpace(scope), "all") {
		selected, err := run.ScopeResult(scope)
		if err != nil {
			return nil, err
		}
		selected, err = evaluateAnalysisResultQuality(selected, config)
		if err != nil {
			return nil, err
		}
		if selected.QualityReport == nil {
			return nil, analysis.NewHostError(analysis.ErrInvalidModel, "quality evaluation did not produce a report", nil)
		}
		if err := run.AttachScopeQualityReport(scope, *selected.QualityReport); err != nil {
			return nil, err
		}
		return selected.QualityReport, nil
	}
	// Keep concrete reports alongside the combined report so an aggregate
	// viewer can switch scopes without losing the quality projection. Failed
	// scopes have no canonical model and are intentionally skipped.
	for _, summary := range run.Scopes {
		if strings.TrimSpace(summary.ScopeID) == "" {
			continue
		}
		selected, scopeErr := run.ScopeResult(summary.ScopeID)
		if scopeErr != nil {
			continue
		}
		selected, scopeErr = evaluateAnalysisResultQuality(selected, config)
		if scopeErr != nil {
			return nil, scopeErr
		}
		if selected.QualityReport == nil {
			return nil, analysis.NewHostError(analysis.ErrInvalidModel, "quality evaluation did not produce a report", map[string]any{"scope_id": summary.ScopeID})
		}
		if scopeErr := run.AttachScopeQualityReport(summary.ScopeID, *selected.QualityReport); scopeErr != nil {
			return nil, scopeErr
		}
	}
	value, ok := run.CombinedCanonicalModel()
	if !ok {
		return nil, analysis.NewHostError(analysis.ErrInvalidModel, "quality evaluation is unavailable because no usable combined model exists", nil)
	}
	withReport, report, err := evaluateModelQuality(value, config)
	if err != nil {
		return nil, err
	}
	if err := run.AttachQualityReport(*report); err != nil {
		return nil, err
	}
	_ = withReport
	return report, nil
}

func qualityExitCode(report *quality.QualityEvaluation, policy quality.ExitPolicy) (int, error) {
	triggered, err := quality.ExitPolicyTriggered(report, policy)
	if err != nil {
		return analysis.ExitCodeForError(analysis.NewHostError(analysis.ErrInvalidOptions, err.Error(), nil)), err
	}
	if triggered {
		return 1, nil
	}
	return 0, nil
}

func writeQualityJSON(writer io.Writer, value any, report *quality.QualityEvaluation, policy quality.ExitPolicy) int {
	if code := writeJSON(writer, value); code != 0 {
		return code
	}
	code, err := qualityExitCode(report, policy)
	if err != nil {
		return analysis.ExitCodeForError(analysis.NewHostError(analysis.ErrInvalidOptions, err.Error(), nil))
	}
	return code
}

func uniqueSortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	for index := range result {
		result[index] = strings.TrimSpace(result[index])
	}
	sort.Strings(result)
	write := 0
	for _, value := range result {
		if value == "" || write > 0 && result[write-1] == value {
			continue
		}
		result[write] = value
		write++
	}
	return result[:write]
}
