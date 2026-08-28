# Advanced ELK renderer support PRD

## Purpose

Extend Arch View's renderer-only layout surface with a small, explicit set of
ELK geometry features while preserving semantic scene meaning, accessibility,
and deterministic fallback behavior.

## Actors

- **Developer:** selects a presentation feature and inspects the resulting architecture view.
- **CI/documentation operator:** produces browser or HTML artifacts and expects stable semantic/layout behavior.
- **Layout adapter:** negotiates supported features with the pinned ELK runtime and returns validated geometry.
- **Renderer/exporter:** consumes the same geometry without inventing architecture facts.

## Goals

1. Support opt-in edge labels, junction points, presentation ports, compound geometry, and spline refinement.
2. Share one versioned geometry contract across the live browser, self-contained HTML, and browser Download SVG.
3. Keep canonical model, semantic scene, analyzer, and project-assignment semantics unchanged.
4. Make unsupported or malformed ELK output degrade to deterministic orthogonal geometry with a visible diagnostic.
5. Preserve accessible relationship text, hierarchy, evidence, and reading order.
6. Keep Go static SVG deterministic orthogonal unless a future specification explicitly extends it.

## Non-goals

- New canonical relationship types, hyperedges, or analyzer facts.
- User-authored manual port graphs or spline control-point editing.
- Exposing the entire pinned ELK catalog as implemented behavior.
- Pixel-identical parity between browser layout and deterministic Go static SVG.
- Canvas/WebGL-specific behavior; future renderers consume the same geometry contract.

## Feature set

| Feature | Default | Exact behavior |
|---|---:|---|
| `edge_labels` | off | Supplies deterministic count labels to ELK, accepts label bounds, and renders label text without changing relationship semantics. |
| `junctions` | off | Accepts finite shared junction coordinates from layered ELK and renders presentation markers only when at least two visible edge routes share the point. |
| `ports` | off | Adds deterministic presentation-only `in`/`out` ports for visible nodes and routes endpoints through them. |
| `compound` | off | Sends visible hierarchy as nested layout containers and accepts parent/child bounds and cross-hierarchy routes. |
| `spline_refinement` | off | Accepts finite, connected piecewise-cubic sections and preserves them through browser/HTML/browser-SVG rendering. |

Features are enabled through the versioned `layout.features` array. An empty
array preserves the current flat scene and route behavior.

## Functional requirements

| ID | Requirement |
|---|---|
| AER-FR-001 | Expose only the five admitted presentation features as selectable; broader catalog entries remain catalog-only. |
| AER-FR-002 | Produce `arch-view.geometry/v1` with deterministic node, port, label, junction, compound, and route identity. |
| AER-FR-003 | Preserve one-to-one mapping from geometry edges to visible semantic relationship IDs. |
| AER-FR-004 | Validate finite coordinates, connected sections, valid bounds, and feature-specific references before rendering. |
| AER-FR-005 | Use deterministic orthogonal fallback for unsupported/malformed feature output and expose a geometry diagnostic. |
| AER-FR-006 | Keep browser, self-contained HTML, and browser Download SVG on the same geometry and feature policy. |
| AER-FR-007 | Keep Go static SVG on deterministic orthogonal output and report ignored advanced features in layout provenance. |
| AER-FR-008 | Preserve semantic accessibility through relationship descriptions, reading order, evidence links, and hierarchy details. |
| AER-FR-009 | Keep `layout.features` separate from analyzer assignments/options and canonical model data. |

## Non-functional requirements

- Geometry serialization is deterministic for equal scene, profile, pinned runtime, and feature output.
- Invalid geometry cannot produce an invalid SVG path or crash the viewer.
- Geometry-only objects cannot be selected as architecture relationships.
- Feature diagnostics are actionable and do not hide the valid semantic scene.

## Success criteria

The scenarios in [acceptance-scenarios.md](acceptance-scenarios.md) pass for
each admitted feature, mixed-feature combinations, malformed output, browser/
HTML/browser-SVG parity, static SVG fallback, and accessibility.
