# Arch View planning map

- [Arch View project](project.md) - Analyze supported codebases and generate navigable architecture views.

## Capabilities

- [Analyze source code](capabilities/analyze-source.md) - Discover project structure and dependencies through language analyzers, compiled external plugins, and multi-analyzer project orchestration.
- [Generate architecture models](capabilities/generate-models.md) - Normalize analyzer output into hierarchical, layered architecture models.
- [Explore and inspect architecture](capabilities/explore-architecture.md) - Navigate diagrams, dependency evidence, cycles, and source files.
- [Export and automate](capabilities/export-and-automate.md) - Produce headless artifacts and support repeatable analysis workflows.

## Planning status

- Initial topology: confirmed on 2026-08-25.
- State totals: 0 `foggy`, 0 `bounded`, 4 `specified`, 10 `implemented`.
- Current delivery frontier: Go analysis, canonical model generation, export
  and automation, the current Explore scope, Python, TypeScript, Rust,
  Clojure, and the first external process slice are implemented after their
  repository verification and declared review gates. Issues 034–038 complete
  the compiled entrypoint migration, deterministic distribution assembly,
  trusted package verification, packaged runtime selection, parity, and
  release verification. [Multi-analyzer project orchestration](capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md)
  has completed implementation issues 039–043, including its declared visual
  review. [Project analyzer assignments and view selection](capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md)
  has verified issues 044–046 and one awaiting-human-review issue 047 for its
  declared visual gate. [Advanced ELK renderer support](capabilities/explore-architecture/advanced-elk-renderer-support.md)
  remains ready for later delivery issue slicing.
  The plugin-runtime and Analyze source nodes remain specified as rollups over
  that unfinished child work.

## Application synthesis

- [Application PRD](../docs/prd.md)
- [Application architecture summary](../docs/architecture/application-architecture-summary.md)
