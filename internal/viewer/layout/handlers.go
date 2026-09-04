package layout

type optionHandler struct {
	targetScopes     []string
	rendererSupport  string
	enrich           func(*LayoutOptionDefinition)
	validate         func(LayoutOptionDefinition, any) error
	algorithmApplies func(LayoutOptionDefinition, string) bool
}

var layoutOptionHandlers = registerOptionHandlers(map[string]optionHandler{
	"org.eclipse.elk.direction":                                        genericOptionHandler("RIGHT", enumValues("RIGHT", "LEFT", "DOWN", "UP")),
	"org.eclipse.elk.edgeRouting":                                      genericOptionHandler("ORTHOGONAL", enumValues("NONE", "POLYLINE", "ORTHOGONAL", "SPLINES")),
	"org.eclipse.elk.aspectRatio":                                      genericOptionHandlerWithBounds("engine default", nil, numberPointer(0), nil, true, false),
	"org.eclipse.elk.spacing.nodeNode":                                 genericOptionHandlerWithBounds(35.0, nil, numberPointer(0), nil, false, false),
	"org.eclipse.elk.spacing.edgeNode":                                 genericOptionHandlerWithBounds(10.0, nil, numberPointer(0), nil, false, false),
	"org.eclipse.elk.spacing.edgeEdge":                                 genericOptionHandlerWithBounds(5.0, nil, numberPointer(0), nil, false, false),
	"org.eclipse.elk.layered.spacing.nodeNodeBetweenLayers":            genericOptionHandlerWithBounds(84.0, nil, numberPointer(0), nil, false, false),
	"org.eclipse.elk.layered.spacing.edgeNodeBetweenLayers":            genericOptionHandlerWithBounds(10.0, nil, numberPointer(0), nil, false, false),
	"org.eclipse.elk.layered.spacing.baseValue":                        genericOptionHandlerWithBounds("engine default", nil, numberPointer(0), nil, false, false),
	"org.eclipse.elk.layered.spacing.edgeEdgeBetweenLayers":            genericOptionHandlerWithBounds(10.0, nil, numberPointer(0), nil, false, false),
	"org.eclipse.elk.layered.layering.strategy":                        genericOptionHandler("NETWORK_SIMPLEX", enumValues("NETWORK_SIMPLEX", "LONGEST_PATH", "LONGEST_PATH_SOURCE", "COFFMAN_GRAHAM", "INTERACTIVE", "STRETCH_WIDTH", "MIN_WIDTH", "BF_MODEL_ORDER", "DF_MODEL_ORDER")),
	"org.eclipse.elk.layered.cycleBreaking.strategy":                   genericOptionHandler("GREEDY", enumValues("GREEDY", "DEPTH_FIRST", "INTERACTIVE", "MODEL_ORDER", "GREEDY_MODEL_ORDER", "SCC_CONNECTIVITY", "SCC_NODE_TYPE", "DFS_NODE_ORDER", "BFS_NODE_ORDER")),
	"org.eclipse.elk.layered.crossingMinimization.strategy":            genericOptionHandler("LAYER_SWEEP", enumValues("LAYER_SWEEP", "MEDIAN_LAYER_SWEEP", "INTERACTIVE", "NONE")),
	"org.eclipse.elk.layered.nodePlacement.strategy":                   genericOptionHandler("BRANDES_KOEPF", enumValues("SIMPLE", "INTERACTIVE", "LINEAR_SEGMENTS", "BRANDES_KOEPF", "NETWORK_SIMPLEX")),
	"org.eclipse.elk.layered.compaction.connectedComponents":           genericOptionHandler(false, nil),
	"org.eclipse.elk.priority":                                         genericOptionHandlerForTargets([]string{"NODES", "EDGES"}, "engine default", nil),
	"org.eclipse.elk.layered.priority.direction":                       genericOptionHandlerForTargets([]string{"EDGES"}, "engine default", nil),
	"org.eclipse.elk.layered.priority.shortness":                       genericOptionHandlerForTargets([]string{"EDGES"}, "engine default", nil),
	"org.eclipse.elk.layered.priority.straightness":                    genericOptionHandlerForTargets([]string{"EDGES"}, "engine default", nil),
	"org.eclipse.elk.layered.thoroughness":                             genericOptionHandlerWithBounds(7.0, nil, numberPointer(1), numberPointer(100), false, false),
	"org.eclipse.elk.layered.mergeEdges":                               genericOptionHandler(false, nil),
	"org.eclipse.elk.layered.mergeHierarchyEdges":                      genericOptionHandler(false, nil),
	"org.eclipse.elk.layered.feedbackEdges":                            genericOptionHandler(false, nil),
	"org.eclipse.elk.layered.crossingMinimization.forceNodeModelOrder": genericOptionHandler(false, nil),
	"org.eclipse.elk.separateConnectedComponents":                      genericOptionHandler(true, nil),
	"org.eclipse.elk.interactive":                                      genericOptionHandler(false, nil),
	"org.eclipse.elk.interactiveLayout":                                genericOptionHandler(false, nil),
	"org.eclipse.elk.randomSeed":                                       genericOptionHandlerWithBounds(1.0, nil, numberPointer(0), nil, false, false),
}, usefulSettingsHandlers())

