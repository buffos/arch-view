package model

import "sort"

type graphEdge struct {
	RelationshipID string
	From           string
	To             string
}

func deriveGraph(model *Model) {
	moduleIDs := make([]string, 0, len(model.Modules))
	for _, module := range model.Modules {
		moduleIDs = append(moduleIDs, module.ID)
	}
	sort.Strings(moduleIDs)
	adjacency := make(map[string][]graphEdge, len(moduleIDs))
	for _, moduleID := range moduleIDs {
		adjacency[moduleID] = []graphEdge{}
	}
	for _, relationship := range model.Relationships {
		if relationship.Type != "depends_on" || relationship.ToModuleID == "" {
			continue
		}
		adjacency[relationship.FromModuleID] = append(adjacency[relationship.FromModuleID], graphEdge{
			RelationshipID: relationship.ID,
			From:           relationship.FromModuleID,
			To:             relationship.ToModuleID,
		})
	}
	for moduleID := range adjacency {
		sort.Slice(adjacency[moduleID], func(i, j int) bool {
			return adjacency[moduleID][i].RelationshipID < adjacency[moduleID][j].RelationshipID
		})
	}

	components := stronglyConnectedComponents(moduleIDs, adjacency)
	cycles := make([]CycleGroup, 0)
	for _, component := range components {
		if !isCyclicComponent(component, adjacency) {
			continue
		}
		moduleSet := make(map[string]struct{}, len(component))
		for _, moduleID := range component {
			moduleSet[moduleID] = struct{}{}
		}
		relationshipIDs := []string{}
		for _, moduleID := range component {
			for _, edge := range adjacency[moduleID] {
				if _, exists := moduleSet[edge.To]; exists {
					relationshipIDs = appendUniqueRelationshipID(relationshipIDs, edge.RelationshipID)
				}
			}
		}
		sort.Strings(component)
		sort.Strings(relationshipIDs)
		cycles = append(cycles, CycleGroup{
			ID:              stableID("cycle", component...),
			ModuleIDs:       component,
			RelationshipIDs: relationshipIDs,
		})
	}
	sort.Slice(cycles, func(i, j int) bool { return cycles[i].ID < cycles[j].ID })

	feedback := selectFeedbackEdges(cycles, adjacency)
	model.Derived.Cycles = cycles
	model.Derived.FeedbackRelationshipIDs = feedback
	model.Derived.Layers = dependencyLayers(moduleIDs, adjacency, feedback)
	if model.Derived.AlgorithmProvenance == nil {
		model.Derived.AlgorithmProvenance = map[string]any{}
	}
	model.Derived.AlgorithmProvenance["relation_type"] = "depends_on"
}

func stronglyConnectedComponents(moduleIDs []string, adjacency map[string][]graphEdge) [][]string {
	index := 0
	indices := make(map[string]int, len(moduleIDs))
	lowLinks := make(map[string]int, len(moduleIDs))
	onStack := make(map[string]bool, len(moduleIDs))
	stack := []string{}
	components := [][]string{}
	var visit func(string)
	visit = func(moduleID string) {
		index++
		indices[moduleID] = index
		lowLinks[moduleID] = index
		stack = append(stack, moduleID)
		onStack[moduleID] = true
		for _, edge := range adjacency[moduleID] {
			if _, seen := indices[edge.To]; !seen {
				visit(edge.To)
				if lowLinks[edge.To] < lowLinks[moduleID] {
					lowLinks[moduleID] = lowLinks[edge.To]
				}
			} else if onStack[edge.To] && indices[edge.To] < lowLinks[moduleID] {
				lowLinks[moduleID] = indices[edge.To]
			}
		}
		if lowLinks[moduleID] != indices[moduleID] {
			return
		}
		component := []string{}
		for {
			last := len(stack) - 1
			member := stack[last]
			stack = stack[:last]
			onStack[member] = false
			component = append(component, member)
			if member == moduleID {
				break
			}
		}
		sort.Strings(component)
		components = append(components, component)
	}
	for _, moduleID := range moduleIDs {
		if _, seen := indices[moduleID]; !seen {
			visit(moduleID)
		}
	}
	return components
}

func isCyclicComponent(component []string, adjacency map[string][]graphEdge) bool {
	if len(component) > 1 {
		return true
	}
	if len(component) == 0 {
		return false
	}
	for _, edge := range adjacency[component[0]] {
		if edge.To == component[0] {
			return true
		}
	}
	return false
}

func selectFeedbackEdges(cycles []CycleGroup, adjacency map[string][]graphEdge) []string {
	feedback := []string{}
	for _, cycle := range cycles {
		members := make(map[string]struct{}, len(cycle.ModuleIDs))
		for _, moduleID := range cycle.ModuleIDs {
			members[moduleID] = struct{}{}
		}
		kept := make(map[string][]string, len(members))
		for moduleID := range members {
			kept[moduleID] = []string{}
		}
		edges := []graphEdge{}
		for _, moduleID := range cycle.ModuleIDs {
			for _, edge := range adjacency[moduleID] {
				if _, exists := members[edge.To]; exists {
					edges = append(edges, edge)
				}
			}
		}
		sort.Slice(edges, func(i, j int) bool { return edges[i].RelationshipID < edges[j].RelationshipID })
		for _, edge := range edges {
			if edge.From == edge.To || reaches(kept, edge.To, edge.From) {
				feedback = appendUniqueRelationshipID(feedback, edge.RelationshipID)
				continue
			}
			kept[edge.From] = append(kept[edge.From], edge.To)
		}
	}
	sort.Strings(feedback)
	return feedback
}

func dependencyLayers(moduleIDs []string, adjacency map[string][]graphEdge, feedback []string) []Layer {
	feedbackSet := make(map[string]struct{}, len(feedback))
	for _, relationshipID := range feedback {
		feedbackSet[relationshipID] = struct{}{}
	}
	depth := make(map[string]int, len(moduleIDs))
	visiting := make(map[string]bool, len(moduleIDs))
	var visit func(string) int
	visit = func(moduleID string) int {
		if value, ok := depth[moduleID]; ok {
			return value
		}
		if visiting[moduleID] {
			return 0
		}
		visiting[moduleID] = true
		value := 0
		for _, edge := range adjacency[moduleID] {
			if _, removed := feedbackSet[edge.RelationshipID]; removed {
				continue
			}
			candidate := visit(edge.To) + 1
			if candidate > value {
				value = candidate
			}
		}
		visiting[moduleID] = false
		depth[moduleID] = value
		return value
	}
	byLayer := make(map[int][]string)
	for _, moduleID := range moduleIDs {
		layer := visit(moduleID)
		byLayer[layer] = append(byLayer[layer], moduleID)
	}
	layers := make([]Layer, 0, len(byLayer))
	for layer, values := range byLayer {
		sort.Strings(values)
		layers = append(layers, Layer{Layer: layer, ModuleIDs: values})
	}
	sort.Slice(layers, func(i, j int) bool { return layers[i].Layer < layers[j].Layer })
	return layers
}

func reaches(adjacency map[string][]string, from, target string) bool {
	if from == target {
		return true
	}
	visited := map[string]struct{}{from: {}}
	queue := []string{from}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacency[current] {
			if next == target {
				return true
			}
			if _, seen := visited[next]; seen {
				continue
			}
			visited[next] = struct{}{}
			queue = append(queue, next)
		}
	}
	return false
}

func appendUniqueRelationshipID(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
