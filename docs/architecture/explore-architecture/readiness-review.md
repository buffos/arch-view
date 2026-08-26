# Explore and inspect architecture readiness review

## Findings

No High or Medium architecture findings remain. A product-level visual review identified an implementation mismatch in the first overview: expanding every non-local reference makes a small project graph unreadable, and the initial layout needs a clearer interaction baseline. The exact artifacts now make the local-first reference policy, import-list inspection, scope/confidence distinction, session-owned layout behavior, user-selectable ELK layout settings, and project configuration discovery/persistence explicit.

## Residual risks

- Renderer performance thresholds and layout implementation require benchmark fixtures.
- Reference visibility defaults and import-list behavior require representative small/large graph fixtures.
- Manual position persistence must remain scoped to model revision and hierarchy path and must never leak into canonical model or export semantics.
- The complete pinned-ELK option catalog and unsupported option combinations are implemented with typed validation and explicit UI treatment; issue 007's settings surface has passed human visual approval.
- Project configuration writes now have platform-specific atomic replacement and destination-handling coverage: ordinary `Save` overwrites the exact active discovered file, while only explicit `Save As` may choose a custom folder; nearest-ancestor selection remains deterministic.
- Browser compatibility, theme tokens, and detailed visual styling are implementation-level choices bounded by the scene contract.
- Source-serving policy needs security tests for symlinks and large/unreadable files.

## Readiness

`READY FOR ARCHITECTURE IMPLEMENTATION` for the revised local-first viewer, renderer-neutral contract, and the bounded layout-settings/project-configuration extension.

## Verification and synthesis gate

Frontend integration and end-to-end surfaces are explicitly covered; backend boundary is applicable for model/source endpoints. Root verification policy is current and application documents are synchronized.

## Artifact impact

- Capability truth: exact-spec set refreshed with reference visibility, import inspection, session-owned layout rules, typed ELK settings, and `.archview.json` discovery/persistence.
- Product truth: updated with the layout-settings journey and project-scoped presentation preferences.
- Architecture truth: viewer/local host owns presentation configuration resolution, active-file `Save`, and explicit custom-folder `Save As`; analyzer options, canonical model facts, and export semantics remain separate.
- Delivery truth: issues 003, 004, 005, 006, and 007 are complete after their explicit visual reviews; issue 008 is the next ready implementation slice.
