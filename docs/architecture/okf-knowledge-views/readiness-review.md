# Configurable OKF knowledge views — architecture readiness review

## Scope and state evaluation

- Owning node: Configurable OKF knowledge views
- State policy: own-state; no roll-up policy is declared
- State at review start: bounded
- Review purpose: determine whether the node-scoped reference set is coherent
  enough to advance to specified

## Reviewed artifacts

- ../../.okf/capabilities/okf-knowledge-views.md
- discovery-notes.md
- requirements-gap-analysis.md
- domain-glossary.md
- prd.md
- canonical-domain-model.md
- canonical-use-cases.md
- canonical-api-cli-contract.md
- acceptance-scenarios.md
- ../../prd.md
- ../application-architecture-summary.md
- ../../.okf/project.md and its root verification policy

## Findings

No material High, Medium, or Low findings were identified in the current
reference set.

## Cross-document consistency

### Vocabulary

The glossary, PRD, domain model, use-case model, and contract consistently use
Viewer user, Bundle maintainer, Profile maintainer, OKF bundle, Concept
document, BundleIndex/lossless index, ViewProfile, containment, semantic link,
ProjectionSnapshot, diagnostic, declared state, effective state, and planning
state. The documents preserve bundle/source identity separately from projected
scene nodes and distinguish rule strategies from rule invocations.

### Workflows and use cases

The PRD workflows map directly to the application commands and queries:
refresh/select bundle, select/apply profile, set depth, focus/back, inspect
concept, bind profile, Save, Save As, rename, delete, and diagnostic reads.
The HTTP mapping preserves those intents, and the reserved CLI mapping is
explicitly parity guidance rather than an unannounced first-slice requirement.

### Rules and invariants

Bundle independence, read-only source consumption, hierarchy precedence,
containment forest validity, semantic-link separation, explicit roll-up
behavior, state neutrality, profile immutability, deterministic rule
composition, application hard caps, safe Markdown, and atomic configuration
writes appear in the PRD, domain invariants/policies, application outcomes,
contract errors/statuses, and acceptance scenarios.

### Failure semantics

Invalid candidates, unavailable resources, hierarchy conflicts, unsupported
rules, stale profile bindings, invalid depth/limits, revision conflicts,
idempotency conflicts, unsafe links, persistence failures, layout failures,
timeouts, cancellation, supersession, and truncation have visible outcomes.
The contract distinguishes displayable truncation from failed projection.

### Extensibility

The requested Open-Closed strategy/registry design is modeled as a bounded
in-process extension seam. Profiles remain data-only, registry metadata is
versioned and namespaced, rule output is renderer-neutral, and external
repository code is not executable. This supports future growth without making
external plugins a first-slice dependency.

## Readiness checks

- Bounded capability scope: pass.
- Stable domain vocabulary: pass; glossary exists.
- Explicit actors and workflows: pass.
- Modeled invariants and lifecycle behavior: pass.
- Stable command/query intent: pass.
- Stable transport-neutral contract and error vocabulary: pass.
- Acceptance scenarios: pass; SC-001 through SC-020 are stable and
  user-visible.
- Verification surface classification: pass; each scenario records backend,
  frontend, and end-to-end intent under the inherited when-supported policy.
- Contradiction check: pass; no behaviorally material contradiction found.
- Architecture neutrality: pass; domain structures are rationale-backed
  semantics, HTTP/CLI are adapters, and events/registry implementations are
  not mandated deployment styles.
- Product-surface reachability: pass; discovery, selection, profile changes,
  projection, detail inspection, navigation, and profile persistence are
  reachable in the stated viewer journey.

## Residual risks and follow-up checks

These are implementation and verification risks, not readiness blockers:

- benchmark the recommended node/relationship/layout budgets with small and
  large representative bundles;
- verify cross-platform configuration replacement and revision conflict
  behavior;
- verify Markdown sanitization and local-link resolution with adversarial
  fixtures;
- verify that the new renderer-neutral scene maps to the existing ELK/SVG
  path without changing architecture-view behavior;
- add contract/domain/frontend tests for the stable scenario IDs once the
  implementation surface exists;
- decide whether a public CLI or OKF export should be promoted from the
  documented future mapping into a later capability scope.

## Application-synthesis gate

The required root documents exist and already describe the bounded OKF
capability:

- ../../prd.md
- ../application-architecture-summary.md

The required synchronization refresh is complete: the root PRD and
architecture summary link the node-scoped exact artifacts and identify the
capability as specified. The application-synthesis gate passes for this
capability. Delivery issue slicing remains a separate next step.

## Artifact-impact classification

- Capability: affected; the complete exact-spec set is now present.
- Product: affected; root product truth must link the capability PRD and
  acceptance behavior.
- Architecture: affected; root architecture truth must link the domain,
  use-case, contract, and readiness boundaries and preserve the shared ELK
  relationship.
- Topology: affected; the node state, artifact references, index totals, and
  log are synchronized.
- Delivery: no impact; no implementation issues or issue references are
  created by this review.

## Review conclusion

The node-scoped artifact set is coherent and ready for the specified planning
state. Application synthesis, map synchronization, and final validation are
complete. No clarification is required and no High or Medium finding blocks
delivery planning.
