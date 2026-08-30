# Deterministic quality checks acceptance scenarios

These are contract/backend scenarios. Viewer coloring and layout are
downstream projections; they must consume the same finding subject/status.

## DQC-AC-001 — File-size threshold boundary

**Given** a file with exactly 500 physical lines and a rule configured as
`greater_than 500`

**When** the quality profile is evaluated

**Then** no finding is emitted for equality, and a file with 501 lines emits
one exact finding with the observed metric, limit, operator, severity, and file
evidence.

## DQC-AC-002 — Callable size uses an explicit body span

**Given** a callable whose extractor supplies a body span and a callable whose
extractor does not

**When** `source:callable.max-lines` is evaluated

**Then** the first uses its declared formula/version and the second receives
`not_evaluable` coverage; no nearest-brace estimate is used.

## DQC-AC-003 — Complexity formula is versioned

**Given** two language providers with different decision vocabularies

**When** they emit cyclomatic metrics

**Then** each metric carries its provider, formula ID/version, decision basis,
and value; a rule compares only a compatible metric contract and does not
pretend the language implementations are identical.

## DQC-AC-004 — Unknown data is not a pass

**Given** a requested complexity rule for a language without a complexity
provider

**When** the report is evaluated

**Then** the report contains unsupported/not-evaluable coverage and no pass or
violation is inferred from an empty metric collection.

## DQC-AC-005 — Documentation coverage is honest

**Given** ten public symbols, six with present documentation, two unsupported
documentation subjects, and two unknown visibility subjects

**When** public documentation coverage is calculated

**Then** the observed denominator is the supported, visibility-observed set,
the six documented subjects are counted, and unsupported/unknown counts remain
visible in coverage rather than being classified as missing comments.

## DQC-AC-006 — Cycles use canonical graph facts

**Given** a canonical model with one reported module cycle and a path/name
pattern that happens to repeat without a graph edge

**When** `architecture:no-cycles` is evaluated

**Then** only the canonical cycle produces an exact finding, with participating
module/relationship evidence; path coincidence produces nothing.

## DQC-AC-007 — Layer and forbidden dependency rules need explicit policy

**Given** modules and relationships but no configured layer or forbidden-edge
policy

**When** architecture rules are evaluated

**Then** the rules are not evaluable or disabled by validated profile policy;
the engine does not infer intended architecture from names/directories.

## DQC-AC-008 — Finding identity survives a line shift

**Given** an active finding whose subject stable identity is unchanged while a
preceding comment shifts its evidence line

**When** a second compatible report is compared

**Then** the same finding key matches, while the report-local ID and evidence
span may change.

## DQC-AC-009 — Rule revision is not silently baselined

**Given** a baseline entry for rule version 1 and a report evaluated with rule
version 2 or a changed metric formula

**When** suppression is resolved

**Then** the old entry does not suppress the new finding; an explicit exact
version baseline is required.

## DQC-AC-010 — Suppression preserves detection

**Given** an active finding with an exact matching baseline entry

**When** the next report is evaluated

**Then** the finding remains present with `suppressed`/`baseline` status,
evidence, key, and reason; it is not deleted or reported as resolved.

## DQC-AC-011 — SOLID is a signal, not a verdict

**Given** a type with a configured structural SRP indicator and evidence of
many members/dependency clusters

**When** `signal:solid.srp` runs

**Then** it emits an informational `assessment_kind: signal` result with
limitations and heuristic provenance. It never emits an exact “SRP violation”
or claims responsibility intent was proven.

## DQC-AC-012 — All SOLID principles obey the same boundary

**Given** indicators for SRP, OCP, LSP, ISP, and DIP

**When** the signal catalog is evaluated

**Then** every result remains explicitly labeled signal, including DIP concrete
dependency or LSP hierarchy indicators; compiler/type-checker facts are not
rebranded as proven SOLID violations.

## DQC-AC-013 — Rule registration is open/closed

**Given** a newly registered rule with a new namespaced metric capability

**When** the existing engine evaluates its profile binding

**Then** it validates and invokes the strategy through the registry without a
core switch or modification to existing rules.

## DQC-AC-014 — Partial provider failure is isolated

**Given** one metric provider fails for one scope while another provider and
other scopes produce valid facts

**When** a report is assembled

**Then** affected rules have diagnostics/partial coverage, valid findings remain,
and source/model facts are not discarded.

## DQC-AC-015 — Report determinism

**Given** equal source/model snapshot IDs, profile, baseline, provider/rule
versions, and options

**When** evaluation runs twice

**Then** metrics, findings, coverage, ordering, evaluation fingerprint, and
report digest are equal apart from excluded operational metadata.

## DQC-AC-016 — File line findings support a human filter

**Given** a complete quality report for one scope with a configured
`source:file.max-lines` limit of 500 and three active file findings

**When** a human-facing consumer renders the report

**Then** it shows that three files exceed the configured limit and offers an
explicit filter that lists only those three file subjects from the same
scope/report; it does not recalculate the result from raw file facts.

**And** if the report has partial, unknown, or unsupported coverage, the
consumer labels that coverage instead of presenting an incomplete count as
zero.
