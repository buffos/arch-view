# Clojure compatibility canonical API/CLI contract

## Analyzer options

```json
{
  "source_roots": ["path"],
  "platform": "clj | cljs | both",
  "include_tests": false,
  "exclude": ["glob"]
}
```

Source-root precedence is explicit option > supported project config > `src`. Default platform is `both`; reader-conditional observations retain platform metadata.

## Manifest and result

`id=org.archview.clojure`, `language=clojure`, markers=`deps.edn`, `project.clj`, `shadow-cljs.edn`, capabilities=`detect`, `static_dependencies`, `polymorphic_metadata`.

The adapter emits common namespace modules and `depends_on` observations. `defprotocol`/`defmulti` tags are metadata; no `abstract`/`direct` core edge classification is emitted.

## CLI example

`arch-view analyze --project ./repo --language clojure --platform both --format analysis-json --output analysis.json`

Parent analysis status, diagnostics, evidence, safety, and exit-code rules apply.
