# OKF bundles

Open Knowledge Format (OKF) bundles are independent, local-first knowledge sources for the viewer. A bundle is made from Markdown concept documents with frontmatter and can be explored without turning its concepts into architecture-model modules.

## A document becomes a concept

Each indexed document retains its:

- stable concept identity;
- source-relative path;
- title, description, and user-defined type;
- frontmatter and Markdown content;
- explicit hierarchy declarations;
- Markdown links;
- source provenance and revision information.

The `type` value is vocabulary supplied by the bundle. It is not interpreted as a fixed Arch View category. Presentation profiles choose which values matter to the graph.

## Relationships

The index keeps relationship kinds separate:

| Relationship | Meaning in the viewer |
| --- | --- |
| Containment | Explicit hierarchy or configured filesystem hierarchy. It participates in layout and points toward the child. |
| Semantic link | An informational Markdown link between concepts. It does not participate in layout and has no arrowhead. |

The hierarchy rules have a defined precedence. When explicit hierarchy is absent or incomplete, a profile may allow the filesystem to provide a fallback. Conflicts are retained as diagnostics instead of being silently guessed away.

Semantic links are intentionally not treated as containment. This keeps the main arrangement readable even when the documents contain many cross-references.

## Discovery and validation

The local server discovers independent bundles inside the project and reports a catalog. A bundle is selectable only when its source can be read, validated, indexed, and kept inside its own boundary.

The catalog and summary can report:

- valid, invalid, unavailable, or unreadable status;
- concept and relationship counts;
- source revision;
- affected files and diagnostics.

One invalid bundle does not make an independent valid bundle unusable. Refreshing the catalog re-reads the source while preserving the source documents themselves.

## Markdown safety

Concept details use sanitized CommonMark. Unsafe local paths and unsafe URL schemes are not made clickable. Safe local concept links stay inside the selected bundle, while safe external links are clearly external.

Read [Concept details](/okf/exploration#concept-details) for the inspection experience and [the OKF API reference](/reference/okf-api) for machine-readable responses.
