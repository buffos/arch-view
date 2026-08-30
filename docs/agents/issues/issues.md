# Issues Registry

| # | Title | Category | Owning Capability | Artifact Root | Issue File | State | Blocked by |
|---|---|---|---|---|---|---|---|
| 059 | Add deterministic report lifecycle, finding identity, and baselines | feature | /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md | docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/ | docs/agents/issues/pending/20260830-059-deterministic-report-lifecycle-and-baselines.md | ready-for-agent | — |
| 060 | Add conservative SOLID structural signals | feature | /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md | docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/ | docs/agents/issues/pending/20260830-060-solid-structural-signals.md | ready-for-agent | 059 |
| 061 | Expose bounded quality report, findings, and evidence queries | feature | /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md | docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/ | docs/agents/issues/pending/20260830-061-quality-report-query-and-evidence.md | ready-for-agent | 060 |
| 062 | Add headless quality-report and export projections | feature | /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md | docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/ | docs/agents/issues/pending/20260830-062-quality-cli-and-export-projections.md | ready-for-agent | 061 |
| 063 | Add quality findings to the viewer and affected-file filter | feature | /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md | docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/ | docs/agents/issues/pending/20260830-063-quality-viewer-and-affected-file-filter.md | ready-for-agent | 061 |

Issues 044–058 are verified and archived, including their declared visual
review gates, source-index backend/query slices, and the initial deterministic
quality segment. Deterministic quality issues 059–063 remain active in
dependency order; issue 059 is now unblocked. Product and architecture
semantics remain aligned with the readiness-reviewed reference set. Advanced
ELK renderer support remains a separate specified frontier.

# Current Max Issue ID

063
