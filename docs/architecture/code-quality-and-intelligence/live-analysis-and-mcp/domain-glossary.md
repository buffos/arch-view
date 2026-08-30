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
| Structural search | query capability | Deterministic filtering over paths, symbols, docs, relations, and findings. | It is not fuzzy/semantic ranking or arbitrary text grep. |
| Query projection | read model | Selected fields and bounded records returned from a snapshot. | It must not mutate the underlying snapshot. |
| Query cursor | pagination value | Opaque continuation token tied to snapshot/query/order/budget. | It is invalid when the requested consistency context changes. |
| MCP server | transport adapter | MCP-compatible read surface over shared snapshot/query services. | It is not an analyzer, rule engine, or source editor. |
| MCP tool | transport operation | Callable bounded query/report operation. | V1 tools are read-only and scope-safe. |
| MCP resource | transport object | Addressable read-only snapshot/module/report representation. | A resource is not a mutable file handle. |
| Consistency selector | query value | Request for a specific revision/snapshot or latest ready revision. | It prevents mixed-revision responses. |
| Detail budget | safety value | Maximum bytes/lines/items permitted in a response/context request. | Byte budget is deterministic; token estimates are advisory. |
| Source context | read projection | Bounded text around an indexed span/entity in an allowed root. | It is explicit and separate from default structural search. |
| Permission policy | security object | Allowed roots, operations, transport, and resource limits. | Read-only default does not imply arbitrary read access. |
| Remediation handoff | workflow boundary | Explicit request/preview that passes a finding to a separate fix workflow. | It is not autonomous MCP mutation. |
| Watch backend | strategy/plugin | OS, polling, or test implementation producing normalized event inputs. | New backends do not change the coordinator. |

Canonical live states:

- Session: `stopped`, `initializing`, `ready`, `stale`, `updating`, `degraded`,
  `failed`.
- Query consistency: `specific_revision`, `latest_ready`, `candidate_not_ready`.
- Transport operation: `read_only`; future write/execute modes require a new
  permissioned capability.
