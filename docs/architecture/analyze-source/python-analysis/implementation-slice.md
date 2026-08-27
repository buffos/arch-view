# Python implementation slice: repository to visible architecture view

## Selected frontier

The [Python analysis capability](../../../../.okf/capabilities/analyze-source/python-analysis.md) implementation frontier is complete, using the existing [analyzer plugin runtime](../../../../.okf/capabilities/analyze-source/plugin-runtime.md), canonical model, viewer, and export contracts. Issues 017 through 019 are archived after automated verification and the declared visual review.

The map has no foggy or bounded nodes. Python is the first non-Go language in the agreed implementation sequence, its exact specification is readiness-reviewed, and the host/model/presentation seams needed for a visible result already exist. This slice therefore adds one language adapter without changing host orchestration, canonical model semantics, layout, or renderer code.

## Current and target truth

- **Observed in code:** the generic `analysis.Analyzer` contract, deterministic host selection, canonical normalization, scene projection, local viewer, and JSON/HTML/SVG paths exist; the built-in Go analyzer and the Python project/module-discovery/static-import adapter are registered.
- **Inferred from docs:** the Python child contract defines static project/configuration discovery, package/module hierarchy, absolute and relative import resolution, dynamic-import uncertainty, evidence, diagnostics, and safe read-only behavior.
- **User-confirmed target:** languages are added gradually behind the common plugin contract; adding Python must not require host language switches or changes to the model/viewer.
- **Mismatch:** none. The Python project/module-discovery and import-uncertainty foundation and the public visible journey are implemented and visually approved.

## Vertical outcome

Given a Python repository, a developer can:

1. select or auto-detect the Python analyzer;
2. discover configured, `src/`, regular-package, and namespace-package modules without importing or executing the repository;
3. see deterministic absolute and relative static import relationships with source evidence;
4. see unresolved, external, standard-library, conditional, and dynamic imports as references, confidence, or diagnostics rather than fabricated local edges;
5. open the resulting language-neutral model in the existing viewer and inspect hierarchy/import evidence; and
6. produce deterministic JSON, self-contained HTML, and SVG artifacts through the existing model/export paths.

## Scope

- In-process Python analyzer implementing `analysis.Analyzer`.
- Static project-marker and source-root resolution using `pyproject.toml`, `setup.cfg`, and `setup.py` metadata without executing `setup.py`.
- Package/module discovery for `.py` files and opt-in `.pyi` stubs, with configured exclusions and deterministic source evidence.
- Absolute and relative import extraction and resolution where the filesystem/configuration proves the target.
- Standard-library, external, unresolved, conditional, and dynamic-import classification with confidence and recoverable diagnostics.
- Composition-root registration, public analyzer selection, Python-specific options, canonical model normalization, and existing viewer/export smoke coverage.

## Explicit non-goals

- Executing, importing, installing, or introspecting target Python code.
- Environment-assisted `importlib` resolution, type checking, call graphs, symbol graphs, or runtime plugin discovery.
- TypeScript, Rust, Clojure, or external NDJSON process analyzers.
- Changes to the canonical model JSON shape, layout configuration, renderer routing, or the upstream [reference repository](https://github.com/unclebob/arch-view).

## Ordered delivery issues

| Issue | Outcome | Owner | Blocked by | Review gate |
|---|---|---|---|---|
| [017](../../../agents/issues/done/20260827-017-python-project-boundary-and-module-discovery.md) | Registered Python analyzer, project/configuration boundary, source scope, package/module discovery, and deterministic module-only analysis path | Python analysis + plugin composition | None | none |
| [018](../../../agents/issues/done/20260827-018-python-static-import-resolution-and-uncertainty.md) | Absolute/relative import relationships, references, evidence, dynamic/conditional diagnostics, confidence, and partial results | Python analysis | 017 | none |
| [019](../../../agents/issues/done/20260827-019-python-cli-and-visible-architecture-path.md) | Public Python options and end-to-end analyze → model → viewer/export integration, including corrected windowed Fit behavior | Python analysis + existing model/viewer/export consumers | 018 | visual-review complete |

## Slice acceptance

- A representative Python project reaches a visible architecture view through the existing CLI and local viewer.
- `src/`, regular-package, namespace-package, stub, test, exclusion, and project-marker behavior is deterministic and source-traceable.
- Proven static imports become directed `depends_on` relationships; dynamic or unresolved behavior remains visible without invented local modules.
- The canonical `arch-view.model/v1` contract, stable ordering, diagnostics, confidence, and source evidence are preserved.
- Existing Go behavior, analyzer selection, viewer behavior, layout configuration, and export contracts remain unchanged.
- Adding the Python adapter modifies only the composition-root registration and Python capability code; host orchestration, model, layout, and renderer code do not gain Python-specific switches.

## Verification surfaces

- **Backend boundary:** manifest/detection, project-marker precedence, source-root resolution, parser safety, package/module identity, import classification, diagnostics, options, deterministic output, and cancellation.
- **Frontend integration:** existing viewer opens the Python model, preserves hierarchy/import evidence, and does not need language-specific rendering logic.
- **End-to-end:** Python repository → analysis JSON → canonical model → local viewer, self-contained HTML, and SVG, with repeated output checks.
- **Repository/OKF integrity:** full repository gates, strict OKF validation, synchronized issue references, and the untouched upstream reference boundary.

## Artifact impact

This is a completed roadmap implementation slice, not a product-topology change. The product and architecture boundaries remain the same: Python is an additional implementation of the existing analyzer contract. Issue 017 supplies the registered project/module-discovery foundation; issue 018 supplies relationships, uncertainty, evidence, and partial-result behavior; issue 019 completes the visible journey after automated verification and visual approval. Delivery sequencing, Python capability references, application synthesis status, the issue registry, and `.okf/log.md` are synchronized; no new shared concern or capability node is required.
