package layout

import "github.com/buffo/arch-view/internal/analysis"

// Register the bounded, pinned-runtime-tested settings tranche. Values that
// crash this pinned Mr. Tree implementation are deliberately not admitted.
func usefulSettingsHandlers() map[string]optionHandler {
	return map[string]optionHandler{
		"org.eclipse.elk.mrtree.edgeRoutingMode":     withSupportNote(genericOptionHandler("AVOID_OVERLAP", enumValues("MIDDLE_TO_MIDDLE", "AVOID_OVERLAP")), "NONE is unavailable: the pinned runtime fails to produce a route."),
		"org.eclipse.elk.mrtree.searchOrder":         withSupportNote(genericOptionHandler("DFS", enumValues("DFS")), "BFS is unavailable: the pinned runtime overflows on graphs with shared descendants."),
		"org.eclipse.elk.mrtree.weighting":           withSupportNote(genericOptionHandler("MODEL_ORDER", enumValues("MODEL_ORDER", "DESCENDANTS", "FAN")), "CONSTRAINT needs per-node position constraints, which are not implemented."),
		"org.eclipse.elk.spacing.componentComponent": genericOptionHandlerWithBounds("engine default", nil, numberPointer(0), nil, false, false),
		"org.eclipse.elk.padding":                    paddingHandler(),
	}
}

func paddingHandler() optionHandler {
	handler := genericOptionHandler("engine default", nil)
	enrich := handler.enrich
	handler.enrich = func(option *LayoutOptionDefinition) {
		enrich(option)
		option.Control = "padding"
		option.SupportedTargets = []string{"PARENTS"}
		option.Algorithms = []string{"layered", "mrtree"}
	}
	handler.validate = func(option LayoutOptionDefinition, value any) error {
		padding, ok := value.(map[string]any)
		if !ok || len(padding) != 4 {
			return invalidPadding(option.ID)
		}
		for _, side := range []string{"top", "right", "bottom", "left"} {
			number, ok := jsonNumber(padding[side])
			if !ok || !isFiniteNumber(number) || number < 0 || number > 10000 {
				return invalidPadding(option.ID)
			}
		}
		return nil
	}
	return handler
}

func withSupportNote(handler optionHandler, note string) optionHandler {
	enrich := handler.enrich
	handler.enrich = func(option *LayoutOptionDefinition) { enrich(option); option.SupportNote = note }
	return handler
}

func invalidPadding(id string) error {
	return analysis.NewHostError(analysis.ErrInvalidOptions, "Graph padding requires finite top, right, bottom and left numbers between 0 and 10000.", map[string]any{"option": id})
}

func registerOptionHandlers(base, extra map[string]optionHandler) map[string]optionHandler {
	for id, handler := range extra {
		if _, exists := base[id]; exists {
			panic("duplicate layout option handler: " + id)
		}
		base[id] = handler
	}
	return base
}
