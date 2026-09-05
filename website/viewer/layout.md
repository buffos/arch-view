# Change the layout

Layout changes how the graph is drawn. They do not change the analysis result.

## Start with the default

Both viewers default to **Layered**. The built-in OKF Neutral and Fog of War profiles also enable **Junctions** and **Presentation ports** by default, so containment and port-aware routing are visible as soon as those profiles open. Both use the same layout dialog and option catalog, but their saved settings remain separate.

Use **Fit** after a large change. Use **Reset layout** when an experiment made the graph harder to read.

Fit centers the complete current geometry with a small margin. It can enlarge a small graph beyond 100% or shrink a large graph; the behavior is the same in normal and Full canvas mode.

## The main algorithms

| Algorithm | Simple explanation | Good first use |
| --- | --- | --- |
| Fixed | Keep positions that are already present. | A hand-arranged diagram. |
| Box | Pack separate boxes. | A graph with no useful connecting edges. |
| Random | Spread nodes randomly. | A quick comparison or demonstration. |
| Layered | Put related nodes into layers. | Most dependency graphs. |
| Tree | Arrange a tree from a root. | A mostly tree-shaped dependency view. |
| Stress | Try to keep connected nodes close. | A dense graph where distance matters. |
| Radial | Put a central item in the middle. | Exploring one important root. |
| Force | Simulate attraction and repulsion. | Exploring a connected network. |
| Spore overlap | Reduce overlap using a compact arrangement. | Specialized dense diagrams. |
| Spore compaction | Compact a structured arrangement. | Specialized dense diagrams. |
| Rectangle packing | Pack rectangular regions. | Many independent regions. |

The names of the algorithms are technical. The important question is simpler: “Which arrangement makes this report easiest for me to read?”

## Editable versus catalog-only settings

The viewer marks a setting as:

- **Editable**: you can change it in this viewer.
- **Catalog only**: the pinned layout engine knows the setting, but this viewer does not safely expose it.
- **Not applicable**: the setting does not apply to the selected algorithm or target.

Catalog-only is not a hidden promise that the control should work. It is an explicit boundary.

## Useful settings

### Direction

Choose the main reading direction:

- RIGHT: flow from left to right.
- LEFT: flow from right to left.
- DOWN: flow from top to bottom.
- UP: flow from bottom to top.

Example: choose DOWN when you want a top-to-bottom architecture diagram.

### Edge routing

Choose how arrows travel:

- NONE: no special routing;
- POLYLINE: connected straight segments;
- ORTHOGONAL: horizontal and vertical segments;
- SPLINES: smooth curves.

Use orthogonal routing for block diagrams. Use splines when a dense graph benefits from smoother lines.

### Node spacing

Node spacing controls the minimum space between nearby nodes. A larger value can improve readability but creates a bigger diagram.

### Edge and node spacing

These values control how much room the layout leaves between edges, nodes, labels, and layers. Start with the defaults. Change one value at a time.

### Layering strategy

This setting changes how the layered algorithm chooses its layers. It can change the shape of the graph without changing the dependencies.

### Crossing minimization

This setting asks the layout to reduce lines crossing one another. It can improve readability and can also take more time on a large graph.

### Thoroughness

Higher thoroughness lets the layered algorithm spend more effort searching for a readable result. It may take longer.

### Random seed

The random seed controls repeatability for algorithms that use randomness. Keeping the same seed helps you compare two layouts.

## Advanced renderer features

The layout dialog also exposes the shared advanced feature registry. The current features are edge labels, spline refinement, shared-route junctions, presentation ports, and nested containers.

These features are currently verified for ELK Layered only. Spline refinement additionally requires `SPLINES` edge routing. Mr. Tree and the other algorithms remain useful for ordinary layout, but the viewer does not claim that they support these advanced geometry features.

Features are opt-in and saved with the relevant presentation profile. They do not change the model. Read [Advanced ELK rendering](/viewer/advanced-rendering) for the feature behavior, fallback rules, and export differences.

## Complete option list

The [layout option reference](/reference/layout-options) contains every option known by the pinned layout catalog, including its type, target, algorithms, default, and availability. Options that are not editable are clearly marked.
