package imports

import (
	"go/build"
	"path/filepath"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/goanalyzer/scanner"
)

type Target struct {
	Scope        string
	TargetScope  string
	ModuleID     string
	ReferenceID  string
	Reference    analysis.Reference
	HasReference bool
}

func ResolveImportTarget(project scanner.Project, packages map[string]*scanner.Package, importPath string, includeExternal bool) Target {
	if pkg, ok := packages[importPath]; ok {
		return Target{
			Scope:       "local",
			TargetScope: "local",
			ModuleID:    scanner.PackageID(project.ModulePath, pkg.RelativeDir),
		}
	}
	if importPath == "C" {
		return referenceTarget(importPath, "external", "cgo", includeExternal)
	}
	if strings.HasPrefix(importPath, project.ModulePath+"/") || importPath == project.ModulePath {
		relative := strings.TrimPrefix(importPath, project.ModulePath)
		relative = strings.TrimPrefix(relative, "/")
		directory := project.ModuleRoot
		if relative != "" {
			directory = filepath.Join(project.ModuleRoot, filepath.FromSlash(relative))
		}
		if scanner.HasGoFile(directory) {
			return referenceTarget(importPath, "unresolved", "conditional", includeExternal)
		}
		return referenceTarget(importPath, "unresolved", "unresolved", includeExternal)
	}
	if isStandardLibraryImport(importPath) {
		return referenceTarget(importPath, "standard_library", "standard_library", includeExternal)
	}
	return referenceTarget(importPath, "external", "external", includeExternal)
}

func referenceTarget(importPath, scope, targetScope string, includeExternal bool) Target {
	metadata := map[string]any{
		"import_path":  importPath,
		"target_scope": targetScope,
	}
	if scope == "external" {
		if includeExternal {
			metadata["detail"] = "full"
		} else {
			metadata["detail"] = "summary"
		}
	}
	reference := analysis.Reference{
		ID:       scanner.StableID("reference", scope, importPath),
		Name:     importPath,
		Scope:    scope,
		Language: "go",
		Metadata: metadata,
	}
	return Target{
		Scope:        scope,
		TargetScope:  targetScope,
		ReferenceID:  reference.ID,
		Reference:    reference,
		HasReference: true,
	}
}

func ConfidenceFor(targetScope string) *analysis.Confidence {
	switch targetScope {
	case "local", "standard_library", "external":
		return &analysis.Confidence{Basis: "resolved", Score: 1}
	case "conditional":
		return &analysis.Confidence{Basis: "inferred", Score: 0.5}
	case "cgo":
		return &analysis.Confidence{Basis: "dynamic", Score: 0.4}
	default:
		return &analysis.Confidence{Basis: "unresolved", Score: 0.2}
	}
}

func DiagnosticCodeFor(targetScope string) string {
	if targetScope == "cgo" {
		return "go_cgo_import"
	}
	return "go_unresolved_import"
}

func isStandardLibraryImport(importPath string) bool {
	first := importPath
	if slash := strings.IndexByte(first, '/'); slash >= 0 {
		first = first[:slash]
	}
	if first == "" || strings.Contains(first, ".") || first == "internal" || first == "vendor" {
		return false
	}
	standardRoot := filepath.Clean(build.Default.GOROOT)
	if standardRoot == "." || standardRoot == "" {
		return false
	}
	imported, err := build.Default.Import(importPath, "", build.FindOnly)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(standardRoot, imported.Dir)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
