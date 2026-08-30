package canonical

import (
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/quality"
)

// WithQualityReport attaches an already evaluated quality report to a
// canonical model and recomputes the model identity. Quality is an optional
// sibling of the architecture model, so attaching it must not alter any
// architecture, source, or derived semantics.
func WithQualityReport(value model.Model, report *quality.QualityEvaluation) (model.Model, error) {
	if report != nil {
		if err := quality.ValidateQualityEvaluation(*report); err != nil {
			return model.Model{}, err
		}
		copyReport := *report
		value.QualityReport = &copyReport
	} else {
		value.QualityReport = nil
	}
	value.ModelID = modelID(value)
	if err := Validate(value); err != nil {
		return model.Model{}, err
	}
	return value, nil
}
