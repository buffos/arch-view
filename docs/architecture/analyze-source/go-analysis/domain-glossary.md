# Go analysis domain glossary

| Term | Definition |
|---|---|
| Go module | Project boundary declared by one `go.mod`. |
| Go workspace | `go.work` context that may contain multiple modules; not a single analysis target by itself. |
| Go package | Directory-level Go architecture node under the selected module. |
| Import observation | Static import declaration from one package to another target. |
| Build view | Source inclusion determined by default or explicitly selected build tags. |
| Generated file | File identified by the standard generated-file marker and excluded by default. |
| Local import | Import path resolving inside the selected module. |
| Non-local import | Standard-library, third-party, cgo, missing, or conditional target retained as reference/diagnostic. |

Critical distinction: a Go package is not a directory string alone; its identity is derived from the selected module path plus relative import path.
