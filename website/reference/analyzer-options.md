# Analyzer option reference

An analyzer option changes what the analyzer reads or how it interprets the project.

## How to read an option

Every option has:

- a name;
- a type;
- a default;
- allowed values when limited;
- a language;
- an explanation of what changes in the report.

The option names in this page are the analyzer names. The CLI uses the matching dashed form. For example, source_roots becomes **--source-root**.

## Shared ideas

### Include and exclude

Include options add files that are normally outside the default source set. Exclude options remove paths from the analysis.

If an exclusion and inclusion overlap, the analyzer's documented source policy decides the result. Keep the terminal command so the choice can be repeated.

### Automatic values

An empty or automatic value asks the analyzer to discover the project setting. Use an explicit value when a project has several modules, configs, targets, or runtimes.

## Complete inventory

The generated reference inventory lists every analyzer option currently exposed by the built-in manifests. The documentation check fails if a new option is added without a human page entry.
