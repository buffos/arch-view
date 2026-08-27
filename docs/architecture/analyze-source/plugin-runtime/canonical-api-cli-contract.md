# Analyzer plugin runtime canonical API/CLI contract

## Manifest shape

~~~json
{
  "id": "org.example.go",
  "version": "1.0.0",
  "language": "go",
  "api_version": "arch-view.analyzer/v1",
  "detection_markers": [{"kind": "file", "value": "go.mod", "weight": 1}],
  "capabilities": ["detect", "static_dependencies"],
  "options": [{"name": "include_tests", "type": "boolean", "default": false, "sensitive": false}]
}
~~~

The existing manifest validation rules remain authoritative: IDs and
languages are lowercase, versions are semantic versions, API major version 1
is supported, markers have positive weights, capabilities and option names
are unique, and option defaults are valid.

## External plugin descriptor

An external analyzer is loaded only from an explicitly supplied local JSON
descriptor. The descriptor is not analyzed-project configuration.

~~~json
{
  "schema_version": "arch-view.plugin/v1",
  "manifest": {
    "id": "org.archview.python.external",
    "version": "1.0.0",
    "language": "python",
    "api_version": "arch-view.analyzer/v1",
    "detection_markers": [
      {"kind": "file", "value": "pyproject.toml", "weight": 1}
    ],
    "capabilities": ["detect", "static_dependencies"],
    "options": []
  },
  "command": "python",
  "args": ["analyzer.py"],
  "working_directory": "."
}
~~~

The command and each argument are passed as argv values. No shell is
involved. Relative command/script paths and the optional working directory
resolve from the descriptor directory. The descriptor schema is published at
[external-plugin-descriptor-v1.schema.json](external-plugin-descriptor-v1.schema.json).

## In-process Go contract

~~~go
type Analyzer interface {
    Manifest() Manifest
    Detect(context.Context, DetectRequest) (DetectionCandidate, error)
    Analyze(context.Context, AnalyzeRequest) (AnalysisResult, error)
}
~~~

The host owns context cancellation, option resolution, panic recovery, result
validation, normalization, and deterministic ordering. A process-backed
adapter implements this same interface.

## External NDJSON protocol v1

One process handles one request. Each line on stdout is one JSON object. The
first frame is hello and has no request ID. Every later frame has a non-empty
request_id. The protocol envelope is published at
[external-protocol-v1.schema.json](external-protocol-v1.schema.json).

### Host-to-plugin frames

- detect contains type, request_id, and project_root.
- analyze contains type, request_id, project_root, selection, and effective
  options.
- cancel contains type, request_id, and an optional reason. The host sends it
  best effort before terminating a cancelled or timed-out process.

Example:

~~~json
{"type": "analyze", "request_id": "req-1", "project_root": "/repo", "selection": {"analyzer_id": "org.archview.python.external", "mode": "explicit-id"}, "options": {"values": {}, "sources": {}, "fingerprint": "sha256:..."}}
~~~

### Plugin-to-host frames

- hello is the first frame and contains protocol plus the complete manifest.
- candidate contains exactly one detection candidate for a detect request.
- result contains exactly one canonical AnalysisResult for an analyze request.
- diagnostic contains a structured diagnostic that the host retains before
  common result validation.
- done contains the terminal status.
- fatal contains a stable process/protocol code, message, and optional details.

Example exchange:

~~~json
{"type": "hello", "protocol": "arch-view.analyzer/v1", "manifest": {"id": "org.archview.python.external", "version": "1.0.0", "language": "python", "api_version": "arch-view.analyzer/v1", "detection_markers": [{"kind": "file", "value": "pyproject.toml", "weight": 1}], "capabilities": ["detect", "static_dependencies"], "options": []}}
{"type": "candidate", "request_id": "req-1", "candidate": {"analyzer_id": "org.archview.python.external", "confidence": 1, "matched_markers": ["pyproject.toml"], "reason": "detected by pyproject.toml"}}
{"type": "done", "request_id": "req-1", "status": "complete"}
~~~

For analysis, diagnostic frames may precede the result. The result payload
uses the existing AnalysisResult shape; streamed diagnostics are merged and
deduplicated before ValidateAnalysisResult runs. Exactly one result and one
terminal done or fatal are accepted. Frames after a terminal frame are
invalid. A done status must agree with the result status when a result exists.

The hello protocol and manifest API version must be supported by the host, and
the hello manifest must match the descriptor manifest. The host does not
silently downgrade an incompatible major version.

The first executable contract slice is implemented by the typed
`internal/analysis/processprotocol` package. It provides bounded frame
encoding/decoding, strict descriptor validation, stateful one-request
conformance checks, and deterministic streamed-diagnostic merging. Its
subprocess fixture is test-only and is not registered as a public analyzer;
process launch, child cleanup, and shared host integration remain in issues
031–033.

### Lifecycle and safety

- Stdout is protocol-only. Human-readable plugin logs belong on stderr.
- A valid run exits after done with a successful process exit code. A
  premature/non-zero exit, missing terminal frame, or fatal frame is a host
  analyzer failure, except that cancellation and timeout map to the existing
  cancelled outcome.
- The initial host limits are an 8 MiB maximum NDJSON line, 1 MiB captured
  stderr context, a 5 second hello deadline, and a 60 second operation
  deadline when the caller supplies no earlier deadline. The limits are
  adapter configuration and may be tuned only with benchmark evidence.
- Cancellation or timeout terminates the child after best-effort cancel,
  waits for the process and pipes, and rejects any later result.
- The host never executes target project code. The external analyzer is an
  explicitly selected analysis tool and must honor the same no-evaluation
  contract as built-in analyzers.

## HTTP mapping

- GET /v1/analyzers lists manifests loaded into the host registry.
- arch-view analyzers follows the same listing contract.
- Analyzer execution uses the parent analysis contract; runtime failures map
  to its stable error codes and exit codes.

## CLI mapping

- arch-view analyzers [--plugin <descriptor>] lists built-in and explicitly
  loaded external manifests in deterministic order.
- arch-view analyze --project <path> [--language <id>] [--analyzer <id>]
  [--plugin <descriptor>] [analyzer options] --format analysis-json
  --output <file>.
- arch-view open --project <path> [--language <id>] [--analyzer <id>]
  [--plugin <descriptor>] [analyzer options] [--port <n>].
- Without --plugin, the existing built-in analyzer registry and output remain
  unchanged. When an external analyzer is loaded alongside a built-in
  analyzer for the same language, the existing explicit-selection and
  ambiguity rules apply.

Exit codes remain 0 for complete or partial results, 2 for invalid request,
manifest, options, or ambiguity, 3 for unsupported/unreadable projects, 4
for fatal analyzer/host failure, and 130 for cancellation.

## Compatibility

API major versions are incompatible. A minor version may add optional fields
or capabilities only when the host can validate them; the host rejects a
manifest or frame it cannot validate and never silently changes semantics.

## Parity and safety

HTTP, CLI, in-process analyzers, and process analyzers preserve status,
diagnostics, identifiers, and result meaning. Paths are normalized relative
to the project boundary. The external Python pilot is compared with the
in-process Python analyzer after allowing only analyzer/run provenance to
differ.
