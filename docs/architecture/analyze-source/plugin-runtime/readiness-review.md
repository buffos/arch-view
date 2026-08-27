# Analyzer plugin runtime architecture readiness review

## Findings

No High or Medium findings remain. The external pilot now has a bounded
descriptor, published protocol and descriptor schemas, explicit registration,
manifest agreement, detect/analyze lifecycle, result validation, cancellation,
output limits, a verified Python parity implementation, and an explicit CLI
and shared-consumer path.

## Residual risks

- The 8 MiB frame, 1 MiB stderr, 5 second hello, and 60 second default
  operation limits need benchmark evidence and may require later tuning.
- Interpreter availability, packaging, and platform-specific launch details
  remain deployment concerns for the opt-in Python plugin.
- Protocol migration, discovery beyond explicit descriptors, and plugin
  distribution are later extensions outside this slice.

## Readiness

READY FOR ARCHITECTURE IMPLEMENTATION

The process runtime was implemented and verified through issues 030–033 under
the existing plugin-runtime capability without changing the canonical model,
viewer, or export contracts. The capability remains `specified` for a future
frontier such as broader plugin distribution or discovery.

## Artifact impact

- Capability truth: the exact external pilot specification and published
  schemas are now linked.
- Product truth: the application remains a static, local-first architecture
  tool; external Python is an opt-in deployment of existing semantics.
- Architecture truth: descriptor, process, NDJSON, lifecycle, and consumer
  boundaries are explicit and synchronized.
- Delivery truth: issues 030–033 are archived after protocol, process-host,
  external Python parity, public CLI, shared viewer/source, export, repository,
  and strict OKF verification.
