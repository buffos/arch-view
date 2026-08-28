# Issues Registry

| # | Title | Category | Owning Capability | Artifact Root | Issue File | State | Blocked by |
|---|---|---|---|---|---|---|---|
| 035 | Assemble the compiled analyzer distribution | feature | `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md` | `docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/` | `docs/agents/issues/pending/20260828-035-compiled-analyzer-distribution-assembly.md` | ready-for-agent | — |
| 036 | Load and verify trusted analyzer packages | feature | `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md` | `docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/` | `docs/agents/issues/pending/20260828-036-trusted-analyzer-package-verification.md` | ready-for-agent | 035 |
| 037 | Make packaged runtime selection the default | feature | `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md` | `docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/` | `docs/agents/issues/pending/20260828-037-packaged-runtime-selection.md` | ready-for-agent | 036 |
| 038 | Verify compiled analyzer parity and release behavior | feature | `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md` | `docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/` | `docs/agents/issues/pending/20260828-038-compiled-analyzer-parity-and-release-verification.md` | ready-for-agent | 037 |

Issues 034–038 are the dependency-ordered compiled-distribution delivery
slice. Issue 034 ports all existing analyzer implementations to compiled
plugin entrypoints before packaging, verification, runtime selection, and
release parity work. Issues 030–033 remain archived delivery evidence for the
external protocol pilot.

# Current Max Issue ID

038
