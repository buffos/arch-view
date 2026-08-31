# Live analysis and MCP domain glossary

| Term | Category | Definition | Distinction |
|---|---|---|---|
| Live session | lifecycle object | Configured watcher/coordinator instance for one or more allowed source roots. | It is not a source-index snapshot or a transport connection. |
| Watch root | security value | Normalized local directory explicitly allowed for observation. | It is narrower than arbitrary filesystem access. |
| Watch event | transient input | Create, modify, delete, rename, or overflow hint from a watcher backend. | It is not committed repository truth until rescanned/hash-verified. |
| Event normalization | process | Converts backend-specific events into root-safe canonical paths and kinds. | It removes backend differences; it does not analyze code. |
| Debounce window | lifecycle policy | Bounded period in which related filesystem hints are coalesced. | It delays work; it does not change snapshot semantics. |
| Event overflow | failure state | Condition in which a backend cannot guarantee delivery of all events. | It requires rescan, not guessed incremental facts. |
| Invalidation set | derived value | Scopes/files known or conservatively assumed to require reanalysis. | It is an execution plan, not proof of semantic impact. |
| Candidate snapshot | transient object | Unpublished source/model/quality result built from one input fingerprint. | Clients cannot observe it as current until commit. |
| Live snapshot | immutable object | Published coherent source index, model, quality, and status view. | All included components refer to one input/revision. |
| Live revision | value object | Monotonic session-local sequence assigned at atomic publication. | It is distinct from analyzer version or source content hash. |
| Last-ready revision | consistency value | Most recent committed usable revision served during an update/failure. | It may be stale relative to filesystem state. |
| Freshness | status | Relationship between filesystem hints, candidate work, and last-ready revision. | `stale` is not the same as `failed`. |
| Degraded snapshot | status | A ready snapshot with explicit partial analysis/diagnostics. | It remains queryable but must disclose limitations. |
| Snapshot store | strategy | Component that atomically publishes and retrieves immutable revisions. | It does not decide language or quality semantics. |
| Analyzer capability | coverage value | A namespaced fact or relation an analyzer/source provider can produce for a scope. | MCP negotiates it; MCP does not assume a language-specific capability. |
| Structural search | query capability | Deterministic filtering over paths, symbols, docs, relations, and findings. | It is not fuzzy/semantic ranking. |
| Exact text search | query capability | Bounded literal or explicitly requested regular-expression matching that returns file/line locations. | It is not embedding search and does not return whole files by default. |
| Query projection | read model | Selected fields and bounded records returned from a snapshot. | It must not mutate the underlying snapshot. |
| Query cursor | pagination value | Opaque continuation token tied to snapshot/query/order/budget. | It is invalid when the requested consistency context changes. |
| MCP server | transport adapter | MCP-compatible bounded surface over shared snapshot, query, and explicitly permitted quality-policy services. | It is not an analyzer, rule engine, or source editor. |
| MCP tool | transport operation | Callable bounded query, analysis, or explicitly authorized quality-policy operation. | Read operations are default; policy writes are scope-safe and separately granted. |
| MCP resource | transport object | Addressable read-only snapshot/module/report representation. | A resource is not a mutable file handle. |
| Consistency selector | query value | Request for a specific revision, latest ready revision, or a verified current revision. | `require_current` performs reconciliation; it is not satisfied by a stale response. |
| Reconciliation | freshness process | Compares the configured source manifest/input identity with the published revision before a strict query. | It verifies freshness; it is not a full analyzer by itself. |
| Stable input | freshness result | Source state that did not change during the bounded scan and fingerprint verification. | An unstable edit stream cannot be published as current. |
| Input unstable | freshness state | The source changed during repeated scan attempts or did not settle within policy. | It is different from analyzer failure and from an empty result. |
| Quality evaluation request | quality command | Request to evaluate a profile or temporary rule bindings against a verified source/model snapshot. | It may be non-persisted and must carry profile/options identity. |
| Quality-policy write | permissioned command | Explicit save of a profile or creation/append of a baseline through deterministic-quality services. | It is not source mutation and is not enabled by default. |
| Baseline preview | quality command | Dry-run list of finding keys and exact policy identities that a baseline would suppress. | It does not change the profile or report. |
| Finding delta | quality query | Added, unchanged, suppressed, or resolved findings between compatible reports. | Partial or incompatible reports cannot imply resolution. |
| Detail budget | safety value | Maximum bytes/lines/items permitted in a response/context request. | Byte budget is deterministic; token estimates are advisory. |
| Source context | read projection | Bounded text around an indexed span/entity in an allowed root. | It is explicit and separate from default structural search. |
| Permission policy | security object | Allowed roots, operations, transport, and resource limits. | Read-only default does not imply arbitrary read access. |
| Remediation handoff | workflow boundary | Explicit request/preview that passes a finding to a separate fix workflow. | It is not autonomous MCP mutation. |
| Watch backend | strategy/plugin | OS, polling, or test implementation producing normalized event inputs. | New backends do not change the coordinator. |

Canonical live states:

- Session: `stopped`, `initializing`, `ready`, `stale`, `updating`, `degraded`,
  `failed`, `input_unstable`.
- Query consistency: `specific_revision`, `latest_ready`, `require_current`,
  `candidate_not_ready`.
- Operation class: `read`, `analysis`, or `quality_policy_write`.
- Coverage: `observed`, `partial`, `unknown`, `unsupported`, `not_evaluable`,
  or `unavailable`; no missing capability is represented as an empty success.
