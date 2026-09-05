# OKF presentation profiles

An OKF profile is the presentation contract between a generic knowledge bundle and the viewer. It decides what a concept looks like without changing the source document.

## Built-in profiles

The built-ins provide useful starting points:

| Profile | Presentation |
| --- | --- |
| Neutral | Name-focused, quiet styling with no state-specific assumptions. |
| Fog of War | Profile-provided state colors, state text, root styling, and roll-up borders. |

Fog of War is not a special capability mode. It is an example of a profile mapping the bundle's vocabulary into visual tokens. If a bundle uses different state names, its profile can map those names explicitly.

## What a profile can map

A project profile can configure:

- one or more node fields for labels and compact metadata;
- user-defined type or frontmatter fields;
- state sources and state-to-token mappings;
- containment and semantic-link visibility;
- explicit hierarchy and filesystem fallback;
- navigation depth and graph safety budgets;
- default, state, root, and roll-up style tokens;
- detail visibility and registered detail renderers;
- layout algorithm, options, and advanced renderer features;
- ordered rules and their parameters.

The default graph remains compact. Additional metadata belongs in the concept details pane rather than being forced into every node.

## Composition

Project profiles may compose built-in or other project profiles. Composition is ordered and validated. Later settings only affect the fields they explicitly provide, and conflicting rules or invalid base references produce diagnostics.

This lets a project keep a general profile while adding a small vocabulary-specific layer for one bundle.

## Edit a profile

Choose **Edit profile** in the OKF view. The normal editor is a form with repeatable fields and Add/Remove controls. It covers common presentation settings, hierarchy, state mapping, style tokens, navigation, details, rules, and layout.

The **Advanced JSON** section is available for extensions and uncommon settings. Valid JSON rehydrates the form, and unknown top-level profile fields are preserved when the profile is copied or saved.

Built-in profiles are immutable. Use **Save As** to create a project-local copy before changing one. Project profiles support **Save**, **Save As**, **Bind bundle**, **Rename**, and **Delete**, subject to revision and idempotency checks.

Read [OKF configuration](/okf/configuration) for persistence details.
