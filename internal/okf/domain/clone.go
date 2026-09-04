package domain

import "slices"

func CloneMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = cloneValue(item)
	}
	return result
}

func cloneValue(value any) any {
	switch item := value.(type) {
	case map[string]any:
		return CloneMap(item)
	case []any:
		result := make([]any, len(item))
		for index, child := range item {
			result[index] = cloneValue(child)
		}
		return result
	case []string:
		return slices.Clone(item)
	default:
		return value
	}
}

func CloneDocument(value ConceptDocument) ConceptDocument {
	value.Tags = append([]string(nil), value.Tags...)
	value.Frontmatter = CloneMap(value.Frontmatter)
	value.UnknownFrontmatter = CloneMap(value.UnknownFrontmatter)
	value.Links = append([]Link(nil), value.Links...)
	value.ExplicitParents = append([]string(nil), value.ExplicitParents...)
	value.ExplicitChildren = append([]string(nil), value.ExplicitChildren...)
	value.Provenance = append([]Provenance(nil), value.Provenance...)
	return value
}

func CloneIndex(value BundleIndex) BundleIndex {
	result := value
	result.Documents = make(map[string]ConceptDocument, len(value.Documents))
	for key, document := range value.Documents {
		result.Documents[key] = CloneDocument(document)
	}
	result.ConceptOrder = append([]string(nil), value.ConceptOrder...)
	result.Relationships = make([]Relationship, len(value.Relationships))
	for index, relationship := range value.Relationships {
		result.Relationships[index] = relationship
		result.Relationships[index].Provenance = append([]Provenance(nil), relationship.Provenance...)
	}
	result.Diagnostics = CloneDiagnostics(value.Diagnostics)
	return result
}

func CloneConfiguration(value ProjectConfiguration) ProjectConfiguration {
	result := value
	result.Bindings = append([]ProfileBinding(nil), value.Bindings...)
	result.Profiles = make([]Profile, len(value.Profiles))
	for index, profile := range value.Profiles {
		result.Profiles[index] = CloneProfile(profile)
	}
	return result
}

func CloneProfile(value Profile) Profile {
	result := value
	if value.Details.Renderer != nil {
		selection := *value.Details.Renderer
		selection.Parameters = CloneMap(selection.Parameters)
		result.Details.Renderer = &selection
	}
	if value.Bases != nil {
		result.Bases = append([]string{}, value.Bases...)
	}
	if value.Rules != nil {
		result.Rules = make([]RuleInvocation, len(value.Rules))
	}
	for index, rule := range value.Rules {
		result.Rules[index] = rule
		result.Rules[index].Parameters = CloneMap(rule.Parameters)
	}
	result.State.Mapping = cloneStringMap(value.State.Mapping)
	if value.NodeFields != nil {
		result.NodeFields = append(make([]NodeField, 0, len(value.NodeFields)), value.NodeFields...)
	}
	if value.Style.Tokens != nil {
		result.Style.Tokens = make(map[string]StyleToken, len(value.Style.Tokens))
		for key, token := range value.Style.Tokens {
			result.Style.Tokens[key] = token
		}
	}
	result.Style.StateTokens = cloneStringMap(value.Style.StateTokens)
	if value.Style.Decorations != nil {
		result.Style.Decorations = make(map[string]StyleDecoration, len(value.Style.Decorations))
		for key, decoration := range value.Style.Decorations {
			result.Style.Decorations[key] = decoration
		}
	}
	result.Layout.Options = CloneMap(value.Layout.Options)
	result.Layout.Features = cloneStrings(value.Layout.Features)
	result.extensions = cloneProfileExtensions(value.extensions)
	return result
}

func cloneStringMap(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func CloneDiagnostics(value []Diagnostic) []Diagnostic {
	result := make([]Diagnostic, len(value))
	for index, diagnostic := range value {
		result[index] = diagnostic
		result[index].Details = CloneMap(diagnostic.Details)
	}
	return result
}

func CloneSnapshot(value ProjectionSnapshot) ProjectionSnapshot {
	result := value
	result.Profile.Layout.Options = CloneMap(value.Profile.Layout.Options)
	result.Profile.Layout.Features = cloneStrings(value.Profile.Layout.Features)
	result.Navigation.Breadcrumbs = append([]string(nil), value.Navigation.Breadcrumbs...)
	result.Nodes = make([]SceneNode, len(value.Nodes))
	for index, node := range value.Nodes {
		result.Nodes[index] = node
		nodeCopy := &result.Nodes[index]
		if node.ShapeDefinition != nil {
			definition := *node.ShapeDefinition
			definition.Points = append([]ShapePoint(nil), definition.Points...)
			nodeCopy.ShapeDefinition = &definition
		}
		nodeCopy.Annotations = CloneMap(node.Annotations)
		if node.PresentationFields != nil {
			nodeCopy.PresentationFields = append(make([]NodeFieldValue, 0, len(node.PresentationFields)), node.PresentationFields...)
		}
		nodeCopy.HierarchyPath = append([]string(nil), node.HierarchyPath...)
	}
	result.Relationships = make([]SceneRelationship, len(value.Relationships))
	for index, relationship := range value.Relationships {
		result.Relationships[index] = relationship
		result.Relationships[index].Provenance = append([]Provenance(nil), relationship.Provenance...)
	}
	result.Legend = append([]LegendEntry(nil), value.Legend...)
	result.Diagnostics = CloneDiagnostics(value.Diagnostics)
	return result
}
