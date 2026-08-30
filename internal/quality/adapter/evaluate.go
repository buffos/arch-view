package adapter

import (
	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/quality"
)

// EvaluateSourceIndex adapts an authoritative source-index attachment and
// evaluates it without changing the analyzer result or source facts.
func EvaluateSourceIndex(profile quality.QualityProfile, index analysis.SourceIndex, catalog *quality.Catalog) (quality.QualityEvaluation, error) {
	input, err := EvaluationInputFromSourceIndex(index)
	if err != nil {
		return quality.QualityEvaluation{}, err
	}
	return quality.EvaluateQualityProfile(profile, input, catalog)
}

// EvaluateModel adapts canonical model and source-index facts into the
// language-neutral quality boundary.
func EvaluateModel(profile quality.QualityProfile, value model.Model, catalog *quality.Catalog) (quality.QualityEvaluation, error) {
	input, err := EvaluationInputFromModel(value)
	if err != nil {
		return quality.QualityEvaluation{}, err
	}
	return quality.EvaluateQualityProfile(profile, input, catalog)
}
