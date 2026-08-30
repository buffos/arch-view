# Compiled external analyzer distribution orchestration status

## State

- Planning state: `implemented`.
- The exact-spec pipeline and dependency-ordered implementation issues 034–038
  are complete.

## Evidence

- The generic external process host is implemented and verified through the
  existing plugin-runtime slice.
- The current Python deployment remains a script-based parity pilot and
  requires a Python interpreter; the compiled packages are the production
  distribution path.
- The five compiled analyzer entrypoints and shared child-side process runner
  are implemented and pass process-adapter parity verification. Issue 035 now
  implements deterministic distribution assembly, descriptor/index generation,
  exact platform metadata, SHA-256 digests, and atomic analyzer/release build
  targets. Issues 036–038 add trusted application-managed verification,
  packaged-by-default runtime selection, explicit fallback/override handling,
  runtime provenance, and five-analyzer parity/release verification.

## Artifact sync

- **Topology:** This child is linked under `Analyzer plugin runtime`.
- **Capability:** Discovery notes record the compiled-binary target, stable
  logical identity, application-managed discovery, build assembly, and
  one-job-per-process policy; the exact package index, trust, runtime, and
  platform rules are implemented in the linked code and tests.
- **Product and architecture:** The application synthesis is refreshed with
  the future external-binary boundary.
- **Delivery:** Issues 034–038 are archived and record the shared runner, five
  compiled entrypoints, deterministic package assembly, trusted verification,
  packaged runtime cutover, explicit fallback, parity, and release evidence.

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

IMPLEMENTED. Issues 034–038 completed the scoped compiled-distribution
delivery. Linux amd64 and Darwin arm64 execution are documented deferrals
under the root `when-supported` policy, with re-entry conditions recorded in
issue 038; the package contract and Windows release path are verified.

## Artifact impact

- Capability truth: updated with the complete exact-spec set.
- Product truth: no new actor or workflow; root PRD remains current.
- Architecture truth: package trust, platform, and launch boundaries are current.
- Delivery truth: issues 034–038 are archived after implementation and
  verification. The capability is `implemented`; the parent plugin-runtime
  node and its separate multi-analyzer and assignment/view children are also
  implemented after their verified delivery batches.
