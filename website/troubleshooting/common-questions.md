# Common questions

## The browser is stuck on “Loading analysis scope”

The browser is waiting for the local server to return the model or scope list. It is not always a slow operation.

Check the terminal that started Arch View. Look for an error, a panic, or a request that never completes. Stop the command with Ctrl+C and run it again with the same project path.

If the problem repeats, save:

- the exact command;
- the complete terminal output;
- the project language and marker files;
- whether you opened a project or a model.

## A check says unsupported

Unsupported means the current analyzer did not report the fact required by the rule. It does not mean the rule found a problem and it does not mean the code is wrong.

Choose a language/analyzer that supports the required capability, or turn off the rule for that project.

## A check says not evaluable

Not evaluable means the rule exists, but the current report lacks a necessary input. Examples include a missing explicit architecture policy or missing layer assignments.

Read the reason beside the status. It should tell you which input is missing.

## Documentation is reported as missing

The documentation check uses documentation records from the analyzer. If the source has a doc comment but the report says it is absent, first check:

1. that the symbol is in the selected scope;
2. that the analyzer supports documentation extraction for that language and symbol;
3. that the report was generated after the source changed;
4. the symbol's evidence and source location.

Do not add a second comment before checking whether the analyzer saw the first one.

## The viewer shows only the first 25 results

This is a browser safety boundary. Use **Load more** or pagination. For automation, use the bounded query or export the JSON report.

## Can I save every setting?

No. Session settings may be temporary. Project-backed sessions can save supported layout and quality configuration. Model-only sessions cannot write to the original project.

The viewer shows whether Save and Save As are available.

## Can Arch View fix a finding?

No. Arch View reports observations. A developer or coding agent must decide whether to change code, change configuration, or record a justified baseline.

## The OKF link is missing

The link appears only when the project-backed server discovers at least one selectable OKF bundle. Refresh the project session and inspect the OKF catalog diagnostics. An invalid, unreadable, out-of-boundary, or empty bundle is not selectable.

The Architecture viewer remains available when OKF discovery fails. Model-only sessions and the self-contained architecture demo do not provide project-backed OKF discovery or persistence.

## Why are semantic links not visible?

Semantic links are informational Markdown cross-references. They are hidden until you select a concept, then only that concept's links are shown. They are dashed and arrowless and never affect the layout. Solid arrowed lines represent containment.

## Why is an advanced layout feature unavailable?

The layout dialog distinguishes cataloged ELK options from features the current renderer can apply. The current advanced features are verified for Layered. Some features also require a specific option, such as `SPLINES` routing for spline refinement. Select a compatible algorithm and option combination before applying the feature.

If saved preferences refer to unavailable geometry, Arch View retains the preference, reports a diagnostic, and uses safe ordinary geometry where possible.

## A profile save reports a conflict

Another operation changed the project configuration after the profile was read. Reload the profile and review the current values before saving again. Save operations validate first, use revision and idempotency checks, and leave the previous file intact when a write fails.
