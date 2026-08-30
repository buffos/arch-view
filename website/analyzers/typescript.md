# TypeScript analyzer

The TypeScript analyzer reads TypeScript projects and can optionally include JavaScript and JSX.

## Project markers

- tsconfig.json
- package.json

## Options

| Option | What it does | Default |
| --- | --- | --- |
| config | Selects a tsconfig file when more than one exists. | automatic |
| include JavaScript | Includes JavaScript and JSX files. | false |
| include tests | Includes test and spec files and directories. | false |
| runtime | Chooses auto, esm, or cjs package-export conditions. | auto |
| exclude | Adds repository-relative exclusion globs. Repeatable. | none |

Example:

~~~powershell
go run ./cmd/arch-view analyze --project . --language typescript --config tsconfig.json --include-js --runtime esm --output analysis.json
~~~

## Why runtime matters

Packages can expose different files to ESM and CommonJS users. Selecting the matching runtime helps the analyzer follow the package's intended entry point.
