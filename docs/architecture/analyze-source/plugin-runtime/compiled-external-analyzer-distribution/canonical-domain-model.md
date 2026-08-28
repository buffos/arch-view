# Compiled external analyzer distribution canonical domain model

## Modeling boundary

This model describes distribution and runtime selection. Analyzer semantics
remain in the language analyzer; protocol payloads remain in the parent
plugin-runtime contract; project assignments remain in the assignment child.

## Aggregate: AnalyzerDistribution

The distribution owns one immutable index and its platform packages for an
application installation.

### AnalyzerIndex

Fields:

- `schema_version`: `arch-view.analyzer-index/v1`
- `host_api_version`: `arch-view.analyzer/v1`
- `generated_by`: build identifier
- `packages[]`: immutable package entries

Invariant: package entries are unique by `(logical_analyzer_id, platform)` and
are sorted by those fields when serialized.

### AnalyzerPackage

Fields:

- `logical_analyzer_id`
- `analyzer_version`
- `language`
- `api_version`
- `platform`
- `executable_path`
- `descriptor_path`
- `executable_sha256`
- `descriptor_sha256`

Invariant: paths are relative to the analyzer root, use normalized separators,
contain no `..`, and resolve beneath the application-managed root.

### PackageManifest

The descriptor carries the existing analyzer manifest plus package metadata.
Its logical ID, semantic version, language, API version, capabilities, and
options must match the index and the hello frame byte-for-byte after canonical
JSON normalization.

### RuntimeSelection

Fields: `logical_analyzer_id`, `mode`, `platform`, `package`, `override_source`.

`mode` is one of `packaged`, `in-process`, or `explicit`. `packaged` is the
default. `in-process` and `explicit` are valid only when directly requested by
the caller or a development/test configuration.

## Value objects and policies

- **PlatformTarget:** lowercase `<os>-<arch>`; first matrix is
  `windows-amd64`, `linux-amd64`, `darwin-arm64`.
- **IntegrityDigest:** lowercase 64-character SHA-256 hex string.
- **CompatibilityKey:** `(logical ID, analyzer version, language, API major)`.
- **TrustPolicy:** packaged index plus digest verification is required for
  automatic discovery; explicit overrides are visibly untrusted/development.

## Lifecycle

`assembled -> indexed -> verified -> available -> selected -> launched`.

Failure transitions are `indexed -> rejected`, `indexed -> unavailable`, or
`selected -> failed`. A rejected package never becomes available in the same
application session.

## Invariants

1. A logical analyzer ID is stable across runtime modes.
2. An unavailable or invalid packaged binary never triggers implicit in-process fallback.
3. A package is not launched before all path, digest, descriptor, manifest, and API checks pass.
4. The package layer cannot mutate canonical model facts or protocol meaning.
5. Package listing has no execution side effects.

## Domain events

`AnalyzerDistributionAssembled`, `AnalyzerPackageIndexed`,
`AnalyzerPackageVerified`, `AnalyzerPackageRejected`,
`AnalyzerRuntimeSelected`, `AnalyzerPackageLaunchFailed`.

## Mapping guidance

Layered, modular-monolith, and process-hosted implementations may store the
index differently, but all must expose the same selection, trust, failure, and
runtime-source semantics.
