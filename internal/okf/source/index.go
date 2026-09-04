package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/markdown"
)

func (scanner *FilesystemScanner) buildIndex(ctx context.Context, candidate domain.BundleCandidate) (domain.BundleIndex, []domain.Diagnostic) {
	index := domain.BundleIndex{BundleID: candidate.BundleID, Root: candidate.AbsolutePath, Documents: make(map[string]domain.ConceptDocument)}
	diagnostics := make([]domain.Diagnostic, 0)
	limits := scanner.Limits.resolved()
	var sourceBytes int64
	limitExceeded := false
	hash := sha256.New()
	resolvedBundleRoot, resolveErr := filepath.EvalSymlinks(candidate.AbsolutePath)
	if resolveErr != nil {
		return index, []domain.Diagnostic{{Code: "okf_bundle_unreadable", Severity: "error", Category: "source", Message: resolveErr.Error(), BundleID: candidate.BundleID, Path: candidate.RelativePath}}
	}
	resolvedBundleRoot = filepath.Clean(resolvedBundleRoot)
	paths := make([]string, 0)
	err := filepath.WalkDir(candidate.AbsolutePath, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			diagnostics = appendDiagnostic(diagnostics, domain.Diagnostic{Code: "okf_bundle_unreadable", Severity: "error", Category: "source", Message: walkErr.Error(), BundleID: candidate.BundleID, Path: filePath})
			return nil
		}
		if err := contextErr(ctx); err != nil {
			return err
		}
		if filePath != candidate.AbsolutePath && entry.IsDir() && scanner.excluded(entry.Name()) {
			return fs.SkipDir
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}
		if len(paths) >= limits.MaxFiles {
			diagnostics = appendDiagnostic(diagnostics, sourceLimitDiagnostic(candidate.BundleID, "files", int64(limits.MaxFiles)))
			limitExceeded = true
			return fs.SkipAll
		}
		paths = append(paths, filepath.ToSlash(filePath))
		return nil
	})
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		diagnostics = appendDiagnostic(diagnostics, domain.Diagnostic{Code: "okf_bundle_unreadable", Severity: "error", Category: "source", Message: err.Error(), BundleID: candidate.BundleID})
	}
	if limitExceeded {
		return index, boundDiagnostics(diagnostics)
	}
	sort.Strings(paths)
	readSource := func(filePath string) error {
		relative, err := filepath.Rel(candidate.AbsolutePath, filePath)
		if err != nil {
			return nil
		}
		relative = filepath.ToSlash(relative)
		resolvedFilePath, resolveErr := filepath.EvalSymlinks(filePath)
		if resolveErr != nil {
			diagnostics = appendDiagnostic(diagnostics, domain.Diagnostic{Code: "okf_bundle_unreadable", Severity: "error", Category: "source", Message: resolveErr.Error(), BundleID: candidate.BundleID, Path: relative})
			return nil
		}
		if !withinPath(resolvedBundleRoot, resolvedFilePath) {
			diagnostics = appendDiagnostic(diagnostics, domain.Diagnostic{Code: "okf_bundle_boundary_violation", Severity: "error", Category: "security", Message: "A source file resolves outside the selected bundle boundary.", BundleID: candidate.BundleID, Path: relative, Recovery: "Remove the out-of-bound symlink or keep the target inside the bundle."})
			return nil
		}
		data, err := readConceptFile(resolvedFilePath)
		if err != nil {
			diagnostics = appendDiagnostic(diagnostics, domain.Diagnostic{Code: "okf_bundle_unreadable", Severity: "error", Category: "source", Message: err.Error(), BundleID: candidate.BundleID, Path: relative})
			return nil
		}
		if int64(len(data)) > limits.MaxBytes-sourceBytes {
			diagnostics = appendDiagnostic(diagnostics, sourceLimitDiagnostic(candidate.BundleID, "bytes", limits.MaxBytes))
			return fs.SkipAll
		}
		sourceBytes += int64(len(data))
		if len(data) > maxConceptBytes {
			diagnostics = appendDiagnostic(diagnostics, domain.Diagnostic{Code: "okf_concept_too_large", Severity: "error", Category: "scale", Message: "Concept exceeds the source document size limit.", BundleID: candidate.BundleID, Path: relative, Details: map[string]any{"max_bytes": maxConceptBytes}})
			return nil
		}
		_, _ = hash.Write([]byte(relative))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(data)
		_, _ = hash.Write([]byte{0})
		base := strings.ToLower(filepath.Base(relative))
		if base == "index.md" || base == "log.md" {
			return nil
		}
		document, err := parseConcept(relative, data)
		if err != nil {
			diagnostics = appendDiagnostic(diagnostics, domain.Diagnostic{Code: "okf_bundle_invalid", Severity: "error", Category: "validation", Message: err.Error(), BundleID: candidate.BundleID, Path: relative, Recovery: "Add valid YAML frontmatter with a non-empty type field."})
			return nil
		}
		index.Documents[document.ConceptID] = document
		return nil
	}
	for _, pathValue := range paths {
		if contextErr(ctx) != nil {
			return index, boundDiagnostics(diagnostics)
		}
		if readSource(filepath.FromSlash(pathValue)) == fs.SkipAll {
			index.Documents = nil
			return index, boundDiagnostics(diagnostics)
		}
	}
	if contextErr(ctx) != nil {
		return index, boundDiagnostics(diagnostics)
	}
	index.SourceRevision = "sha256:" + hex.EncodeToString(hash.Sum(nil))
	index.ConceptOrder = make([]string, 0, len(index.Documents))
	for conceptID := range index.Documents {
		index.ConceptOrder = append(index.ConceptOrder, conceptID)
	}
	sort.Strings(index.ConceptOrder)
	resolveLinks(ctx, &index, &diagnostics)
	return index, boundDiagnostics(diagnostics)
}

func withinPath(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

// Bound the read itself, not just the size of an already allocated document.
func readConceptFile(filePath string) ([]byte, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("concept source must be a regular file")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, maxConceptBytes+1))
}

func resolveLinks(ctx context.Context, index *domain.BundleIndex, diagnostics *[]domain.Diagnostic) {
	for _, conceptID := range index.ConceptOrder {
		if contextErr(ctx) != nil {
			return
		}
		document := index.Documents[conceptID]
		for linkIndex, original := range document.Links {
			if contextErr(ctx) != nil {
				return
			}
			original.ID = fmt.Sprintf("%s#link-%d", conceptID, linkIndex+1)
			link, diagnostic := markdown.Resolve(document, *index, original)
			document.Links[linkIndex] = link
			if diagnostic != nil {
				*diagnostics = appendDiagnostic(*diagnostics, *diagnostic)
			}
			if link.Safe && link.Resolved && link.Kind == "local" {
				index.Relationships = append(index.Relationships, domain.Relationship{RelationshipID: "semantic:" + conceptID + ":" + link.TargetID + ":" + fmt.Sprint(linkIndex), Kind: domain.RelationshipSemantic, From: conceptID, To: link.TargetID, Provenance: []domain.Provenance{{Source: "markdown_link", Path: document.SourcePath, Explanation: link.RawTarget}}})
			}
		}
		index.Documents[conceptID] = document
	}
}
