package domain

import (
	"bytes"
	"encoding/json"
)

// UnmarshalJSON records whether a profile supplied hierarchy settings so an
// explicit false value can override an inherited true value.
func (settings *HierarchySettings) UnmarshalJSON(data []byte) error {
	var value struct {
		UseExplicit   *bool `json:"use_explicit"`
		UseFilesystem *bool `json:"use_filesystem_fallback"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*settings = HierarchySettings{ExplicitConfigured: value.UseExplicit != nil, FilesystemConfigured: value.UseFilesystem != nil}
	if value.UseExplicit != nil {
		settings.UseExplicit = *value.UseExplicit
	}
	if value.UseFilesystem != nil {
		settings.UseFilesystem = *value.UseFilesystem
	}
	settings.Configured = value.UseExplicit != nil || value.UseFilesystem != nil
	return nil
}

// UnmarshalJSON records whether a profile supplied relationship settings so
// each relationship layer remains independently configurable.
func (settings *RelationshipSettings) UnmarshalJSON(data []byte) error {
	var value struct {
		ShowContainment *bool `json:"show_containment"`
		ShowSemantic    *bool `json:"show_semantic_links"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*settings = RelationshipSettings{ContainmentConfigured: value.ShowContainment != nil, SemanticConfigured: value.ShowSemantic != nil}
	if value.ShowContainment != nil {
		settings.ShowContainment = *value.ShowContainment
	}
	if value.ShowSemantic != nil {
		settings.ShowSemantic = *value.ShowSemantic
	}
	settings.Configured = value.ShowContainment != nil || value.ShowSemantic != nil
	return nil
}

// UnmarshalJSON records field presence so a project profile can explicitly
// disable a state behavior inherited from a base profile.
func (settings *StateSettings) UnmarshalJSON(data []byte) error {
	var value struct {
		Field        *string           `json:"field"`
		Mapping      map[string]string `json:"mapping"`
		RollUp       *bool             `json:"roll_up"`
		ShowDeclared *bool             `json:"show_declared"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*settings = StateSettings{}
	if value.Field != nil {
		settings.Field, settings.FieldConfigured = *value.Field, true
	}
	if value.Mapping != nil {
		settings.Mapping, settings.MappingConfigured = value.Mapping, true
	}
	if value.RollUp != nil {
		settings.RollUp, settings.RollUpConfigured = *value.RollUp, true
	}
	if value.ShowDeclared != nil {
		settings.ShowDeclared, settings.ShowDeclaredConfigured = *value.ShowDeclared, true
	}
	settings.Configured = settings.FieldConfigured || settings.MappingConfigured || settings.RollUpConfigured || settings.ShowDeclaredConfigured
	return nil
}
