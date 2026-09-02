# Arch View planning map

- [Arch View project](project.md) - Analyze supported codebases and generate navigable architecture views.

## Capabilities

- [Analyze source code](capabilities/analyze-source.md) - Discover project structure and dependencies through language analyzers, compiled external plugins, and multi-analyzer project orchestration.
- [Generate architecture models](capabilities/generate-models.md) - Normalize analyzer output into hierarchical, layered architecture models.
- [Explore and inspect architecture](capabilities/explore-architecture.md) - Navigate diagrams, dependency evidence, cycles, and source files.
- [Configurable OKF knowledge views](capabilities/okf-knowledge-views.md) - Explore arbitrary OKF bundles through configurable projections and ELK-backed interactive views.
- [Export and automate](capabilities/export-and-automate.md) - Produce headless artifacts and support repeatable analysis workflows.
- [Code quality and code intelligence](capabilities/code-quality-and-intelligence.md) - Build deterministic quality findings and compact, searchable source intelligence.

## Planning status

- Initial topology: confirmed on 2026-08-25.
- State totals: 0 `foggy`, 1 `bounded`, 1 `specified`, 17 `implemented`.
- The [Configurable OKF knowledge views](capabilities/okf-knowledge-views.md)
  capability is a first-class root-level capability with a bounded discovery
  baseline. Its pre-PRD notes are maintained under
  `../docs/architecture/okf-knowledge-views/discovery-notes.md`.
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
`implemented`, the minimum of its three children. All three code-quality child
  capabilities are readiness-reviewed; the source-facts child has delivered
  issues 048–052 and is implemented after its approved final visual-review
  gate. Deterministic quality has delivered and archived issues 053–063 and
  077–078 and is implemented after its desktop and responsive visual review.
  Live-analysis/MCP issues 064–079 are delivered, verified, and product-
  approved. The parent roll-up is now implemented.

## Application synthesis

- [Application PRD](../docs/prd.md)
- [Application architecture summary](../docs/architecture/application-architecture-summary.md)
