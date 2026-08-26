# 005 — Deterministic JSON, HTML, and SVG export

Execution type: AFK
Review gate: visual-review
Status: done

## Parent PRD

docs/architecture/export-and-automate/prd.md

## What to build

Implement repeatable artifacts from the same validated model and renderer-neutral view used by the local viewer.

- Implement arch-view export for JSON, HTML, and SVG and wire the analyze-to-export path.
- Preserve the arch-view.model/v1 envelope, analysis status, diagnostics, evidence IDs, cycles, layers, provenance, and stable ordering.
- Reuse the viewer's reference-visibility policy for visual artifacts: JSON always retains canonical references, while HTML/SVG overview output is local-first by default and supports explicit aggregated/expanded reference views.
- Preserve an accessible imports/evidence list in HTML and equivalent reference-scope metadata in SVG without turning every import into an overview node.
- Produce self-contained HTML with embedded model/view data and no network dependency.
- Produce script-free accessible SVG with stable module/relationship attributes, titles, descriptions, cycle/diagnostic styling, and deterministic geometry.
- Refuse existing outputs unless overwrite is explicit, write atomically, and return the specified status and exit codes.
- Reject source embedding and invalid models clearly; do not parse source in the exporter.

## Acceptance criteria

- [x] JSON export is versioned, schema-valid, byte-stable for identical input/options, and preserves partial status and diagnostics.
- [x] HTML export is self-contained, opens without network access, and exposes overview, hierarchy, search, evidence/details, cycle/diagnostic states, and accessible list/details mode.
- [x] HTML/SVG exports preserve the local-first reference policy, expose standard-library/external/unresolved/dynamic scope and confidence distinctly, and keep individual imports available through the appropriate details/metadata path.
- [x] SVG export is scalable, script-free, accessible, deterministic, and includes stable module/relationship identifiers and recorded layout provenance.
- [x] Repeated exports from identical models and options produce identical bytes and equivalent semantics across CLI; no HTTP export entrypoint exists in the current boundary.
- [x] Existing output is protected unless overwrite is supplied; writes are atomic and failed writes do not leave a misleading completed artifact.
- [x] Invalid models, unsupported source embedding, invalid invocation, render/write failures, and cancellation map to the specified exit/status behavior.
- [x] A visual review confirms HTML/SVG parity with the local viewer for overview, relationships, cycles, diagnostics, and labels.

## Artifact sync required

- Application PRD: none — JSON, HTML, and SVG are already in the specified MVP.
- Application architecture summary: none — exporters continue to consume the existing model/view contract.
- Owning capability node/artifacts: required: .okf/capabilities/export-and-automate.md; .okf/capabilities/explore-architecture.md; docs/architecture/export-and-automate/orchestration-status.md; docs/architecture/explore-architecture/orchestration-status.md; docs/architecture/analyze-source/go-analysis/implementation-slice.md.
- Issue registry: required; node issues reference required: .okf/capabilities/export-and-automate.md; .okf/capabilities/explore-architecture.md.
- Reason/no-impact decision: delivery truth is being added; no product or architecture decision changes, and source contents remain excluded from v1 artifacts.

## Blocked by

—

## User stories addressed

- US-EXP-001
- US-EXP-002
- US-EX-001

## Contract and scenario trace

- Contract: docs/architecture/export-and-automate/canonical-api-cli-contract.md; docs/architecture/explore-architecture/canonical-api-cli-contract.md
- Scenarios: SC-EXPT-001, SC-EXPT-002, SC-EXPT-003, SC-EXPT-004, SC-EXPT-005, SC-EXPT-006, SC-EXPT-007, SC-EXPT-008, SC-EXPT-009

## Scenario traceability and verification plan

| Source rule / use case | Scenario | Issue criterion | Planned verification | State |
|---|---|---|---|---|
| Canonical JSON envelope, model validation, and stable ordering | SC-EXPT-001, SC-EXPT-005 | JSON is versioned, valid, byte-stable, and preserves model facts | Export package determinism/validation tests and canonical JSON decode/validate | implemented |
| Complete/partial status and diagnostics remain visible | SC-EXPT-002 | JSON/HTML/SVG preserve partial status and diagnostics | Partial-model export tests plus CLI analyze-to-export test | implemented |
| ExportService reuses the renderer-neutral view and local-first policy | SC-EXPT-003, SC-EXPT-009 | HTML is self-contained and visual exports preserve reference visibility/scope/confidence | Embedded HTML asset/no-network assertions and hidden/aggregated/expanded scene tests | implemented |
| Static SVG accessibility, stable identifiers, and layout provenance | SC-EXPT-004 | SVG is scalable, script-free, accessible, deterministic, and traceable | SVG structure/attribute/metadata tests and repeated-byte comparison | implemented |
| Atomic artifact writing and overwrite protection | SC-EXPT-007 | Existing output is protected and replacement is atomic | Export writer tests for refusal, replacement, and failed-write cleanup | implemented |
| Invalid input/options and headless CLI status semantics | SC-EXPT-006, SC-EXPT-008 | Invalid requests fail without artifacts and repeatable CLI runs expose machine-readable status | CLI exit-code tests, unsupported source-embedding test, and project analyze-to-export integration | implemented |

## Verification performed

- `go test ./... -count=1` passed.
- `go test -race ./...` passed.
- `go vet ./...` passed.
- `go build ./...` passed.
- `staticcheck ./...` passed.
- `golangci-lint run` passed with `0 issues`.
- `node --check internal/viewer/web/app.js` passed.
- `git diff --check` passed; Git reported only its normal LF-to-CRLF warning for Windows working copies.
- CLI tests cover `analyze --format html`, `export --format json|html|svg`, partial status, invalid source embedding, deterministic bytes, visibility policy, and output protection.
- No HTTP export endpoint exists in the current product boundary, so parity is verified at the shared export service and CLI entrypoint; HTTP parity remains a future adapter concern.

## Review handoff

The user explicitly approved the visual review in the current task. The generated HTML/SVG were reviewed against the approved local viewer for overview, hierarchy, relationships, cycles, diagnostics, reference visibility, and labels. The implementation, verification, and required artifact synchronization are complete; this issue is ready for normal closeout.
