# Analyzer plugin runtime architecture readiness review

## Findings

No High or Medium findings remain. The external pilot now has a bounded
descriptor, published protocol and descriptor schemas, explicit registration,
manifest agreement, detect/analyze lifecycle, result validation, cancellation,
output limits, and a concrete Python parity target.

## Residual risks

- The 8 MiB frame, 1 MiB stderr, 5 second hello, and 60 second default
  operation limits need benchmark evidence and may require later tuning.
- Interpreter availability, packaging, and platform-specific launch details
  remain deployment concerns for the opt-in Python plugin.
- Protocol migration, discovery beyond explicit descriptors, and plugin
  distribution are later extensions outside this slice.

## Readiness

READY FOR ARCHITECTURE IMPLEMENTATION

The process runtime can be implemented in issues 030–033 under the existing
plugin-runtime capability without changing the canonical model, viewer, or
export contracts.

## Artifact impact

- Capability truth: the exact external pilot specification and published
  schemas are now linked.
- Product truth: the application remains a static, local-first architecture
  tool; external Python is an opt-in deployment of existing semantics.
- Architecture truth: descriptor, process, NDJSON, lifecycle, and consumer
  boundaries are explicit and synchronized.
- Delivery truth: issue 030 is complete for the protocol fixture and typed
  contract foundation; issues 031–033 remain ordered for host adapter, Python
  parity, and public shared-path integration.
