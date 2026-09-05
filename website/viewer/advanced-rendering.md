# Advanced ELK rendering

Architecture and OKF views use the same layout settings, ELK runtime, geometry validation, and SVG rendering pipeline. Their saved settings remain separate because the two scenes represent different kinds of information.

Advanced rendering is opt-in. Existing graphs keep their ordinary appearance until a feature is enabled and saved in the relevant layout or OKF profile.

## Feature registry

The viewer resolves advanced features through a shared registry. Each feature declares its prerequisites, compatible algorithms, owned geometry, and fallback behavior. This keeps feature behavior out of scene-specific switch statements and makes the same settings meaningful in both viewers.

The current feature set is:

| Feature | What it adds | Current support |
| --- | --- | --- |
| Edge labels | ELK-positioned relationship-count labels. | Layered. |
| Spline refinement | Connected cubic curve sections returned by ELK. | Layered with `SPLINES` edge routing. |
| Shared-route junctions | Markers at validated points where routes genuinely share a path. | Layered when the pinned runtime returns junction points. |
| Presentation ports | Deterministic visual input/output ports and validated edge endpoints. | Layered. |
| Nested containers | Presentation-only frames for visible compound hierarchy and cross-boundary routes. | Layered; primarily useful for OKF projections with visible hierarchy. |

The renderer consumes geometry regardless of which basic algorithm produced it. Individual advanced features still depend on the behavior and output supported by the pinned ELK runtime. Other algorithms are not currently advertised as advanced-feature-compatible.

## Example OKF export

This is a real SVG export of the example OKF bundle, captured with ELK Layered, edge labels enabled, and junctions enabled.

<img src="/examples/okf-layered-edge-labels-junctions.svg" alt="Example OKF graph exported from the viewer">

The file is a static export. It shows the rendered graph and its geometry, but it does not include the viewer's selection, navigation, or profile-editing behavior.

## Settings and fallbacks

The layout dialog distinguishes supported options, catalog-only options, algorithm-inapplicable options, and options that require an enabled feature. For example, spline refinement requires `SPLINES` routing.

The feature registry validates dependencies and geometry before rendering. Invalid or incomplete advanced geometry falls back to deterministic ordinary geometry for the affected portion, and the viewer reports a diagnostic without replacing the last valid scene unnecessarily.

Junctions are visual aids, not new relationships. Ports are presentation objects, not concepts or canonical relationships. Containers are presentation frames and do not invent hidden descendants or turn semantic links into hierarchy.

## Export behavior

The live browser, embedded interactive HTML, and browser **Download SVG** share the advanced geometry contract. The deterministic static Go SVG exporter remains orthogonal and reports when advanced features were not applied.

Architecture's normal progressive projection usually exposes one hierarchy level at a time, so enabling nested containers may not visibly change that scene. OKF profiles can expose visible nested hierarchy where containers are useful.

Read [Change the layout](/viewer/layout) for everyday settings and [Layout options](/reference/layout-options) for the complete generated catalog.
