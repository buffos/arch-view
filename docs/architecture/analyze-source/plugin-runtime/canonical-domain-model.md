# Analyzer plugin runtime canonical domain model

## Registry aggregate

`AnalyzerRegistry` owns a set of manifests and implementations.

### AnalyzerManifest

Required fields: `id`, `version`, `language`, `api_version`, `detection_markers[]`, `capabilities[]`, `options[]`.

`id` is lowercase stable reverse-DNS or namespace-like text; `version` is semantic-version text; `api_version` is `arch-view.analyzer/v1` for the first contract.

Each option declares `name`, `type`, `default`, `allowed_values?`, `description`, and `sensitive=false` unless explicitly supported.

### DetectionCandidate

`analyzer_id`, `confidence` (`0..1`), `matched_markers[]`, `boundary_hint?`, `reason`.

### AnalyzerSession

`session_id`, `project_root`, `selection`, `effective_options`, `status`, `started_at?`, `finished_at?`, `result?`, `diagnostics[]`.

Invariants: one selected analyzer; options are validated before running; terminal statuses cannot transition back; cancellation is terminal.

## External process objects

### ProcessPluginDescriptor

Required fields are schema_version, manifest, and command. The descriptor
also supports argv arguments and an optional working directory. Relative
command/script paths resolve from the descriptor directory. The command and
arguments are passed directly to the operating system; they are never parsed
as a shell command.

Invariants: the manifest is valid before registration; the descriptor is
loaded only by explicit user or test configuration; a process cannot replace
the descriptor manifest during a handshake.

### ProcessInvocation

Fields are operation, request_id, command, argv arguments, working directory,
project root, effective options, frame limit, stderr limit, hello deadline,
operation deadline, and cancellation state.

Invariants: command and arguments are passed as separate operating-system
arguments with no shell; the project root and options are sent in the
request frame; limits are host-owned; one invocation has one child process and
one terminal outcome.

### ProtocolSession

Fields are request_id, operation, process identity, protocol version,
started_at, finished_at, received frames, streamed diagnostics, terminal
status, and bounded stderr context.

Invariants: one process handles one detect or analyze operation; hello is the
first frame; request frames use one request_id; exactly one candidate or
result is accepted; done or fatal is terminal; frames after terminal,
unknown frame types, malformed payloads, and oversized lines are rejected.

## Extension points

Capabilities, analyzer versions, option schemas, process transports, and
future language adapters extend the registry without changing model
vocabulary. Process transport is an implementation of the Analyzer contract,
not a second result model.

## Domain events

`AnalyzerRegistered`, `AnalyzerSelected`, `AnalyzerStarted`, `AnalyzerProducedPartialResult`, `AnalyzerCompleted`, `AnalyzerFailed`, `AnalyzerCancelled`.
