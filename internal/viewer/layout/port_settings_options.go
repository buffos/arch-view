package layout

// Presentation ports own their role-specific sides. The profile may expose
// the verified node constraint, but cannot rewrite individual generated ports.
func portSettingsHandlers() map[string]optionHandler {
	return map[string]optionHandler{
		"org.eclipse.elk.portConstraints": genericOptionHandlerForTargets(
			[]string{"NODES"}, "FIXED_SIDE", enumValues("FIXED_SIDE"),
		),
	}
}
