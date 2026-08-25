# Analyzer plugin runtime domain glossary

| Term | Category | Definition | Distinction |
|---|---|---|---|
| Plugin | deployment term | Analyzer implementation registered with the host. | The product contract is the analyzer, not the packaging. |
| Analyzer manifest | domain object | Versioned declaration of identity, language, detection, capabilities, and options. | Not a source-analysis result. |
| Detection candidate | value object | Analyzer recommendation with confidence and marker reasons. | Not final selection until ambiguity is resolved. |
| Analyzer selection | value object | Explicit or automatically chosen analyzer for a run. | Explicit choice can override detection but not compatibility. |
| Capability | policy/metadata | Named operation or feature an analyzer supports. | Not a product capability node. |
| Analyzer session | domain object | One host-managed invocation with context, options, and lifecycle status. | Not the target project's runtime. |
| In-process analyzer | deployment term | Analyzer implementing the Go host interface. | External process is a later adapter. |
| Protocol frame | integration object | One NDJSON message in the future process boundary. | Not an exported architecture artifact. |

Statuses: `registered`, `selected`, `running`, `completed`, `partial`, `failed`, `cancelled`.
