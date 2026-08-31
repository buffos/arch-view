# MCP safety and freshness

Arch View is a read-only code intelligence service by default.

## What the server can do

- read analyzer-neutral models and bounded source facts;
- search paths, symbols, documentation, and exact text;
- show quality rules, findings, evidence, and comparisons;
- wait for a verified revision after source changes.

## What it cannot do through MCP

MCP v1 does not run shell commands, execute the target program, install
packages, edit source files, or choose a path outside the opened repository.
The coding assistant may use its ordinary tools for edits, then ask Arch View
to verify the new revision.

## Freshness

Filesystem watcher events are hints. They may be duplicated or missed. A
`latest_ready` query returns the last published revision and labels it stale or
updating when needed. `ensure_current_snapshot` performs an authoritative
reconciliation, joins one rebuild, and checks the input again after analysis.
If edits continue during the bounded wait, the result says `input_unstable`.
The previous ready revision remains available during a failed rebuild.

## Policy writes

Profile and baseline writes are separate operations. They are disabled unless
the session was explicitly started with policy-write permission and the client
provided the configured authorization. Baselines contain selected finding keys
and exact version identities; creating one never means “hide every finding.”

Preview a baseline first. Then create it only after a human or agent has
recorded why each finding is accepted.
