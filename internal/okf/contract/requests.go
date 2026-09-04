package contract

import "github.com/buffo/arch-view/internal/okf/domain"

type SessionBundleRequest struct {
	BundleID string `json:"bundle_id"`
}
type SessionProfileRequest struct {
	ProfileID string `json:"profile_id"`
}
type NavigationDepthRequest struct {
	Depth int  `json:"depth"`
	Full  bool `json:"full"`
}
type NavigationFocusRequest struct {
	ConceptID string `json:"concept_id"`
}
type BindingRequest struct {
	ProfileID        string `json:"profile_id,omitempty"`
	ExpectedRevision string `json:"expected_revision,omitempty"`
	OperationID      string `json:"operation_id,omitempty"`
}
type ProfileSaveRequest struct {
	Profile          domain.Profile `json:"profile"`
	ExpectedRevision string         `json:"expected_revision,omitempty"`
	OperationID      string         `json:"operation_id,omitempty"`
}
type ProfileSaveAsRequest struct {
	Profile          domain.Profile `json:"profile"`
	SourceProfileID  string         `json:"source_profile_id,omitempty"`
	NewProfileID     string         `json:"new_profile_id"`
	ExpectedRevision string         `json:"expected_revision,omitempty"`
	OperationID      string         `json:"operation_id,omitempty"`
}
type ProfileRenameRequest struct {
	NewProfileID     string `json:"new_profile_id"`
	NewName          string `json:"new_name,omitempty"`
	ExpectedRevision string `json:"expected_revision,omitempty"`
	OperationID      string `json:"operation_id,omitempty"`
}
type ProfileDeleteRequest struct {
	ReplacementProfileID string `json:"replacement_profile_id,omitempty"`
	NeutralFallback      bool   `json:"neutral_fallback,omitempty"`
	ExpectedRevision     string `json:"expected_revision,omitempty"`
	OperationID          string `json:"operation_id,omitempty"`
}
