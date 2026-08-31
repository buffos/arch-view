# Live sessions

Live mode keeps one analysis session open while you edit a project. Arch View
rechecks the project after changes and keeps the last verified result available
while a new result is being built.

## Start

~~~powershell
go run ./cmd/arch-view live start --project . --port 0
~~~

The command prints JSON with the session ID, the local HTTP endpoint, and the
status endpoint. Keep this terminal open. Press `Ctrl+C` to stop the session.

Use `--no-watch` when you want a long-lived session that changes only when a
client asks for a current revision.

### Start options

| Option | What it does |
| --- | --- |
| **--project** | Repository directory to analyze. Required. |
| **--port** | Loopback HTTP port. `0` chooses an available port. |
| **--session-id** | Optional stable ID for this session. |
| **--quality-profile** | Loads a quality profile JSON file for the session. |
| **--no-source-index** | Disables source files, symbols, and documentation queries. |
| **--no-watch** | Disables filesystem watching. Strict currentness can still reconcile the files. |
| **--watch-root** | Restricts watching to a repository-relative directory. Repeat it for several roots. |
| **--analyzer-runtime** | Selects `auto`, `packaged`, `in-process`, or `explicit` analyzer runtime. |
| **--plugin** | Supplies an external analyzer descriptor. Repeatable; use the explicit runtime permission. |
| **--allow-untrusted-plugin** | Allows the explicitly supplied local descriptor. |
| **--language** | Selects an analyzer language. |
| **--analyzer** | Selects an analyzer ID. |
| **--module** | Selects a Go module or workspace-relative directory. |
| **--build-tag** | Adds a Go build tag. Repeatable. |
| **--crate** | Selects a Rust crate or workspace-relative directory. |
| **--feature** | Enables a Rust Cargo feature. Repeatable. |
| **--target** | Selects a Rust target. |
| **--config** | Selects a TypeScript `tsconfig` file. |
| **--source-root** | Adds an analyzer source root. Repeatable. |
| **--exclude** | Excludes a repository-relative glob. Repeatable. |
| **--include-js** | Includes JavaScript and JSX files in TypeScript analysis. |
| **--include-tests** | Includes test files. |
| **--include-examples** | Includes Rust examples and benches. |
| **--include-generated** | Includes generated files. |
| **--include-external** | Keeps non-local reference details. |
| **--safe-mode** | Keeps target execution and tool-assisted execution disabled. |
| **--python-version** | Sets the Python major/minor version used by static analysis. |
| **--include-stubs** | Includes Python `.pyi` files. |
| **--platform** | Selects Clojure `clj`, `cljs`, or `both` reader conditionals. |
| **--runtime** | Selects the TypeScript runtime context: `auto`, `esm`, or `cjs`. |
| **--allow-policy-writes** | Enables the separately protected profile/baseline write operations. Off by default. |
| **--policy-token** | Supplies the opaque token required with `--allow-policy-writes`. |

## Status

~~~powershell
go run ./cmd/arch-view live status --endpoint http://127.0.0.1:PORT/v1/live/SESSION/status
~~~

Status can be `initializing`, `ready`, `degraded`, or `failed`. The response
also says whether the returned revision is `current`, `stale`, `updating`, or
`input_unstable` and includes diagnostics.

| Option | What it does |
| --- | --- |
| **--endpoint** | The status URL printed by `live start`. Required. |

## Wait for a verified revision

Use `wait` when the last ready revision is acceptable. Use
`ensure-current` when the next operation must use the current files.

~~~powershell
go run ./cmd/arch-view live wait --endpoint http://127.0.0.1:PORT/v1/live/SESSION --consistency latest_ready --timeout 10s
go run ./cmd/arch-view live ensure-current --endpoint http://127.0.0.1:PORT/v1/live/SESSION --timeout 10s
~~~

| Option | What it does |
| --- | --- |
| **--endpoint** | The live session endpoint. Required. |
| **--consistency** | `latest_ready` returns the last ready revision; `require_current` waits for verified current input. |
| **--timeout** | Maximum client wait, such as `10s` or `1m`. |

`ensure-current` always requests `require_current`. A timeout, unstable input,
or failed analysis is returned as an explicit error. It is not reported as an
empty successful result.

## Open the live viewer

~~~powershell
go run ./cmd/arch-view open --project . --live --port 0
~~~

`open` keeps its normal one-shot behavior unless `--live` is present. In live
mode the browser shows the session status and refreshes the graph, inspection,
scope, and quality views only after one coherent revision is published.
