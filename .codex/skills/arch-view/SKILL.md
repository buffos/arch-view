---
name: arch-view
description: This skill gives an agent a repeatable way to inspect any supported source repository with Arch View, retrieve bounded quality evidence, and manage reviewed baselines. Use when an agent needs current modules, symbols, documentation gaps, line locations, complexity, coupling, cycles, architecture checks, SOLID signals, quality profiles, or baselines.
---

# Arch View agent workflow

Use Arch View as an evidence service before reading large amounts of source.
It supports all registered analyzers, not only Go. It reports facts; it does
not edit source files or decide whether a design choice is wrong.

This skill is a compact router. It keeps the shared decisions here and puts
the private JSON shapes and transport procedures in focused references. Read
only the references needed for the current task:

- [Quality profile](references/quality-profile.md) before selecting or editing
  a profile.
- [Quality report](references/quality-report.md) before interpreting findings
  or creating a baseline.
- [MCP workflow](references/mcp.md) when an Arch View MCP server is available.
- [CLI workflow](references/cli.md) when MCP is unavailable.

## Short request

The skill contains the transport-specific workflow. A short request is enough:

```text
Use the Arch View skill to perform a complete quality review of this repository. Use MCP when it is available, otherwise use the Arch View CLI. Temporarily use a 600-line file threshold, inspect all SOLID signals, fix real findings, append only reviewed accepted findings to the canonical baseline, re-scan after each fix, and report changes and remaining findings.
```

The skill selects the MCP or CLI steps, keeps the other profile rules intact,
and explains how to inspect evidence before changing source.

## Start

1. Establish the target repository and whether an Arch View MCP server is
   available.
2. Read [MCP workflow](references/mcp.md) for MCP, or [CLI workflow](references/cli.md)
   for terminal-only work.
3. Read [Quality profile](references/quality-profile.md) before using the
   `profile:full` profile or changing a threshold.
4. Read [Quality report](references/quality-report.md) before interpreting the
   result.

The profile is not public knowledge inside the executable. Discover it through
the MCP catalog or a JSON file, then use the reference to preserve its shape.

## Shared review loop

1. Use all relevant analyzers unless the user selects a narrower scope.
2. Get a current analysis. MCP uses `ensure_current_snapshot` with
   `require_current`; CLI runs a new `analyze` command.
3. Select an exact profile ID and version. Read the full profile shape before
   making a temporary copy.
4. Evaluate without persisting a profile.
5. Read coverage and diagnostics before treating findings as meaningful.
6. Retrieve all requested file-size and SOLID findings. MCP follows cursors;
   CLI filters the complete JSON array locally.
7. Read evidence and a bounded source context only for findings under review.
8. Fix real source problems with normal editing tools.
9. Re-scan after every fix.
10. For a complete review, temporarily disable baseline suppression. MCP uses
    `baseline_mode: "none"`; CLI uses `--no-quality-baseline`.
11. Baseline only current, observed, reviewed, intentionally accepted
    findings. MCP reads existing baselines with `get_quality_baselines` and
    appends through `append_baseline`; CLI uses `quality baseline add` with
    the newest clean report. Never baseline unsupported, not-evaluable,
    partial, stale, failed, resolved, or already suppressed findings.
12. Re-run the normal profile evaluation after appending. MCP uses
    `baseline_mode: "profile"`; CLI omits both baseline override flags.
    Confirm accepted findings are suppressed and changed or new findings stay
    visible.
13. Finish with changed files, remaining findings, accepted baselines, and
    unsupported or not-evaluable coverage.

For a saved profile with a baseline reference, the normal evaluation loads
that exact baseline automatically. Use the clean/no-baseline evaluation for
the review itself so existing suppressions cannot hide findings. Append
accepted findings through the managed baseline workflow, then run the normal
evaluation again. A missing baseline warns without suppressing; an invalid or
ambiguous baseline is an error.

## Useful checks

Use these rule IDs when filtering quality findings:

```text
source:public-symbol.documentation
source:file.max-lines
source:callable.max-lines
source:callable.max-cyclomatic-complexity
source:callable.max-nesting-depth
architecture:no-cycles
architecture:forbidden-dependency
architecture:layer-direction
architecture:module.max-afferent-coupling
architecture:module.max-efferent-coupling
signal:solid.srp, signal:solid.ocp, signal:solid.lsp, signal:solid.isp, signal:solid.dip
```

Use `find_symbols` for declarations, `find_text` for exact text, and
`get_source_context` only after an indexed entity or evidence span is known.
Line numbers come from evidence/source spans; source excerpts are opt-in.
SOLID findings are advisory review signals, never proof of a violation. The
transport references contain the exact request fields and baseline commands.
