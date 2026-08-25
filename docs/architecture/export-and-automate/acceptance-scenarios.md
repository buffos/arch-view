# Export and automate acceptance scenarios

## SC-EXPT-001 — Export canonical JSON

Given a valid complete model, when JSON export runs, then a versioned JSON artifact contains the model, evidence, diagnostics, cycles, layers, status, and deterministic ordering.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-EXPT-002 — Export partial model

Given a valid partial model, when JSON/HTML/SVG export runs, then the artifact is written with visible partial status and diagnostics and the process exits `0`.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EXPT-003 — Produce self-contained HTML

Given a model and HTML format, when export completes, then the file opens without network access and exposes overview, navigation, evidence, cycle/diagnostic states, and accessible list/details mode.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EXPT-004 — Produce accessible SVG

Given a model and SVG format, when export completes, then the artifact is scalable, script-free, has accessible metadata/labels, stable IDs, and deterministic geometry.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EXPT-005 — Repeat deterministically

Given identical model, view, renderer version, and options, when export runs twice, then bytes/content hash match.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-EXPT-006 — Reject invalid input

Given an invalid model or unsupported option such as source embedding in v1, when export runs, then no claimed artifact is produced and a non-zero stable error is returned.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-EXPT-007 — Protect existing output

Given an existing target file and no `--overwrite`, when export runs, then it refuses without truncating the existing artifact; with `--overwrite`, replacement is atomic.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-EXPT-008 — Reproducible CI run

Given a fixed project/model, options, tool version, and output target, when CI invokes headless export, then artifacts and exit status are reproducible and diagnostics are machine-readable.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.
