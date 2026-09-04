package application

import "github.com/buffo/arch-view/internal/okf/domain"

type SessionView struct {
	SessionID   string                     `json:"session_id"`
	BundleID    string                     `json:"bundle_id,omitempty"`
	ProfileID   string                     `json:"profile_id,omitempty"`
	Navigation  domain.NavigationState     `json:"navigation"`
	Projection  *domain.ProjectionSnapshot `json:"projection,omitempty"`
	Diagnostics []domain.Diagnostic        `json:"diagnostics"`
}

type ProfileInput struct {
	Profile          domain.Profile `json:"profile"`
	SourceProfileID  string         `json:"source_profile_id,omitempty"`
	NewProfileID     string         `json:"new_profile_id,omitempty"`
	NewName          string         `json:"new_name,omitempty"`
	ReplacementID    string         `json:"replacement_profile_id,omitempty"`
	NeutralFallback  bool           `json:"neutral_fallback,omitempty"`
	ExpectedRevision string         `json:"expected_revision,omitempty"`
	OperationID      string         `json:"operation_id,omitempty"`
}

type ProfilePreview struct {
	Profile          domain.Profile      `json:"profile"`
	EffectiveProfile domain.Profile      `json:"effective_profile"`
	Valid            bool                `json:"valid"`
	Diagnostics      []domain.Diagnostic `json:"diagnostics"`
}
