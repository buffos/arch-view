# analyze

The analyze command reads a project and writes an analysis or an export.

Basic command:

~~~powershell
go run ./cmd/arch-view analyze --project . --format analysis-json --output analysis.json
~~~

## Required options

| Option | What it does |
| --- | --- |
| **--project** | The project folder to read. It is required. |
| **--output** | The file to write, or a single dash to write JSON to standard output. It is required. |

## Analyzer selection

| Option | What it does |
| --- | --- |
| **--analyzer-runtime** | Chooses auto, packaged, in-process, or explicit analyzer execution. Auto is the normal choice. |
| **--plugin** | Adds an external analyzer descriptor. Repeat it for more than one descriptor. |
| **--allow-untrusted-plugin** | Allows an explicitly supplied local descriptor. Keep this off unless you trust the file. |
| **--language** | Selects an analyzer by language, such as Go or Python. |
| **--analyzer** | Selects one exact analyzer ID. Use it when automatic selection is not enough. |
| **--scope** | Reuses a cached aggregate scope. Leave it empty for the normal All scope. It cannot be combined with explicit analyzer or language selection. |

## Source selection

| Option | What it does |
| --- | --- |
| **--module** | Selects a Go module path or workspace-relative module directory. |
| **--crate** | Selects a Rust package name or workspace-relative crate directory. |
| **--target** | Selects a Rust target triple or target selector. |
| **--config** | Gives the TypeScript analyzer a specific tsconfig path. |
| **--source-root** | Adds an explicit source root. Repeat it when a project has more than one root. |
| **--exclude** | Excludes repository-relative paths using a glob. Repeat it for multiple patterns. |
| **--build-tag** | Adds a Go build tag. Repeat it for multiple tags. |
| **--feature** | Adds a Rust Cargo feature. Repeat it for multiple features. |
| **--features** | Alias for --feature. It is repeatable too. |
| **--include-js** | Includes JavaScript and JSX files in a TypeScript analysis. |
| **--include-tests** | Includes test files. |
| **--include-examples** | Includes Rust examples and benches. |
| **--include-generated** | Includes files marked as generated. |
| **--include-external** | Keeps more detail for non-local references. |
| **--safe-mode** | Defaults to true and disables target-code execution and tool-assisted execution. |
| **--python-version** | Gives the Python analyzer a major/minor version for static interpretation. |
| **--include-stubs** | Includes Python .pyi stub files. |
| **--platform** | Chooses the Clojure reader-conditional platform: clj, cljs, or both. |
| **--runtime** | Chooses the TypeScript runtime context: auto, esm, or cjs. |

## Output options

| Option | What it does |
| --- | --- |
| **--format** | Chooses analysis-json, json, html, or svg. The default is analysis-json. |
| **--reference-visibility** | Chooses hidden, aggregated, or expanded visual references. The default is hidden. |
| **--view-path** | Selects a hierarchy segment for a visual export. Repeat it for a deeper path. |
| **--reference-scope** | Limits visual references to a selected reference scope. Repeat it when needed. |
| **--deterministic** | Defaults to true and asks for repeatable output. |
| **--overwrite** | Allows replacement of an existing export file. It is off by default. |
| **--embed-source** | Requests source embedding. It is not supported in model version 1. |

## Quality options

| Option | What it does |
| --- | --- |
| **--quality-profile** | Loads a versioned quality profile JSON file. If it references a baseline, Arch View loads the unique matching file from `quality-baselines/` automatically. |
| **--quality-baseline** | Uses one explicit baseline for this run. It does not change the profile. |
| **--no-quality-baseline** | Runs a clean report without the profile's baseline. It cannot be combined with **--quality-baseline**. |
| **--quality-exit-on** | Makes the command return exit code 1 when a finding at or above info, warning, error, or blocker matches. |
| **--quality-exit-status** | Limits the exit policy to selected statuses. Repeat it or use a comma-separated list. |

## Example: analyze and fail on errors

~~~powershell
go run ./cmd/arch-view analyze --project . --quality-profile quality-profiles/full.json --quality-exit-on error --format analysis-json --output analysis.json
~~~

The report is still written. The exit code tells automation whether the selected
policy matched. If the profile points to `baseline:main@1.0.0`, the command
reads that exact baseline from `quality-baselines/`. A missing reference warns
and leaves findings unsuppressed. An invalid or ambiguous reference stops the
command.
