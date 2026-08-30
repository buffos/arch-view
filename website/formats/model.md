# Model JSON

The canonical model is the stable input for the viewer and exporter.

Its schema version is **arch-view.model/v1**.

## Main fields

| Field | Meaning |
| --- | --- |
| schema_version | The model format version. |
| model_id | A stable identity for the model content. |
| status | complete, partial, or failed. |
| project | Project boundary and language information. |
| analyzer | Analyzer identity and version. |
| modules | Reported local architecture modules. |
| references | External, standard-library, unresolved, or dynamic references. |
| source_references | File and location references. |
| relationships | Reported dependency relationships. |
| diagnostics | Problems or limitations retained in the model. |
| derived | Cycles, layers, and other graph projections. |
| source_index | Optional source facts and symbols. |
| quality_report | Optional quality evaluation. |

## Complete and partial

**Complete** means the model passed validation without a recoverable diagnostic. **Partial** means the model is usable but carries a recoverable limitation. **Failed** models cannot be exported as normal architecture views.

The viewer should show the status rather than hiding it.
