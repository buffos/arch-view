# Analyzer plugin runtime domain glossary

| Term | Category | Definition | Distinction |
|---|---|---|---|
| Plugin | deployment term | Analyzer implementation registered with the host. | The product contract is the analyzer, not the packaging. |
| Analyzer manifest | domain object | Versioned declaration of identity, language, detection, capabilities, and options. | Not a source-analysis result. |
| Detection candidate | value object | Analyzer recommendation with confidence and marker reasons. | Not final selection until ambiguity is resolved. |
| Analyzer selection | value object | Explicit or automatically chosen analyzer for a run. | Explicit choice can override detection but not compatibility. |
| Capability | policy/metadata | Named operation or feature an analyzer supports. | Not a product capability node. |
| Analyzer session | domain object | One host-managed invocation with context, options, and lifecycle status. | Not the target project's runtime. |
| In-process analyzer | deployment term | Analyzer implementing the Go host interface inside the Arch View process. | External process analyzers implement the same meaning through a transport adapter. |
| Process plugin descriptor | deployment object | Explicit local JSON declaration containing a validated manifest and argv command for an external analyzer. | It is not project configuration and is never discovered implicitly in the pilot. |
| Process invocation | integration object | One host-owned child-process launch for a single detect or analyze request, including argv, limits, and cancellation state. | It is not a long-lived plugin daemon. |
| Protocol session | integration object | One request-scoped exchange between the host and one external analyzer process. | It is not the target project's runtime. |
| Protocol frame | integration object | One NDJSON message in a protocol session. | It is not an exported architecture artifact. |
| Manifest agreement | protocol invariant | The hello manifest must match the descriptor manifest and a supported API version. | A process cannot replace its declared identity at runtime. |
| External analyzer parity | verification concept | Agreement between an external adapter's normalized observations and the in-process baseline for the same fixture. | Run and analyzer provenance may differ by design. |

Statuses: `registered`, `selected`, `running`, `completed`, `partial`, `failed`, `cancelled`.
