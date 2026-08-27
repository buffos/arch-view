# Arch View planning map

- [Arch View project](project.md) - Analyze supported codebases and generate navigable architecture views.

## Capabilities

- [Analyze source code](capabilities/analyze-source.md) - Discover project structure and dependencies through language analyzers.
- [Generate architecture models](capabilities/generate-models.md) - Normalize analyzer output into hierarchical, layered architecture models.
- [Explore and inspect architecture](capabilities/explore-architecture.md) - Navigate diagrams, dependency evidence, cycles, and source files.
- [Export and automate](capabilities/export-and-automate.md) - Produce headless artifacts and support repeatable analysis workflows.

## Planning status

- Initial topology: confirmed on 2026-08-25.
- State totals: 1 `foggy`, 0 `bounded`, 8 `specified`, 2 `implemented`.
- Current delivery frontier: the current Explore and inspect architecture scope and the [Python analysis implementation slice](../docs/architecture/analyze-source/python-analysis/implementation-slice.md), issues 017–019, are implemented after their repository verification and visual-review gates. No delivery issue remains active; TypeScript is the next language in the agreed sequence, while the separate foggy [Advanced ELK renderer support](capabilities/explore-architecture/advanced-elk-renderer-support.md) child must be specified before it receives delivery issues.

## Application synthesis

- [Application PRD](../docs/prd.md)
- [Application architecture summary](../docs/architecture/application-architecture-summary.md)
