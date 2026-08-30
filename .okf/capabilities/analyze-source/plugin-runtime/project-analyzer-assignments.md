---
type: capability
title: Project analyzer assignments and view selection
description: Persist project-relative analyzer assignments and let users switch the visible analyzer scope in the application.
tags: [plugins, configuration, selection, viewer, multi-language]
timestamp: 2026-08-29T00:00:00Z
state: implemented
state_changed: 2026-08-29T00:00:00Z
project: /project.md
parent: /capabilities/analyze-source/plugin-runtime.md
artifact_root: docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments
discovery_notes: docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/orchestration-status.md
prd: docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/prd.md
glossary: docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/domain-glossary.md
domain_model: docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/canonical-domain-model.md
use_cases: docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/canonical-use-cases.md
contract: docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/canonical-api-cli-contract.md
scenarios: docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/acceptance-scenarios.md
readiness_review: docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/readiness-review.md
issues:
  - docs/agents/issues/done/20260829-044-load-and-validate-analysis-configuration.md
  - docs/agents/issues/done/20260829-045-resolve-configured-assignments-and-source-scopes.md
  - docs/agents/issues/done/20260829-046-session-cache-and-selective-invalidation.md
  - docs/agents/issues/done/20260829-047-configured-scope-viewer-journey.md
---

# Intent

Give users deterministic control over which analyzer owns each project folder
and which analyzer result is visible in the architecture application.

# Scope

This capability owns project-relative folder or project-root assignments and
invocation-root source-scope filters in a separate `analysis` section of
`.archview.json`, configuration precedence and validation, logical analyzer
selection, and the application control that switches between individual
analyzer/project scopes or the combined view. The viewer remains responsible
for the presentation surface; [Explore and inspect architecture](/capabilities/explore-architecture.md)
consumes the selected scope without owning analyzer semantics.

# Relationships

- Parent: [Analyzer plugin runtime](/capabilities/analyze-source/plugin-runtime.md)
- Viewer consumer: [Explore and inspect architecture](/capabilities/explore-architecture.md)
- Input: [Analyze source code](/capabilities/analyze-source.md)

# Planning state

This child is implemented after issues 044–047 were completed, verified, and
archived, including the declared visual review for the configured viewer. The
user-confirmed target is a
configuration mapping from repository-relative folders to stable logical
analyzer IDs, with automatic detection as a fallback and a dropdown that
switches among the resulting analyzer/project scopes. The cached `All` and
individual-scope viewer control is implemented by the sibling multi-analyzer
capability; this child owns the persisted assignment and source-scope rules
that supply those scopes. The nearest discovered configuration remains
the one complete file; within it, the deepest matching assignment wins,
explicit CLI selection overrides assignments, and invalid nearest
configuration is surfaced instead of bypassed. The v2 schema also defines
global exclusion globs and analyzer-ID-scoped include globs relative to the
invocation root; filtering follows root discovery and exclusions win. The
exact v1/v2 schema, precedence, validation, cache keys, invalidation, API, UI
states, and acceptance scenarios are linked above. The combined model is
backed by cached per-job results so scope switching does not re-run analysis.
