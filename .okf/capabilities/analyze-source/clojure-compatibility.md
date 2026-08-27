---
type: capability
title: Clojure compatibility
description: Preserve the useful behavior of the reference Clojure analyzer through the language-neutral plugin contract.
tags: [clojure, compatibility, reference]
timestamp: 2026-08-27T13:51:53Z
state: implemented
state_changed: 2026-08-27T13:51:53Z
project: /project.md
parent: /capabilities/analyze-source.md
discovery_notes: docs/architecture/analyze-source/clojure-compatibility/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/clojure-compatibility/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/clojure-compatibility/orchestration-status.md
prd: docs/architecture/analyze-source/clojure-compatibility/prd.md
glossary: docs/architecture/analyze-source/clojure-compatibility/domain-glossary.md
domain_model: docs/architecture/analyze-source/clojure-compatibility/canonical-domain-model.md
use_cases: docs/architecture/analyze-source/clojure-compatibility/canonical-use-cases.md
contract: docs/architecture/analyze-source/clojure-compatibility/canonical-api-cli-contract.md
scenarios: docs/architecture/analyze-source/clojure-compatibility/acceptance-scenarios.md
readiness_review: docs/architecture/analyze-source/clojure-compatibility/readiness-review.md
issues:
  - docs/agents/issues/done/20260827-026-clojure-project-and-namespace-discovery.md
  - docs/agents/issues/done/20260827-027-clojure-static-namespace-dependencies.md
  - docs/agents/issues/done/20260827-028-clojure-platform-polymorphism-and-safety.md
  - docs/agents/issues/done/20260827-029-clojure-public-integration-and-deterministic-exports.md
---

# Intent

Keep the reference tool's Clojure-oriented architecture discovery available without making Clojure assumptions part of the shared core.

# Scope

The adapter covers Clojure-family source discovery, namespace and dependency extraction, source-file evidence, and language-specific abstraction markers. The upstream [reference implementation](https://github.com/unclebob/arch-view) remains read-only design material.

# Relationships

- Parent: [Analyze source code](../analyze-source.md)
- Reference input: [upstream reference repository](https://github.com/unclebob/arch-view)
- Produces input for: [Generate architecture models](../generate-models.md)

# Planning state

This child capability is implemented. Its reference-compatible source/configuration rules, namespace/dependency extraction, platform metadata, polymorphic tags, diagnostics, public shared-path integration, and read-only policy are linked from the exact-spec artifacts.

## Delivery progress

Issues 026–029 are complete: project and namespace discovery, static dependency extraction, platform/polymorphic/safety metadata, and public shared-path integration were implemented and verified on the isolated Clojure analyzer branch.
