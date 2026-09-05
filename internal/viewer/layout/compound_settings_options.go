package layout

// Compound layout owns the actual hierarchy policy. Expose only the verified
// single-run value so a profile cannot request geometry the renderer does not
// normalize.
func compoundSettingsHandlers() map[string]optionHandler {
	return map[string]optionHandler{
		"org.eclipse.elk.hierarchyHandling": genericOptionHandlerForTargets(
			[]string{"PARENTS"}, "INCLUDE_CHILDREN", enumValues("INCLUDE_CHILDREN"),
		),
	}
}
