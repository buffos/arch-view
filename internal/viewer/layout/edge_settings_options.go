package layout

// Advanced edge controls are registered only after their browser geometry
// handlers have passed the pinned-runtime acceptance fixtures.
func advancedEdgeSettingsHandlers() map[string]optionHandler {
	return map[string]optionHandler{
		"org.eclipse.elk.edgeLabels.placement": genericOptionHandlerForTargets(
			[]string{"LABELS"}, "engine default", enumValues("HEAD", "CENTER", "TAIL"),
		),
		"org.eclipse.elk.layered.edgeRouting.splines.mode": genericOptionHandler(
			"CONSERVATIVE", enumValues("CONSERVATIVE", "SLOPPY"),
		),
	}
}
