# Go analysis acceptance scenarios

## SC-GO-001 — Discover module packages

Given a valid `go.mod` module with two package directories, when analysis runs, then both packages are returned as local module observations with source evidence.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-GO-002 — Resolve local imports

Given package A imports package B inside the selected module, when analysis runs, then one `depends_on` observation connects A to B with import-location evidence.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-GO-003 — Handle workspace ambiguity

Given `go.work` exposes multiple modules and no module option is supplied, when analysis starts, then it returns `AmbiguousWorkspace` and does not merge modules silently.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-GO-004 — Exclude defaults

Given test, vendor, generated, cache, and directories named `external`, when default analysis runs, then they are excluded; the `include_external` option affects only the detail retained for non-local imports and does not include excluded directories.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-GO-005 — Report non-local imports

Given standard-library, third-party, cgo, and missing imports, when analysis runs, then they are classified as references/diagnostics and never fabricated as local packages.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.
