# Issues Registry

| # | Title | Category | Owning Capability | Artifact Root | Issue File | State | Blocked by |
|---|---|---|---|---|---|---|---|
| 036 | Load and verify trusted analyzer packages | feature | `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md` | `docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/` | `docs/agents/issues/pending/20260828-036-trusted-analyzer-package-verification.md` | ready-for-agent | — |
| 037 | Make packaged runtime selection the default | feature | `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md` | `docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/` | `docs/agents/issues/pending/20260828-037-packaged-runtime-selection.md` | ready-for-agent | 036 |
| 038 | Verify compiled analyzer parity and release behavior | feature | `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md` | `docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/` | `docs/agents/issues/pending/20260828-038-compiled-analyzer-parity-and-release-verification.md` | ready-for-agent | 037 |

Issues 034–038 are the dependency-ordered compiled-distribution delivery
slice. Issue 034 ports all existing analyzer implementations to compiled
plugin entrypoints, issue 035 assembles the deterministic packages and release
tree, and issues 036–038 own verification, runtime selection, and release
parity closure. Issues 030–033 remain archived delivery evidence for the
external protocol pilot.

# Current Max Issue ID

038
