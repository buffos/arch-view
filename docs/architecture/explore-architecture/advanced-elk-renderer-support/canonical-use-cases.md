# Advanced ELK renderer support canonical use cases

The approved [delivery contract](delivery-contract.md) updates this baseline
for shared architecture/OKF delivery, persistence, stage ordering, and browser-only
ELK execution. Its AER-R rules and scenario mapping are authoritative.

## Application services

### RendererFeatureService

- `ListSupportedRendererFeatures`
- `ValidateFeatureProfile`
- `NegotiateRendererFeatures`

### GeometryLayoutService

- `BuildFeatureAwareLayoutInput`
- `NormalizeELKGeometry`
- `ApplyGeometryFallback`

### RendererParityService

- `RenderLiveBrowserGeometry`
- `RenderEmbeddedHTMLGeometry`
- `RenderBrowserSVGGeometry`
- `RenderDeterministicStaticSVG`

## Canonical commands and queries

### `ListSupportedRendererFeatures` — query

**Input:** pinned ELK adapter and target renderer.

**Output:** the five admitted features, supported combinations, required
catalog options, and surface support. Catalog-only options are clearly marked.

### `ValidateFeatureProfile` — command

**Input:** layout algorithm, options, features.

**Rules:** reject unknown features, incompatible algorithm/feature combinations,
and options that are not supported for the selected feature/target. An empty
feature list is always valid when the underlying layout profile is valid.

### `BuildFeatureAwareLayoutInput` — command

**Input:** renderer-neutral semantic scene and validated feature profile.

**Responsibilities:** add only presentation labels/ports/compound structure;
preserve semantic IDs; never add inferred relationships.

### `NormalizeELKGeometry` — command

**Input:** pinned ELK output and semantic scene.

**Responsibilities:** validate bounds, references, sections, finite points,
cubic controls, labels, ports, junction incidence, and compound hierarchy.

**Outcome:** `GeometrySnapshot` or a valid degraded snapshot with diagnostics
and deterministic orthogonal fallback for invalid feature portions.

### `RenderLiveBrowserGeometry` / `RenderEmbeddedHTMLGeometry` /
`RenderBrowserSVGGeometry` — commands

All three consume the same validated geometry snapshot and preserve the same
semantic IDs, labels, evidence, accessibility descriptions, and route meaning.

### `RenderDeterministicStaticSVG` — command

Uses the existing deterministic orthogonal exporter. Advanced browser-only
features are not applied; the output records that limitation in provenance and
retains valid semantic labels/accessibility.

## Failure model

- `RendererFeatureUnknown`
- `RendererFeatureUnsupported`
- `GeometryReferenceInvalid`
- `GeometryNonFinite`
- `GeometryDisconnected`
- `GeometryHierarchyInvalid`
- `GeometryRouteInvalid`
- `GeometryFallbackApplied`

Feature failures do not change analyzer/model status when a valid fallback scene
is rendered.

## Architecture-neutral mapping

The layout adapter may be local, embedded, or remote in a later architecture;
the stable boundary is feature negotiation, geometry validation, semantic
identity preservation, and surface-specific fallback.
