# Compiled external analyzer distribution orchestration status

## State

- Planning state: `specified`.
- The exact-spec pipeline is complete; approved implementation issues 034–038
  now define the dependency-ordered delivery sequence.

## Evidence

- The generic external process host is implemented and verified through the
  existing plugin-runtime slice.
- The current Python deployment is a script-based parity pilot and requires a
  Python interpreter.
- The five compiled analyzer entrypoints and shared child-side process runner
  are implemented and pass process-adapter parity verification. Distribution
  assembly, package indexing, trust verification, and automatic packaged
  runtime selection do not exist yet.

## Artifact sync

- **Topology:** This child is linked under `Analyzer plugin runtime`.
- **Capability:** Discovery notes record the compiled-binary target, stable
  logical identity, application-managed discovery, build assembly, and
  one-job-per-process policy; exact packaging fields remain deferred to the
  specification stage.
- **Product and architecture:** The application synthesis is refreshed with
  the future external-binary boundary.
- **Delivery:** Issue 034 is complete and records the shared runner plus five
  compiled entrypoints. Issues 035–038 remain the approved dependency-ordered
  sequence for packaging, trust, runtime cutover, and release verification.

## Exact-spec inventory

- Requirements gap analysis
- Domain glossary
- Capability PRD
- Canonical domain model
- Canonical use-case model
- Canonical API/CLI contract
- Acceptance scenarios
- Architecture readiness review

All artifacts are linked from the capability node and agree on package index,
platform selection, SHA-256 integrity, trust boundary, build assembly, runtime
modes, and explicit fallback.

## Readiness decision

READY FOR ARCHITECTURE IMPLEMENTATION. This child is now in delivery through
issues 035–038 after issue 034's completed entrypoint migration. The remaining
specified nodes proceed to reference-document issue slicing after their
application synthesis gates are checked.

## Artifact impact

- Capability truth: updated with the complete exact-spec set.
- Product truth: no new actor or workflow; root PRD remains current.
- Architecture truth: package trust, platform, and launch boundaries are current.
- Delivery truth: issue 034 is archived after implementation and verification;
  issues 035–038 remain active in dependency order. The capability remains
  `specified` until package assembly, trust, runtime, and release verification
  complete.