func genericOptionHandler(defaultValue any, allowedValues []string) optionHandler {
	return genericOptionHandlerForTargets([]string{"PARENTS"}, defaultValue, allowedValues)
}

func genericOptionHandlerWithBounds(defaultValue any, allowedValues []string, minimum, maximum *float64, minimumExclusive, maximumExclusive bool) optionHandler {
	return genericOptionHandlerForTargetsWithBounds([]string{"PARENTS"}, defaultValue, allowedValues, minimum, maximum, minimumExclusive, maximumExclusive)
}

func genericOptionHandlerForTargets(targetScopes []string, defaultValue any, allowedValues []string) optionHandler {
	return genericOptionHandlerForTargetsWithBounds(targetScopes, defaultValue, allowedValues, nil, nil, false, false)
}

func genericOptionHandlerForTargetsWithBounds(targetScopes []string, defaultValue any, allowedValues []string, minimum, maximum *float64, minimumExclusive, maximumExclusive bool) optionHandler {
	rendererSupport := "supported"
	return optionHandler{
		targetScopes:    append([]string(nil), targetScopes...),
		rendererSupport: rendererSupport,
		enrich: func(option *LayoutOptionDefinition) {
			if allowedValues != nil {
				option.AllowedValues = make([]any, len(allowedValues))
				for index, value := range allowedValues {
					option.AllowedValues[index] = value
				}
			}
			option.Minimum = minimum
			option.Maximum = maximum
			option.MinimumExclusive = minimumExclusive
			option.MaximumExclusive = maximumExclusive
			if !layoutOptionTargetsAll(*option, targetScopes) {
				return
			}
			option.DefaultValue = defaultValue
			option.Editable = true
			option.RendererSupport = rendererSupport
		},
		validate:         validateCatalogOptionValue,
		algorithmApplies: catalogOptionApplies,
	}
}

func enumValues(values ...string) []string {
	return values
}

func numberPointer(value float64) *float64 {
	return &value
}

func enrichLayoutOption(option LayoutOptionDefinition) LayoutOptionDefinition {
	option = cloneLayoutOption(option)
	if option.Description == "" {
		option.Description = "ELK layout option: " + option.Name + "."
	}
	if option.AllowedValues == nil {
		option.AllowedValues = []any{}
	}
	option.DefaultValue = "engine default"
	option.Editable = false
	option.RendererSupport = "unsupported"
	if handler, ok := layoutOptionHandlers[option.ID]; ok && handler.enrich != nil {
		handler.enrich(&option)
	}
	return option
}

func layoutOptionTargets(option LayoutOptionDefinition, target string) bool {
	for _, candidate := range option.Targets {
		if candidate == target {
			return true
		}
	}
	return false
}

func layoutOptionTargetsParent(option LayoutOptionDefinition) bool {
	return layoutOptionTargets(option, "PARENTS")
}

func layoutOptionTargetsAll(option LayoutOptionDefinition, targets []string) bool {
	for _, target := range targets {
		if !layoutOptionTargets(option, target) {
			return false
		}
	}
	return true
}

func catalogOptionApplies(option LayoutOptionDefinition, algorithm string) bool {
	if len(option.Algorithms) == 0 {
		return true
	}
	for _, candidate := range option.Algorithms {
		if candidate == "all" || candidate == algorithm {
			return true
		}
	}
	return false
}
