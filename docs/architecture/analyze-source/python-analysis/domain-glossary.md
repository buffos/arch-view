# Python analysis domain glossary

| Term | Definition |
|---|---|
| Python project | One selected root with effective source roots and configuration. |
| Package | Directory/package architecture unit, including a namespace package when statically identifiable. |
| Module | One Python source/stub file represented under package hierarchy. |
| Absolute import | Import resolved from an effective source root/package namespace. |
| Relative import | Import resolved from the importing package's hierarchy. |
| Dynamic import | Import whose target cannot be proven statically, retained as diagnostic/reference. |
| Source root | Directory from which package/module hierarchy is interpreted. |
| Stub | `.pyi` source representation, opt-in by default. |

Package and module are distinct nodes when both have architectural meaning; a file is always evidence, not automatically a separate node.
