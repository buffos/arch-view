# Rust analyzer

The Rust analyzer reads Cargo workspaces and packages.

## Project marker

- Cargo.toml

## Options

| Option | What it does | Default |
| --- | --- | --- |
| crate | Selects one Cargo package or workspace-relative crate directory. | automatic |
| feature | Enables a Cargo feature. Repeatable. | none |
| target | Selects a Rust target triple or target selector. | automatic |
| include tests | Includes test targets and cfg(test) modules. | false |
| include examples | Includes examples and benches. | false |
| exclude | Adds repository-relative exclusion globs. Repeatable. | none |

Example:

~~~powershell
go run ./cmd/arch-view analyze --project . --language rust --crate engine --feature server --target x86_64-unknown-linux-gnu --output analysis.json
~~~

## Features and targets

Features and targets can change which modules exist. Use the same selection as the build you want to inspect.
