package source

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
)

const (
	maxConceptBytes = 8 << 20
	maxDiagnostics  = 200
)

var defaultExclusions = map[string]bool{
	".git": true, ".hg": true, ".svn": true, "node_modules": true,
	"vendor": true, "dist": true, "build": true, "generated": true,
	"coverage": true, "cache": true, ".cache": true, "target": true,
	"bin": true, "out": true, "tmp": true, ".venv": true, "__pycache__": true,
	".next": true, ".nuxt": true, ".terraform": true,
}

type FilesystemScanner struct {
	Exclusions map[string]bool
	Limits     IndexLimits
}

func NewFilesystemScanner() *FilesystemScanner {
	copyExclusions := make(map[string]bool, len(defaultExclusions))
	for key, value := range defaultExclusions {
		copyExclusions[key] = value
	}
	return &FilesystemScanner{Exclusions: copyExclusions}
}

func (scanner *FilesystemScanner) Scan(ctx context.Context, projectRoot string) ([]domain.BundleCandidate, []domain.Diagnostic) {
	root, err := normalizeRoot(projectRoot)
	if err != nil {
		return nil, []domain.Diagnostic{{Code: "okf_project_not_found", Severity: "error", Category: "discovery", Message: err.Error(), Recovery: "Choose an accessible project directory."}}
	}
	candidates := make([]domain.BundleCandidate, 0)
	diagnostics := make([]domain.Diagnostic, 0)
	err = filepath.WalkDir(root, func(pathValue string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			diagnostics = appendDiagnostic(diagnostics, domain.Diagnostic{Code: "okf_bundle_unavailable", Severity: "error", Category: "discovery", Message: walkErr.Error(), Path: pathValue})
			return nil
		}
		if err := contextErr(ctx); err != nil {
			return err
		}
		if pathValue != root && entry.IsDir() && scanner.excluded(entry.Name()) {
			return fs.SkipDir
		}
		if !entry.IsDir() || !strings.EqualFold(entry.Name(), ".okf") {
			return nil
		}
		candidate := domain.BundleCandidate{Status: domain.BundleDiscovered, AbsolutePath: pathValue}
		candidate.BundleID, candidate.RelativePath = relativeBundleID(root, pathValue)
		validated, candidateDiagnostics := scanner.inspect(ctx, candidate)
		validated.Diagnostics = candidateDiagnostics
		candidates = append(candidates, validated)
		return fs.SkipDir
	})
	if contextError := contextErr(ctx); contextError != nil {
		code, message := "okf_operation_cancelled", "bundle discovery was cancelled"
		if contextError == context.DeadlineExceeded {
			code, message = "okf_operation_timeout", "bundle discovery exceeded its processing deadline"
		}
		diagnostics = appendDiagnostic(diagnostics, domain.Diagnostic{Code: code, Severity: "error", Category: "cancellation", Message: message, Recovery: "Retry the catalog refresh."})
	}
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		diagnostics = appendDiagnostic(diagnostics, domain.Diagnostic{Code: "okf_discovery_failed", Severity: "error", Category: "discovery", Message: err.Error()})
	}
	sort.SliceStable(candidates, func(left, right int) bool { return candidates[left].BundleID < candidates[right].BundleID })
	return candidates, boundDiagnostics(diagnostics)
}

func (scanner *FilesystemScanner) Validate(ctx context.Context, candidate domain.BundleCandidate) (domain.BundleCandidate, []domain.Diagnostic) {
	validated, diagnostics := scanner.inspect(ctx, candidate)
	validated.Diagnostics = diagnostics
	return validated, diagnostics
}

