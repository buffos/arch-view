# Go analyzer

The Go analyzer looks for Go modules and workspaces. It reads Go source statically.

## Project markers

- go.mod
- go.work

## Options

| Option | What it does | Default |
| --- | --- | --- |
| module | Selects one module path or workspace-relative module directory. | automatic |
| build tags | Adds build tags used to choose files. Repeatable. | none |
| include tests | Includes test files. | false |
| include generated | Includes generated files. | false |
| include external | Keeps detail for non-local references. | false |
| exclude | Adds repository-relative exclusion globs. Repeatable. | none |
| safe mode | Disables target-code execution and tool-assisted execution. | true |

Example:

~~~powershell
go run ./cmd/arch-view analyze --project . --language go --include-tests --build-tag linux --output analysis.json
~~~

## Build tags

If a file is selected only under a build tag, the report changes when the tag changes.

Example:

~~~text
build-tag: linux
~~~

Use the same tags as the build you want to understand.
