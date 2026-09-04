# Issues Registry

| # | Title | Category | Owning Capability | Artifact Root | Issue File | State | Blocked by |
|---|---|---|---|---|---|---|---|
| 077 | Implement the managed quality-baseline store and merge lifecycle | feature | Deterministic quality checks | docs/architecture/code-quality-and-intelligence/deterministic-quality-checks | docs/agents/issues/done/20260831-077-managed-quality-baseline-store-and-merge.md | done | — |
| 078 | Add automatic baseline loading and managed CLI append | feature | Deterministic quality checks | docs/architecture/code-quality-and-intelligence/deterministic-quality-checks | docs/agents/issues/done/20260831-078-managed-quality-baseline-cli.md | done | 077 |
| 079 | Expose managed baselines through live analysis and MCP | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/done/20260831-079-live-mcp-baseline-read-selection-and-append.md | done | 077 |
Issues 044–079 are verified and archived, including their declared visual
review gates, source-index backend/query slices, deterministic quality checks,
managed baseline lifecycle, and live-analysis/MCP session, CLI, viewer, stdio,
and HTTP transport slices. Issue 076's final cross-analyzer and human product
approval is recorded in its archived issue file. Product and architecture
semantics remain aligned with the readiness-reviewed reference set. Advanced
ELK renderer support remains a separate specified frontier.

The Configurable OKF knowledge-view delivery records are also verified and
archived under the dated `20260903-064` through `20260903-071` paths after
explicit user approval on 2026-09-04. Their numeric labels overlap the earlier
parallel Live analysis batch; the dated paths and owning capability references
are authoritative for the OKF records.

# Current Max Issue ID

079
