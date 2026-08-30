# Settings reference

This page is the map of settings a person can change or configure.

## Viewer settings

The viewer exposes:

- analysis scope;
- reference visibility;
- layout algorithm;
- layout options;
- quality profile;
- quality finding filter;
- source and symbol filters;
- bounded page size and load-more controls.

Each control changes the presentation or the selected data. It does not rewrite the project.

## Quality settings

Quality settings live in the profile JSON or in the viewer's profile editor:

- enabled state for each rule;
- rule severity;
- threshold operator and limit;
- SOLID signal thresholds;
- coupling external-reference policy;
- forbidden dependency selectors;
- layer direction selectors;
- baseline reference.

Read [Quality profiles](/quality/profiles) and [Quality profile JSON](/formats/quality-profile).

## Analyzer settings

Analyzer settings select the project files and language interpretation:

- source roots;
- modules, crates, targets, and build tags;
- runtime and platform;
- tests, examples, stubs, generated files, and JavaScript;
- exclusions;
- safe mode;
- external reference detail.

Read the language pages under [Analyzers](/analyzers/overview).

## Layout settings

Layout settings change only the drawing. They are grouped into:

- direction and edge routing;
- spacing;
- layering;
- crossing minimization;
- node placement;
- compaction;
- randomness;
- algorithm-specific options.

Read [Change the layout](/viewer/layout) for the useful options and [Layout options](/reference/layout-options) for the complete catalog.
