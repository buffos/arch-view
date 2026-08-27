# Arch View planning map

- [Arch View project](project.md) - Analyze supported codebases and generate navigable architecture views.

## Capabilities

- [Analyze source code](capabilities/analyze-source.md) - Discover project structure and dependencies through language analyzers.
- [Generate architecture models](capabilities/generate-models.md) - Normalize analyzer output into hierarchical, layered architecture models.
- [Explore and inspect architecture](capabilities/explore-architecture.md) - Navigate diagrams, dependency evidence, cycles, and source files.
- [Export and automate](capabilities/export-and-automate.md) - Produce headless artifacts and support repeatable analysis workflows.

## Planning status

- Initial topology: confirmed on 2026-08-25.
- State totals: 1 `foggy`, 0 `bounded`, 7 `specified`, 3 `implemented`.
- Current delivery frontier: the current Explore and inspect architecture scope and the [Python analysis implementation slice](../docs/architecture/analyze-source/python-analysis/implementation-slice.md), issues 017–019, are implemented after their repository verification and visual-review gates. The user-selected [Clojure analysis implementation slice](../docs/architecture/analyze-source/clojure-compatibility/implementation-slice.md), issues 026–029, is implemented and verified; TypeScript, Rust, the external plugin protocol, and the separate foggy [Advanced ELK renderer support](capabilities/explore-architecture/advanced-elk-renderer-support.md) child remain later frontiers.

## Application synthesis

- [Application PRD](../docs/prd.md)
- [Application architecture summary](../docs/architecture/application-architecture-summary.md)
