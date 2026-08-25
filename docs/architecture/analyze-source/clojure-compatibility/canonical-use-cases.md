# Clojure compatibility canonical use cases

## ClojureAnalyzerService

- `DetectClojureProject`
- `ResolveClojureSourceRoots`
- `ParseNamespaceForms`
- `ExtractStaticNamespaceDependencies`
- `ExtractPolymorphicMetadata`
- `EmitClojureAnalysisResult`

## Orchestration

Resolve source roots/config → enumerate Clojure-family files → parse namespace forms → extract static dependency clauses and platform metadata → tag recognized polymorphic forms → emit common result.

Failures: missing config/root, malformed form, missing `ns`, unsupported reader conditional, unresolved namespace, dynamic loading. Recoverable cases produce partial status.

The adapter must never call `require`, evaluate forms, or load the target classpath.
