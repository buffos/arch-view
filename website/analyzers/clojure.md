# Clojure analyzer

The Clojure analyzer reads Clojure source and namespaces. It supports reader-conditional views.

## Options

| Option | What it does | Default |
| --- | --- | --- |
| source roots | Selects repository-relative Clojure source roots. Repeatable. | automatic |
| platform | Chooses clj, cljs, or both. | both |
| include tests | Includes Clojure test files and test directories. | false |
| exclude | Adds repository-relative exclusion globs. Repeatable. | none |

Example:

~~~powershell
go run ./cmd/arch-view analyze --project . --language clojure --platform cljs --output analysis.json
~~~

If a file has different branches for clj and cljs, the selected platform decides which branch is included.
