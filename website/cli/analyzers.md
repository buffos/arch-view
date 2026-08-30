# analyzers

The analyzers command lists the analyzers that Arch View can use.

~~~powershell
go run ./cmd/arch-view analyzers
~~~

The result is JSON so scripts can inspect it.

## Options

| Option | What it does |
| --- | --- |
| **--analyzer-runtime** | Chooses auto, packaged, in-process, or explicit analyzer execution. |
| **--plugin** | Adds an external analyzer descriptor. Repeat it for multiple descriptors. |
| **--allow-untrusted-plugin** | Allows an explicitly supplied local descriptor. Use it only for a trusted file. |

## Why list analyzers?

Use this command when automatic selection is surprising or when a packaged analyzer is not available.

Example:

~~~powershell
go run ./cmd/arch-view analyzers --analyzer-runtime packaged
~~~

The output includes the analyzer ID, language, version, capabilities, and options. See the [analyzer overview](/analyzers/overview) for how those fields affect analysis.
