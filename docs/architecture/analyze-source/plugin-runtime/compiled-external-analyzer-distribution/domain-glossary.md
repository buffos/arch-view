# Compiled external analyzer distribution domain glossary

| Term | Category | Definition | Distinction |
|---|---|---|---|
| Logical analyzer ID | business object | Stable identifier for an analyzer's semantics, such as `org.archview.python`. | It remains the same across in-process and compiled external deployments. |
| Analyzer package | business object | One platform-specific executable, descriptor, and integrity metadata distributed with Arch View. | It is a deployment unit, not a language implementation or analysis result. |
| Analyzer index | business object | Application-owned manifest listing the packages available to the current installation. | It is the automatic discovery source; a target repository is not an index. |
| Package descriptor | business object | Manifest describing one packaged executable, its logical analyzer, platform, and digests. | It is distinct from `.archview.json` project configuration. |
| Platform target | value object | Normalized operating-system and CPU pair used to select one package, such as `windows-amd64`. | It is not the analyzer language or host project platform. |
| Integrity digest | value object | SHA-256 digest expected for a packaged executable or descriptor. | A digest verifies bytes; it does not grant permission to discover arbitrary files. |
| Trusted package | policy term | Package whose path, index entry, manifest, and digests pass application verification. | An explicitly supplied development descriptor is an override, not a trusted packaged entry. |
| Runtime mode | policy term | Selection of `packaged`, `in-process`, or `explicit` execution for an analyzer job. | Packaged is the production default; fallback is never implicit. |
| Distribution assembly | workflow/action | Reproducible build operation that compiles analyzers and writes the expected application tree. | It is separate from runtime discovery and analysis execution. |
| Package compatibility | policy term | Agreement among host-supported API major, package descriptor, hello manifest, language, and logical ID. | Analyzer semantic version may advance within the supported compatibility policy. |
| Explicit developer override | workflow/action | Deliberate use of a local descriptor or in-process runtime for development, parity, or migration. | It must be visible in invocation/configuration and is never auto-selected from a repository. |

Canonical statuses: `available`, `selected`, `verified`, `unavailable`, `rejected`, `launched`, `failed`.
