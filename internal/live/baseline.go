package live

import (
	"context"
	"fmt"
	"strings"

	"github.com/buffo/arch-view/internal/quality"
	qualitypolicy "github.com/buffo/arch-view/internal/quality/policy"
)

type qualityBaselineResolution struct {
	Profile  quality.QualityProfile
	Baseline *quality.Baseline
	Warning  string
}

func resolveQualityBaseline(ctx context.Context, service QualityPolicyService, profile quality.QualityProfile, mode BaselineMode, files []string) (qualityBaselineResolution, error) {
	if mode == "" {
		mode = BaselineModeProfile
	}
	resolution := qualityBaselineResolution{Profile: cloneQualityProfile(profile)}
	switch mode {
	case BaselineModeNone:
		resolution.Profile.Baseline = nil
		return resolution, nil
	case BaselineModeProfile:
		if profile.Baseline == nil {
			return resolution, nil
		}
		if service == nil {
			resolution.Warning = "the quality profile references a baseline, but the baseline reader is unavailable; findings remain unsuppressed"
			return resolution, nil
		}
		baseline, _, err := service.ResolveBaseline(ctx, profile.Baseline.BaselineID, profile.Baseline.Revision)
		if err != nil {
			if liveErrorCode(err) == ErrorBaselineNotFound {
				resolution.Warning = fmt.Sprintf("the quality profile references baseline %s@%s, but no matching baseline was found; findings remain unsuppressed", profile.Baseline.BaselineID, profile.Baseline.Revision)
				return resolution, nil
			}
			return qualityBaselineResolution{}, err
		}
		resolution.Baseline = &baseline
		return resolution, nil
	case BaselineModeSelected:
		if len(files) == 0 {
			return qualityBaselineResolution{}, newLiveError(ErrorQualityEvaluation, "selected baseline mode requires one or more baseline files", nil)
		}
		if service == nil {
			return qualityBaselineResolution{}, newLiveError(ErrorQualityEvaluation, "selected baseline mode requires a baseline reader", nil)
		}
		var merged quality.Baseline
		for index, fileName := range files {
			fileName = strings.TrimSpace(fileName)
			if fileName == "" {
				return qualityBaselineResolution{}, newLiveError(ErrorQualityEvaluation, "selected baseline files may not be empty", nil)
			}
			if err := qualitypolicy.ValidateManagedBaselineFileName(fileName); err != nil {
				return qualityBaselineResolution{}, newLiveError("QueryInvalid", "selected baseline files must be direct project-local JSON files", map[string]any{"file_name": fileName})
			}
			baseline, _, err := service.ReadBaseline(ctx, fileName)
			if err != nil {
				return qualityBaselineResolution{}, newLiveError(ErrorQualityEvaluation, "selected quality baseline could not be read", map[string]any{"file_name": fileName, "error": err.Error()})
			}
			if index == 0 {
				merged = baseline
				continue
			}
			if baseline.BaselineID != merged.BaselineID || baseline.Revision != merged.Revision {
				return qualityBaselineResolution{}, newLiveError(ErrorQualityEvaluation, "selected baselines must share one baseline ID and revision", map[string]any{"file_name": fileName, "baseline_id": baseline.BaselineID, "revision": baseline.Revision})
			}
			merge, mergeErr := quality.MergeBaselineEntries(merged, baseline.Entries)
			if mergeErr != nil {
				return qualityBaselineResolution{}, newLiveError(ErrorQualityEvaluation, "selected quality baselines could not be merged", map[string]any{"file_name": fileName, "error": mergeErr.Error()})
			}
			merged = merge.Baseline
		}
		// Selected baselines are an explicit, per-evaluation override. The
		// temporary profile copy below makes the evaluator identity coherent;
		// the saved profile and its canonical reference remain untouched.
		resolution.Profile.Baseline = &quality.BaselineRef{BaselineID: merged.BaselineID, Revision: merged.Revision}
		resolution.Baseline = &merged
		return resolution, nil
	default:
		return qualityBaselineResolution{}, newLiveError(ErrorQualityEvaluation, "baseline mode is unsupported", map[string]any{"baseline_mode": mode})
	}
}

func liveErrorCode(err error) string {
	if value, ok := err.(*QueryError); ok && value != nil {
		return value.Code
	}
	return ""
}
