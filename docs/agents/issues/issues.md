# Issues Registry

| # | Title | Category | Owning Capability | Artifact Root | Issue File | State | Blocked by |
|---|---|---|---|---|---|---|---|
| 064 | Configure and start a live session | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/done/20260831-064-live-session-config-and-initial-snapshot.md | done | — |
| 065 | Normalize and coalesce watcher events | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/done/20260831-065-watcher-events-and-coalescing.md | done | — |
| 066 | Publish coherent immutable revisions | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/done/20260831-066-coherent-revision-store-and-publication.md | done | — |
| 067 | Reconcile freshness and coalesce rebuilds | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/done/20260831-067-freshness-reconciliation-and-single-flight.md | done | — |
| 068 | Expose analyzer-neutral structural queries | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/done/20260831-068-analyzer-neutral-query-surface.md | done | — |
| 069 | Add exact text search and bounded source context | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/done/20260831-069-exact-text-and-source-context.md | done | — |
| 070 | Delegate quality queries and temporary evaluation | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/done/20260831-070-quality-gateway-and-temporary-evaluation.md | done | — |
| 071 | Add permissioned quality-policy operations | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/pending/20260831-071-permissioned-quality-policy-operations.md | ready-for-agent | — |
| 072 | Add the local live-session CLI bridge | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/pending/20260831-072-local-live-session-cli-bridge.md | ready-for-agent | — |
| 073 | Integrate live sessions with the viewer | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/pending/20260831-073-live-viewer-integration.md | ready-for-agent | 072 |
| 074 | Ship the MCP stdio server and installation documentation | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/pending/20260831-074-mcp-stdio-server-and-documentation.md | ready-for-agent | 071, 072 |
| 075 | Add authenticated HTTP MCP transport | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/pending/20260831-075-authenticated-http-mcp-transport.md | ready-for-agent | 074 |
| 076 | Verify the cross-analyzer agent workflow and final product behavior | feature | Live analysis and MCP | docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp | docs/agents/issues/pending/20260831-076-cross-analyzer-agent-workflow-and-final-review.md | ready-for-agent | 073, 074, 075 |
Issues 044–063 are verified and archived, including their declared visual
review gates, source-index backend/query slices, and the deterministic quality
report lifecycle, SOLID signals, bounded query boundary, and headless/export
projections. Issue 063's desktop and responsive visual review is complete.
Product and architecture semantics remain aligned with the readiness-reviewed
reference set. Advanced ELK renderer support remains a separate specified
frontier.

# Current Max Issue ID

076
