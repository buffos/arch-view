# Export and automate canonical use cases

## Application services

### ExportService

- `LoadExportInput`
- `ValidateExportRequest`
- `WriteCanonicalJson`
- `WriteInteractiveHtml`
- `WriteSvg`

### AutomationService

- `RunHeadlessAnalysisAndExport`
- `SummarizeExportStatus`

## Orchestration

1. Load/validate canonical model and optional view projection.
2. Validate format/options/output policy.
3. Normalize deterministic inputs and layout provenance.
4. Render one artifact from the model/view contract.
5. Atomically write it and return metadata/status.

## Outcomes

Complete or partial input can produce a valid artifact. Invalid model, unsupported format/option, render failure, output permission failure, and cancellation are distinct failures. Headless analysis/export may return a partial artifact with exit `0`; fatal errors are non-zero.

## Retry expectations

Export is idempotent for the same input/options/output target; atomic replacement prevents truncated files. It does not mutate the model or source repository.