func (scanner *FilesystemScanner) Index(ctx context.Context, candidate domain.BundleCandidate) (domain.BundleIndex, error) {
	if !candidate.Selectable || candidate.AbsolutePath == "" {
		return domain.BundleIndex{}, domain.NewError("okf_bundle_invalid", 422, "only a valid bundle can be indexed", map[string]any{"bundle_id": candidate.BundleID})
	}
	index, diagnostics := scanner.buildIndex(ctx, candidate)
	index.Diagnostics = boundDiagnostics(diagnostics)
	if err := contextErr(ctx); err != nil {
		return index, domain.ContextOperationError(err, "bundle indexing").WithDiagnostics(diagnostics...)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			return index, domain.NewError("okf_bundle_invalid", 422, "bundle changed or became invalid while indexing", map[string]any{"bundle_id": candidate.BundleID}).WithDiagnostics(diagnostics...)
		}
	}
	return index, nil
}

func (scanner *FilesystemScanner) inspect(ctx context.Context, candidate domain.BundleCandidate) (domain.BundleCandidate, []domain.Diagnostic) {
	if _, err := os.Stat(candidate.AbsolutePath); err != nil {
		candidate.Status = domain.BundleUnavailable
		candidate.Selectable = false
		return candidate, []domain.Diagnostic{{Code: "okf_bundle_unavailable", Severity: "error", Category: "discovery", Message: err.Error(), BundleID: candidate.BundleID, Path: candidate.RelativePath, Recovery: "Restore the bundle directory or refresh the catalog."}}
	}
	index, diagnostics := scanner.buildIndex(ctx, candidate)
	if err := contextErr(ctx); err != nil {
		candidate.Status = domain.BundleUnavailable
		candidate.Selectable = false
		code := "okf_operation_cancelled"
		if err == context.DeadlineExceeded {
			code = "okf_operation_timeout"
		}
		return candidate, append(diagnostics, domain.Diagnostic{Code: code, Severity: "error", Category: "cancellation", Message: "bundle validation did not complete", BundleID: candidate.BundleID})
	}
	candidate.ConceptCount = len(index.ConceptOrder)
	candidate.SourceRevision = index.SourceRevision
	candidate.Selectable = true
	candidate.Status = domain.BundleValid
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			candidate.Selectable = false
			candidate.Status = statusForDiagnostic(diagnostic.Code)
			break
		}
	}
	return candidate, boundDiagnostics(diagnostics)
}

func statusForDiagnostic(code string) string {
	switch code {
	case "okf_bundle_unavailable":
		return domain.BundleUnavailable
	case "okf_bundle_unreadable":
		return domain.BundleUnreadable
	default:
		return domain.BundleInvalid
	}
}

func (scanner *FilesystemScanner) excluded(name string) bool {
	return scanner.Exclusions[strings.ToLower(name)]
}

func normalizeRoot(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("project root is required")
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project root is not a directory")
	}
	return filepath.Clean(absolute), nil
}

func relativeBundleID(root, bundleRoot string) (string, string) {
	relative, err := filepath.Rel(root, bundleRoot)
	if err != nil || relative == "." {
		relative = filepath.Base(bundleRoot)
	}
	value := filepath.ToSlash(filepath.Clean(relative))
	return value, value
}

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func appendDiagnostic(values []domain.Diagnostic, value domain.Diagnostic) []domain.Diagnostic {
	if len(values) < maxDiagnostics {
		return append(values, value)
	}
	last := maxDiagnostics - 1
	// Preserve an error even when it is the entry displaced by the notice.
	retainedError := -1
	for index, diagnostic := range values[:last] {
		if diagnostic.Severity == "error" {
			retainedError = index
			break
		}
	}
	if retainedError < 0 {
		if values[last].Severity == "error" {
			values[last-1] = values[last]
		} else if value.Severity == "error" {
			values[last-1] = value
		}
	}
	values[last] = domain.Diagnostic{Code: "okf_diagnostics_truncated", Severity: "warning", Category: "scale", Message: "Additional diagnostics were omitted at the catalog limit."}
	return values
}

func boundDiagnostics(values []domain.Diagnostic) []domain.Diagnostic {
	if len(values) <= maxDiagnostics {
		return append([]domain.Diagnostic(nil), values...)
	}
	result := make([]domain.Diagnostic, 0, maxDiagnostics)
	for _, value := range values {
		result = appendDiagnostic(result, value)
	}
	return result
}
