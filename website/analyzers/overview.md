# Analyzers

An analyzer reads one kind of project. It detects the project, extracts the facts it understands, and reports its limits.

## Automatic selection

The normal command is:

~~~powershell
go run ./cmd/arch-view analyze --project . --output analysis.json
~~~

Arch View looks for project markers such as go.mod, pyproject.toml, Cargo.toml, or tsconfig.json.

## Explicit selection

Use an explicit language when a project contains several possible markers:

~~~powershell
go run ./cmd/arch-view analyze --project . --language python --output analysis.json
~~~

Use an exact analyzer when you need one registered implementation:

~~~powershell
go run ./cmd/arch-view analyze --project . --analyzer org.archview.go --output analysis.json
~~~

## Runtime modes

| Mode | Meaning |
| --- | --- |
| auto | Choose the available analyzer source automatically. |
| in-process | Run the built-in analyzer in the Arch View process. |
| packaged | Use a packaged analyzer when one is installed. |
| explicit | Use a descriptor supplied with the command. |

## What an analyzer reports

An analyzer can report modules, dependencies, source files, symbols, documentation, metrics, and diagnostics. Each language supports a different set.

That is why one quality rule may be observed for Go and unsupported for another language. Read the capability and coverage message instead of guessing.

## List available analyzers

~~~powershell
go run ./cmd/arch-view analyzers
~~~

The JSON output contains the analyzer's language, version, capabilities, and option descriptors.
