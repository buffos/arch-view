# Compiled external analyzer distribution canonical use cases

## Application services

### AnalyzerDistributionService

- `AssembleAnalyzerDistribution`
- `IndexAnalyzerPackages`
- `ListAvailableAnalyzers`
- `VerifyAnalyzerPackage`
- `SelectAnalyzerRuntime`

### AnalyzerLaunchBoundary

- `LaunchVerifiedAnalyzerOperation`
- `RejectUntrustedAutomaticPackage`
- `RecordRuntimeFailure`

## Commands and queries

### `AssembleAnalyzerDistribution` — command

**Input:** enabled analyzer IDs, build platform, compiler/toolchain inputs.

**Responsibilities:** compile each analyzer entrypoint, produce descriptors,
calculate digests, generate the index, validate uniqueness/completeness, and
write the assembled tree atomically.

**Outcomes:** complete distribution or a build failure naming each failed
analyzer. A partial index is not publishable.

### `ListAvailableAnalyzers` — query

**Input:** application installation.

**Responsibilities:** read and validate the index and report available,
unavailable, rejected, or override analyzers without launching a process.

### `VerifyAnalyzerPackage` — command

**Input:** index entry and application analyzer root.

**Responsibilities:** validate safe paths, file existence, descriptor digest,
executable digest, manifest compatibility, and platform match.

**Outcomes:** verified package or a stable rejection (`package_not_found`,
`integrity_mismatch`, `manifest_mismatch`, `api_incompatible`,
`platform_unsupported`).

### `SelectAnalyzerRuntime` — command

**Input:** logical analyzer ID, caller runtime mode, host platform, registry.

**Rules:** packaged mode is the default; explicit in-process or local-descriptor
mode is accepted only when requested. Package failure is terminal for packaged
selection and does not change mode.

### `LaunchVerifiedAnalyzerOperation` — command

**Input:** verified package, detect/analyze request, cancellation context.

**Responsibilities:** launch one child with argv, perform the existing hello
handshake, exchange one operation, enforce process limits, and return the
common analyzer result/status.

**Retry:** a caller may retry a failed operation as a new invocation; the host
does not retry automatically because analyzers may be expensive.

## Failure model

- `DistributionBuildFailed`
- `PackageIndexInvalid`
- `PackageNotFound`
- `PlatformUnsupported`
- `IntegrityMismatch`
- `ManifestMismatch`
- `ApiIncompatible`
- `RuntimeOverrideRejected`
- `AnalyzerLaunchFailed`
- `AnalyzerProtocolFailed`

## Events at the application boundary

Successful verification emits `AnalyzerPackageVerified`; a runtime operation
still emits the parent analyzer lifecycle events. The distribution service
does not emit model or viewer events.

## Architecture-neutral mapping

The build may be implemented by Make, a CI runner, or another reproducible
build coordinator. The application service semantics remain identical and the
host remains responsible for final verification immediately before launch.
