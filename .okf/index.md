# Arch View planning map

- [Arch View project](project.md) - Analyze supported codebases and generate navigable architecture views.

## Capabilities

- [Analyze source code](capabilities/analyze-source.md) - Discover project structure and dependencies through language analyzers, compiled external plugins, and multi-analyzer project orchestration.
- [Generate architecture models](capabilities/generate-models.md) - Normalize analyzer output into hierarchical, layered architecture models.
- [Explore and inspect architecture](capabilities/explore-architecture.md) - Navigate diagrams, dependency evidence, cycles, and source files.
- [Export and automate](capabilities/export-and-automate.md) - Produce headless artifacts and support repeatable analysis workflows.
- [Code quality and code intelligence](capabilities/code-quality-and-intelligence.md) - Build deterministic quality findings and compact, searchable source intelligence.

## Planning status

- Initial topology: confirmed on 2026-08-25.
- State totals: 0 `foggy`, 0 `bounded`, 5 `specified`, 13 `implemented`.
- Current delivery frontier: Go analysis, canonical model generation, export
  and automation, the current Explore scope, Python, TypeScript, Rust,
  Clojure, and the first external process slice are implemented after their
  repository verification and declared review gates. Issues 034–038 complete
  the compiled entrypoint migration, deterministic distribution assembly,
  trusted package verification, packaged runtime selection, parity, and
  release verification. [Multi-analyzer project orchestration](capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md)
  has completed implementation issues 039–043, including its declared visual
  review. [Project analyzer assignments and view selection](capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md)
  has completed issues 044–047, including its declared visual gate. [Advanced ELK renderer support](capabilities/explore-architecture/advanced-elk-renderer-support.md)
  remains specified and is ready for later delivery issue slicing. The
  plugin-runtime and Analyze source nodes are implemented own-state capabilities
  whose child implementation is complete; their own PRDs and scopes remain
  authoritative. The new [Code quality and code intelligence](capabilities/code-quality-and-intelligence.md)
  capability is a pure structural-child roll-up with effective state
`specified`, the minimum of its three children. All three code-quality child
  capabilities are specified and readiness-reviewed; the source-facts child has
  delivered issues 048–051 and has issue 052 implemented behind its final
  visual-review gate. Its capability state remains specified until that gate is
  approved; the remaining children are unchanged.

## Application synthesis

- [Application PRD](../docs/prd.md)
- [Application architecture summary](../docs/architecture/application-architecture-summary.md)
