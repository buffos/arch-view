# Explore an OKF view

The OKF graph uses the same viewport and interaction behavior as the architecture viewer, while its concepts and relationships come from the selected bundle and profile.

## Navigate the projection

- **Depth** limits how far containment is shown from the current focus.
- **Full** requests the full permitted containment scope, subject to safety budgets.
- Double-click a concept to focus its subtree and run a new projection and layout.
- **Back** returns to the previous focus.
- Breadcrumbs return to an earlier focus or the top level.
- Hidden concepts remain available through the bounded hidden-concepts list when the profile or safety limits exclude them from the current graph.

Ordinary selection only changes the selected concept and details pane. It does not run ELK again or move the graph.

## Read the lines

Solid lines are directed containment relationships. Their arrowhead is on the child side of the relationship.

Dashed lines are semantic links. They are informational overlays:

- they are hidden until a concept is selected;
- selecting another concept replaces the previous concept's semantic links;
- they never influence ELK placement;
- they have no arrowheads.

This keeps the hierarchy readable while still allowing a selected concept's cross-references to be inspected.

## Use the viewport

- Drag the background to pan.
- Use the mouse wheel or `+` and `−` controls to zoom.
- Hold **Shift** while dragging a node to move it manually.
- Use **Fit** to center the entire current geometry with a small margin. Fit may enlarge a small graph beyond 100% or shrink a large graph.
- **Full canvas** provides the expanded canvas view, with the same Fit behavior.
- **Reset layout** removes manual positions and returns to calculated geometry.
- **Download SVG** exports the current browser geometry, including supported labels, curves, ports, junction markers, and containers.

Manual positions are scoped to the bundle, source revision, profile, focus, depth, and Full state so one exploration does not corrupt another.

See the [Layered OKF SVG example](/viewer/advanced-rendering#example-okf-export) for a static export from the viewer.

## Concept details

Selecting a concept shows the current profile's detail presentation, including available metadata, source path, provenance, state, relationships, and rendered Markdown. The default detail presentation stays compact, while profiles may select registered detail renderers.

Markdown is rendered as sanitized CommonMark. Safe local concept links select another concept inside the bundle. Safe external links remain external. Unsafe links are shown as unavailable rather than executed.

Read [OKF profiles](/okf/profiles) to change what appears in nodes and details.
