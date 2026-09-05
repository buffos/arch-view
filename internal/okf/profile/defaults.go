package profile

import "github.com/buffo/arch-view/internal/okf/domain"

const defaultOKFLayoutAlgorithm = "layered"

func defaultOKFLayoutFeatures() []string {
	return []string{"junctions", "ports"}
}

func defaultOKFLayout() domain.LayoutSettings {
	return domain.LayoutSettings{Algorithm: defaultOKFLayoutAlgorithm, Features: defaultOKFLayoutFeatures()}
}

func normalize(value domain.Profile) domain.Profile {
	value = domain.CloneProfile(value)
	if value.Name == "" {
		value.Name = value.ProfileID
	}
	if value.Origin == "" {
		value.Origin = "project_local"
	}
	if value.Hierarchy.ExplicitConfigured || value.Hierarchy.FilesystemConfigured {
		if !value.Hierarchy.ExplicitConfigured {
			value.Hierarchy.UseExplicit = true
		}
		if !value.Hierarchy.FilesystemConfigured {
			value.Hierarchy.UseFilesystem = true
		}
	}
	if value.Relationships.ContainmentConfigured || value.Relationships.SemanticConfigured {
		if !value.Relationships.ContainmentConfigured {
			value.Relationships.ShowContainment = true
		}
		if !value.Relationships.SemanticConfigured {
			value.Relationships.ShowSemantic = true
		}
	}
	if value.Hierarchy == (domain.HierarchySettings{}) || (!value.Hierarchy.Configured && value.Hierarchy.UseExplicit == false && value.Hierarchy.UseFilesystem == false) {
		value.Hierarchy = domain.HierarchySettings{UseExplicit: true, UseFilesystem: true}
		value.Hierarchy.Configured = true
	} else if value.Hierarchy.UseExplicit || value.Hierarchy.UseFilesystem {
		value.Hierarchy.Configured = true
	}
	if value.Relationships == (domain.RelationshipSettings{}) || (!value.Relationships.Configured && !value.Relationships.ShowContainment && !value.Relationships.ShowSemantic) {
		value.Relationships = domain.RelationshipSettings{ShowContainment: true, ShowSemantic: true}
		value.Relationships.Configured = true
	} else if value.Relationships.ShowContainment || value.Relationships.ShowSemantic {
		value.Relationships.Configured = true
	}
	if value.Navigation.DefaultDepth == 0 && !value.Navigation.DepthConfigured {
		value.Navigation.DefaultDepth = 2
	}
	if value.Navigation.MaxNodes == 0 && !value.Navigation.NodesConfigured {
		value.Navigation.MaxNodes = 1000
	}
	if value.Navigation.MaxRelationships == 0 && !value.Navigation.RelationshipsConfigured {
		value.Navigation.MaxRelationships = 10000
	}
	if value.Style.DefaultToken == "" {
		value.Style.DefaultToken = "state.unknown"
	}
	if value.Style.Tokens == nil {
		value.Style.Tokens = defaultTokens()
	}
	if value.Layout.Algorithm == "" {
		value.Layout.Algorithm = defaultOKFLayoutAlgorithm
	}
	if value.Layout.Options == nil {
		value.Layout.Options = map[string]any{}
	}
	if value.Bases == nil {
		value.Bases = []string{}
	}
	if value.Rules == nil {
		value.Rules = []domain.RuleInvocation{}
	}
	if !value.State.ShowDeclaredConfigured {
		value.State.ShowDeclared = true
	}
	if !value.Details.RawConfigured {
		value.Details.ShowRawMarkdown = true
	}
	if !value.Details.UnknownConfigured {
		value.Details.ShowUnknown = true
	}
	return value
}
