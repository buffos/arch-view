# Read the graph

The graph is a map of reported relationships between parts of a project.

## What a node means

A node can represent a module, a group of modules, or another visible architecture item.

The label tells you the name. The small badges tell you the language and status. The line count or relationship count gives a quick idea of size and connectivity.

In an OKF view, the node is a generic concept. Its label, compact fields, state, shape, and colors come from the selected profile. A concept is not automatically a capability, module, or other fixed category.

## What an arrow means

An arrow means that the report contains a dependency relationship.

~~~text
web -> orders
~~~

Read this as:

> “The report says that web depends on orders.”

It does not automatically mean:

- the dependency is bad;
- the dependency is used on every request;
- the two modules should be merged;
- the target is local source code.

Look at the relationship details and source evidence before deciding what to change.

## OKF containment and semantic links

OKF scenes separate structural and informational relationships:

- solid containment edges describe explicit or filesystem hierarchy and point toward the child;
- dashed semantic links describe Markdown cross-references and have no arrowheads.

Semantic links are hidden until a concept is selected. Selecting another concept replaces the previous overlay. They never participate in layout, so adding or removing them does not rearrange the hierarchy.

## Layers

Layers are a visual arrangement of modules. They help you see direction.

If the graph places web above orders, that placement is a layout choice. It is not automatically an architecture rule. A layer-direction quality check only has meaning when you configure an explicit layer policy.

## Groups and hierarchy

A group collects related modules. Select the group when you want a summary of the whole group. Select a child node when you want that one module.

The hierarchy path answers “where is this item inside the project map?” It is not necessarily a filesystem path.

## Search and keyboard use

Use the node search to narrow the scene. The scene list remains keyboard accessible:

- Tab moves to the list.
- Arrow keys move between visible items.
- Enter selects the focused item.
- Escape closes an open control when supported.

If a search returns no items, clear the search instead of refreshing the page.

## Containers, ports, and labels

When enabled and supported by the selected layout, advanced rendering may add relationship-count labels, presentation-only ports, and frames around visible nested hierarchy. These objects are part of the drawing, not new concepts or canonical relationships. Read [Advanced ELK rendering](/viewer/advanced-rendering) for the compatibility boundary.

## References

Non-local references are shown according to the selected reference visibility:

- **Hidden**: keep the graph focused on local modules.
- **Aggregated**: show an external dependency as a summarized item.
- **Expanded**: show more individual references.

More visible references can make the graph harder to read. Use them when the boundary matters.
