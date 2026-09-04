package domain

import "encoding/json"

func optionalBool(value, configured bool) *bool {
	if value || configured {
		return &value
	}
	return nil
}

func (value HierarchySettings) MarshalJSON() ([]byte, error) {
	full := !value.ExplicitConfigured && !value.FilesystemConfigured && (value.Configured || value.UseExplicit || value.UseFilesystem)
	return json.Marshal(struct {
		Explicit   *bool `json:"use_explicit,omitempty"`
		Filesystem *bool `json:"use_filesystem_fallback,omitempty"`
	}{optionalBool(value.UseExplicit, value.ExplicitConfigured || full), optionalBool(value.UseFilesystem, value.FilesystemConfigured || full)})
}

func (value RelationshipSettings) MarshalJSON() ([]byte, error) {
	full := !value.ContainmentConfigured && !value.SemanticConfigured && (value.Configured || value.ShowContainment || value.ShowSemantic)
	return json.Marshal(struct {
		Containment *bool `json:"show_containment,omitempty"`
		Semantic    *bool `json:"show_semantic_links,omitempty"`
	}{optionalBool(value.ShowContainment, value.ContainmentConfigured || full), optionalBool(value.ShowSemantic, value.SemanticConfigured || full)})
}

func (value StateSettings) MarshalJSON() ([]byte, error) {
	var field *string
	if value.FieldConfigured || value.Field != "" {
		field = &value.Field
	}
	var mapping *map[string]string
	if value.MappingConfigured || value.Mapping != nil {
		mapping = &value.Mapping
	}
	return json.Marshal(struct {
		Field        *string            `json:"field,omitempty"`
		Mapping      *map[string]string `json:"mapping,omitempty"`
		RollUp       *bool              `json:"roll_up,omitempty"`
		ShowDeclared *bool              `json:"show_declared,omitempty"`
	}{field, mapping, optionalBool(value.RollUp, value.RollUpConfigured), optionalBool(value.ShowDeclared, value.ShowDeclaredConfigured)})
}

func (value DetailSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Renderer *DetailRendererSelection `json:"renderer,omitempty"`
		Raw      *bool                    `json:"show_raw_markdown,omitempty"`
		Unknown  *bool                    `json:"show_unknown_frontmatter,omitempty"`
	}{value.Renderer, optionalBool(value.ShowRawMarkdown, value.RawConfigured), optionalBool(value.ShowUnknown, value.UnknownConfigured)})
}

func (value *DetailSettings) UnmarshalJSON(data []byte) error {
	var raw struct {
		Renderer *DetailRendererSelection `json:"renderer"`
		Raw      *bool                    `json:"show_raw_markdown"`
		Unknown  *bool                    `json:"show_unknown_frontmatter"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*value = DetailSettings{Renderer: raw.Renderer, RawConfigured: raw.Raw != nil, UnknownConfigured: raw.Unknown != nil}
	if raw.Raw != nil {
		value.ShowRawMarkdown = *raw.Raw
	}
	if raw.Unknown != nil {
		value.ShowUnknown = *raw.Unknown
	}
	return nil
}
