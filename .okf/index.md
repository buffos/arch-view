# Arch View planning map

- [Arch View project](project.md) - Analyze supported codebases and generate navigable architecture views.

## Capabilities

- [Analyze source code](capabilities/analyze-source.md) - Discover project structure and dependencies through language analyzers, compiled external plugins, and multi-analyzer project orchestration.
- [Generate architecture models](capabilities/generate-models.md) - Normalize analyzer output into hierarchical, layered architecture models.
- [Explore and inspect architecture](capabilities/explore-architecture.md) - Navigate diagrams, dependency evidence, cycles, and source files.
- [Export and automate](capabilities/export-and-automate.md) - Produce headless artifacts and support repeatable analysis workflows.

## Planning status

- Initial topology: confirmed on 2026-08-25.
- State totals: 0 `foggy`, 0 `bounded`, 6 `specified`, 8 `implemented`.
- Current delivery frontier: Go analysis, canonical model generation, export
  and automation, the current Explore scope, Python, TypeScript, Rust,
  Clojure, and the first external process slice are implemented after their
  repository verification and declared review gates. Issue 034 completes the
  compiled entrypoint migration; issue 035 completes deterministic distribution
  assembly, and the compiled external analyzer distribution remains in active
  delivery through issues 036–038. Three
  leaf capabilities remain ready for delivery issue slicing: [Advanced ELK renderer support](capabilities/explore-architecture/advanced-elk-renderer-support.md),
  [Multi-analyzer project orchestration](capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md),
  and [Project analyzer assignments and view selection](capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md).
  The plugin-runtime and Analyze source nodes remain specified as rollups over
  that unfinished child work.

## Application synthesis

- [Application PRD](../docs/prd.md)
- [Application architecture summary](../docs/architecture/application-architecture-summary.md)
