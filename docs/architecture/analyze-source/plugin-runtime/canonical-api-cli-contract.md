# Analyzer plugin runtime canonical API/CLI contract

## Manifest shape

```json
{
  "id": "org.example.go",
  "version": "1.0.0",
  "language": "go",
  "api_version": "arch-view.analyzer/v1",
  "detection_markers": [{"kind":"file","value":"go.mod","weight":1}],
  "capabilities": ["detect", "static_dependencies"],
  "options": [{"name":"include_tests","type":"boolean","default":false}]
}
```

## In-process Go contract

```go
type Analyzer interface {
    Manifest() Manifest
    Detect(context.Context, DetectRequest) (DetectionCandidate, error)
    Analyze(context.Context, AnalyzeRequest) (AnalysisResult, error)
}
```

The host owns context cancellation, option resolution, panic recovery, result validation, normalization, and deterministic ordering.

## External NDJSON protocol v1

One process handles one request. Each line is one JSON object. Required frame types: `hello`, `analyze`, `result`, `diagnostic`, `cancel`, `done`, `fatal`. `hello` declares `api_version`, manifest, and capabilities; `analyze` carries the canonical request; `result` carries one validated result envelope; `done` carries terminal status. stdout is protocol-only, stderr is log-only, and unknown frame types are fatal compatibility errors.

## HTTP/CLI mapping

- `GET /v1/analyzers` lists manifests.
- `arch-view analyzers` lists manifests in deterministic order.
- Analyzer execution uses the parent analysis contract; runtime-specific failures map to its stable error codes.

## Compatibility

Major API versions are incompatible. Minor versions may add optional fields/capabilities. A host must reject a manifest it cannot validate; it must not silently downgrade semantics.
