package model

import (
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

func BuildHierarchyProjection(model Model, selectedPath []string) (HierarchyProjection, error) {
	if err := Validate(model); err != nil {
		return HierarchyProjection{}, err
	}
	for _, segment := range selectedPath {
		if strings.TrimSpace(segment) == "" {
			return HierarchyProjection{}, invalidProjection("hierarchy path contains an empty segment")
		}
	}
	pathCopy := append([]string{}, selectedPath...)
	projection := HierarchyProjection{
		Path:          pathCopy,
		Nodes:         []ProjectionNode{},
		Relationships: []ProjectionRelationship{},
	}
	nodes := make(map[string]ProjectionNode)
	moduleNodeIDs := make(map[string]string)
	for _, module := range model.Modules {
		if !hasHierarchyPrefix(module.Hierarchy, selectedPath) {
			continue
		}
		if len(module.Hierarchy) == len(selectedPath) {
			nodes[module.ID] = ProjectionNode{
				ID:        module.ID,
				Kind:      "module",
				Label:     module.DisplayName,
				Hierarchy: append([]string{}, module.Hierarchy...),
				ModuleIDs: []string{module.ID},
			}
			moduleNodeIDs[module.ID] = module.ID
			continue
		}
		groupPath := append([]string{}, module.Hierarchy[:len(selectedPath)+1]...)
		groupID := "group:" + strings.Join(groupPath, "/")
		node := nodes[groupID]
		if node.ID == "" {
			node = ProjectionNode{
				ID:        groupID,
				Kind:      "group",
				Label:     groupPath[len(groupPath)-1],
				Hierarchy: groupPath,
				ModuleIDs: []string{},
			}
		}
		node.ModuleIDs = appendUniqueProjectionID(node.ModuleIDs, module.ID)
		nodes[groupID] = node
		moduleNodeIDs[module.ID] = groupID
	}

	referenceNodeIDs := make(map[string]struct{})
	referencesByID := make(map[string]struct {
		Name string
	}, len(model.References))
	for _, reference := range model.References {
		referencesByID[reference.ID] = struct{ Name string }{Name: reference.Name}
	}
	relationships := make(map[string]ProjectionRelationship)
	for _, relationship := range model.Relationships {
		fromNodeID, fromVisible := moduleNodeIDs[relationship.FromModuleID]
		if !fromVisible {
			continue
		}
		toNodeID := ""
		if relationship.ToModuleID != "" {
			var targetVisible bool
			toNodeID, targetVisible = moduleNodeIDs[relationship.ToModuleID]
			if !targetVisible {
				continue
			}
		} else if relationship.ToReferenceID != "" {
			toNodeID = relationship.ToReferenceID
			if _, exists := referencesByID[relationship.ToReferenceID]; !exists {
				continue
			}
			if _, exists := referenceNodeIDs[toNodeID]; !exists {
				referenceNodeIDs[toNodeID] = struct{}{}
				reference := referencesByID[toNodeID]
				nodes[toNodeID] = ProjectionNode{ID: toNodeID, Kind: "reference", Label: reference.Name, ModuleIDs: []string{}}
			}
		}
		key := relationship.Type + "\x00" + fromNodeID + "\x00" + toNodeID
		projected, exists := relationships[key]
		if !exists {
			projected = ProjectionRelationship{
				ID:                 stableID("projection", relationship.Type, fromNodeID, toNodeID),
				Type:               relationship.Type,
				FromNodeID:         fromNodeID,
				ToNodeID:           toNodeID,
				RelationshipIDs:    []string{},
				SourceReferenceIDs: []string{},
			}
		}
		projected.RelationshipIDs = appendUniqueProjectionID(projected.RelationshipIDs, relationship.ID)
		for _, sourceID := range relationship.SourceReferenceIDs {
			projected.SourceReferenceIDs = appendUniqueProjectionID(projected.SourceReferenceIDs, sourceID)
		}
		projected.Count = len(projected.RelationshipIDs)
		sort.Strings(projected.RelationshipIDs)
		sort.Strings(projected.SourceReferenceIDs)
		relationships[key] = projected
	}

	for _, node := range nodes {
		sort.Strings(node.ModuleIDs)
		projection.Nodes = append(projection.Nodes, node)
	}
	sort.Slice(projection.Nodes, func(i, j int) bool { return projection.Nodes[i].ID < projection.Nodes[j].ID })
	for _, relationship := range relationships {
		projection.Relationships = append(projection.Relationships, relationship)
	}
	sort.Slice(projection.Relationships, func(i, j int) bool { return projection.Relationships[i].ID < projection.Relationships[j].ID })
	return projection, nil
}

func hasHierarchyPrefix(hierarchy, prefix []string) bool {
	if len(prefix) > len(hierarchy) {
		return false
	}
	for index, segment := range prefix {
		if hierarchy[index] != segment {
			return false
		}
	}
	return true
}

func appendUniqueProjectionID(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func invalidProjection(message string) error {
	return analysis.NewHostError(analysis.ErrInvalidRequest, message, nil)
}
