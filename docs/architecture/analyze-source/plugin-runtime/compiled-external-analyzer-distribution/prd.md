# Compiled external analyzer distribution PRD

## Purpose

Make supported analyzer semantics available as application-managed compiled
executables so end users do not need Python, Rust, Node, or another analyzer
language runtime installed.

## Actors

- **Arch View user:** opens a repository and expects a supported analyzer to be available without installing its implementation runtime.
- **Release maintainer:** builds and packages all supported analyzer executables for a platform.
- **Developer/test operator:** explicitly selects an in-process analyzer or local descriptor while validating parity or migrating an implementation.

## Goals

1. Compile each supported analyzer from the same implementation source used by the in-process adapter.
2. Assemble a deterministic application-managed `analyzers/` tree and index.
3. Select the exact host-platform package for a stable logical analyzer ID.
4. Verify package metadata and executable bytes before launch.
5. Preserve the existing analyzer protocol, canonical model, viewer, and export semantics.
6. Fail explicitly when a package is missing, incompatible, or tampered with.

## Non-goals

- A new analyzer language implementation or a second semantic analyzer codebase.
- A new process protocol or canonical model schema.
- Scanning target repositories, the machine `PATH`, or arbitrary user folders for executables.
- Remote package registries, automatic downloads, hot updates, rollback, or code-signing policy.
- Silent fallback from a failed packaged executable to in-process execution.

## Required workflows

### Build and assemble

The release maintainer runs `make analyzers`. The build compiles every enabled
analyzer entrypoint for the requested platform, writes the executable and
descriptor under the canonical analyzer tree, computes digests, and writes one
index. `make release` includes the host application and the same tree. A build
fails if an enabled analyzer cannot be compiled or the generated index is
incomplete.

### Discover and verify

At application startup the host reads only its application-managed analyzer
index. For a requested logical analyzer ID it selects the exact host platform,
confirms the package path remains inside the analyzer root, verifies descriptor
and executable digests, validates the manifest, and registers the package. No
executable is launched during listing or discovery.

### Run and report

The host launches one verified executable for one detect or analyze operation
through the existing NDJSON process contract. The descriptor and hello
manifest must agree. Package failures are returned as stable diagnostics and
do not select an in-process implementation implicitly.

### Explicit development override

A developer may select `in-process` or provide an explicit descriptor through a
visible override. The result records the runtime source. This path is not an
automatic discovery mechanism and is not the production default.

## Functional requirements

| ID | Requirement |
|---|---|
| CED-FR-001 | Every enabled analyzer has one stable logical ID shared by in-process and packaged implementations. |
| CED-FR-002 | `make analyzers` compiles and assembles all enabled analyzer packages and a complete index. |
| CED-FR-003 | The host discovers packages only from the application-managed index and analyzer root. |
| CED-FR-004 | Platform selection is exact and deterministic; no nearest or cross-platform binary is substituted. |
| CED-FR-005 | Descriptor, index, hello manifest, and host API compatibility are validated before payload acceptance. |
| CED-FR-006 | Executable and descriptor digests are verified before launch. |
| CED-FR-007 | Packaged execution is the production default for a logical analyzer ID. |
| CED-FR-008 | In-process and explicit local-descriptor modes require an explicit runtime override and are observable. |
| CED-FR-009 | Package, platform, integrity, and compatibility failures have stable error outcomes and never silently fall back. |
| CED-FR-010 | One detect or analyze operation launches one child process and applies the existing cancellation, timeout, output, and cleanup policy. |

## Non-functional requirements

- A package index and its referenced paths are deterministic and reproducible from build inputs.
- The host never uses a shell to launch an analyzer.
- A package path cannot escape the application-managed analyzer root through traversal or symlink substitution.
- The package layer adds no language-specific fields to the canonical model.
- Listing analyzers is safe and does not execute them.

## Success criteria

The acceptance scenarios in [acceptance-scenarios.md](acceptance-scenarios.md)
pass for every enabled analyzer and each supported release platform. The
existing external protocol scenarios remain green unchanged.
