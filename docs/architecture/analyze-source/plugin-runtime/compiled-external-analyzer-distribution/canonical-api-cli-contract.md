# Compiled external analyzer distribution canonical API/CLI contract

## Package index contract

The application-managed `analyzers/index.json` uses:

```json
{
  "schema_version": "arch-view.analyzer-index/v1",
  "host_api_version": "arch-view.analyzer/v1",
  "generated_by": "arch-view-build-<id>",
  "packages": [
    {
      "logical_analyzer_id": "org.archview.python",
      "analyzer_version": "1.0.0",
      "language": "python",
      "api_version": "arch-view.analyzer/v1",
      "platform": "windows-amd64",
      "executable_path": "org.archview.python/windows-amd64/analyzer.exe",
      "descriptor_path": "org.archview.python/windows-amd64/descriptor.json",
      "executable_sha256": "<64 lowercase hex characters>",
      "descriptor_sha256": "<64 lowercase hex characters>"
    }
  ]
}
```

The package directory is rooted at `analyzers/`. Index and descriptor paths are
relative, normalized, and cannot escape that root. Each descriptor retains the
existing `arch-view.plugin/v1` manifest and points to the executable using argv
metadata; it does not change the NDJSON protocol.

## Runtime selection

The transport-neutral runtime mode is:

```json
{
  "analyzer_id": "org.archview.python",
  "runtime_mode": "packaged",
  "runtime_source": "application-index",
  "platform": "windows-amd64"
}
```

Allowed modes:

- `packaged` — default and required for normal releases.
- `in-process` — explicit development/test/migration mode.
- `explicit` — explicit local descriptor mode; requires a developer/test
  override and is never discovered from the target repository.

The selected package's index entry, descriptor, hello manifest, logical ID,
language, analyzer version, and API major must agree.

## Errors

Stable error codes are:

| Code | Meaning |
|---|---|
| `analyzer_package_index_invalid` | The application index is malformed or incompatible. |
| `analyzer_package_not_found` | No exact package exists for the logical ID and host platform. |
| `analyzer_platform_unsupported` | The requested platform is not in the supported matrix. |
| `analyzer_package_integrity_mismatch` | Descriptor or executable digest differs from the index. |
| `analyzer_package_manifest_mismatch` | Descriptor, index, or hello manifest disagrees. |
| `analyzer_package_api_incompatible` | The package API major is not supported by the host. |
| `analyzer_runtime_override_required` | An untrusted/explicit runtime was requested without its opt-in. |
| `analyzer_package_launch_failed` | A verified executable could not be launched or completed. |

HTTP mappings use the common error body `{ "error": { "code", "message", "details" } }`:

- `GET /v1/analyzers` returns packaged availability, platform, version, and
  runtime source without launching analyzers.
- `400` is used for invalid runtime mode requests.
- `409` is used for an unavailable or conflicting runtime selection.
- `422` is used for incompatible package metadata.
- `500` is used for verified launch/process failure.

## CLI mapping

```text
arch-view analyzers
arch-view analyze --project <path> --analyzer <logical-id> --analyzer-runtime packaged
arch-view analyze --project <path> --analyzer-runtime in-process
arch-view analyze --project <path> --plugin <descriptor> --analyzer-runtime explicit --allow-untrusted-plugin
```

`--analyzer-runtime packaged` is the default. `in-process` and `explicit`
require an explicit invocation and are reported in diagnostics/result
provenance. `--plugin` is rejected in packaged mode. The existing analyzer
protocol flags, options, timeouts, and exit codes remain authoritative.

## Build contract

- `make analyzers` builds all enabled analyzer packages for the current target
  and validates the generated index.
- `make release` builds the host and packages under the application release
  tree.
- A successful target leaves `analyzers/index.json` plus one package directory
  per enabled analyzer/platform. A failed analyzer fails the target and does
  not publish a replacement index.

## Parity rules

The runtime mode and package provenance may differ, but analyzer manifest
meaning, process protocol, canonical observations, diagnostics, status, and
viewer/export behavior must remain equivalent.
