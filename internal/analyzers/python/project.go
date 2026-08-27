package pyanalyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

// SourceRoot is an effective, repository-contained Python source root.
type SourceRoot struct {
	Absolute string
	Relative string
}

// Project contains the resolved Python boundary and static discovery policy.
type Project struct {
	Root               string
	Boundary           string
	ConfigurationFiles []string
	SourceRoots        []SourceRoot
	PythonVersion      string
	ConfigDiagnostics  []analysis.Diagnostic
}

// ResolveProject selects the preferred project marker, reads only its data,
// and resolves source roots without loading a Python interpreter.
func ResolveProject(root string, options analysis.EffectiveOptions) (Project, error) {
	absoluteRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return Project{}, analysis.WrapHostError(analysis.ErrInvalidRequest, "Python project root could not be normalized", err, nil)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)
	info, err := os.Stat(absoluteRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return Project{}, analysis.NewHostError(analysis.ErrUnreadableProject, "Python project root does not exist", map[string]any{"project_root": root})
		}
		return Project{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "Python project root could not be read", err, map[string]any{"project_root": root})
	}
	if !info.IsDir() {
		return Project{}, analysis.NewHostError(analysis.ErrInvalidRequest, "Python project root must be a directory", map[string]any{"project_root": root})
	}

	boundary := preferredBoundary(absoluteRoot)
	if boundary == "" {
		return Project{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "Python project requires pyproject.toml, setup.cfg, or setup.py at the selected root", map[string]any{"project_root": root})
	}
	config := readConfiguration(absoluteRoot, boundary)
	diagnostics := append([]analysis.Diagnostic{}, config.Diagnostics...)

	explicitRoots := optionStrings(options, "source_roots")
	configuredRoots := config.SourceRoots
	rootValues := explicitRoots
	rootSource := "explicit analyzer options"
	if len(rootValues) == 0 {
		rootValues = configuredRoots
		rootSource = "project configuration"
	}
	if len(rootValues) == 0 {
		if directoryExists(filepath.Join(absoluteRoot, "src")) {
			rootValues = []string{"src"}
			rootSource = "src layout default"
		} else {
			rootValues = []string{"."}
			rootSource = "project root default"
		}
	}

	roots := make([]SourceRoot, 0, len(rootValues))
	for _, value := range rootValues {
		sourceRoot, rootErr := normalizeSourceRoot(absoluteRoot, value)
		if rootErr != nil {
			diagnostics = append(diagnostics, sourceRootDiagnostic(value, rootErr))
			continue
		}
		if !containsSourceRoot(roots, sourceRoot.Absolute) {
			roots = append(roots, sourceRoot)
		}
	}
	if len(roots) == 0 {
		fallback, fallbackErr := normalizeSourceRoot(absoluteRoot, ".")
		if fallbackErr == nil {
			roots = append(roots, fallback)
			diagnostics = append(diagnostics, analysis.Diagnostic{
				Code:        "python_source_root_fallback",
				Severity:    "warning",
				Message:     fmt.Sprintf("No usable source roots were found from %s; the project root was used as a safe fallback.", rootSource),
				Recoverable: true,
			})
		}
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Relative < roots[j].Relative })

	pythonVersion := optionString(options, "python_version")
	if pythonVersion == "" {
		pythonVersion = config.PythonVersion
	}
	if pythonVersion != "" && !supportedPythonVersion(pythonVersion) {
		diagnostics = append(diagnostics, analysis.Diagnostic{
			Code:        "python_unsupported_version",
			Severity:    "warning",
			Message:     fmt.Sprintf("Python version %q is not a supported static version selector; discovery continued without executing Python.", pythonVersion),
			Subject:     pythonVersion,
			Recoverable: true,
		})
		pythonVersion = ""
	}

	return Project{
		Root:               absoluteRoot,
		Boundary:           boundary,
		ConfigurationFiles: []string{boundary},
		SourceRoots:        roots,
		PythonVersion:      pythonVersion,
		ConfigDiagnostics:  diagnostics,
	}, nil
}

func preferredBoundary(root string) string {
	for _, marker := range []string{"pyproject.toml", "setup.cfg", "setup.py"} {
		if safeProjectFile(root, marker) {
			return marker
		}
	}
	return ""
}

func safeProjectFile(root, relative string) bool {
	filePath := filepath.Join(root, filepath.FromSlash(relative))
	return existsAsFile(filePath) && resolvedPathWithin(root, filePath)
}

func normalizeSourceRoot(root, value string) (SourceRoot, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return SourceRoot{}, fmt.Errorf("source root is empty")
	}
	absolute := trimmed
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(root, absolute)
	}
	absolute = filepath.Clean(absolute)
	if !pathWithin(root, absolute) {
		return SourceRoot{}, fmt.Errorf("source root escapes the selected project root")
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return SourceRoot{}, fmt.Errorf("source root could not be read: %w", err)
	}
	if !info.IsDir() {
		return SourceRoot{}, fmt.Errorf("source root is not a directory")
	}
	if !resolvedPathWithin(root, absolute) {
		return SourceRoot{}, fmt.Errorf("source root resolves outside the selected project root")
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return SourceRoot{}, fmt.Errorf("source root could not be made repository-relative: %w", err)
	}
	if relative == "." {
		return SourceRoot{Absolute: absolute, Relative: "."}, nil
	}
	return SourceRoot{Absolute: absolute, Relative: filepath.ToSlash(relative)}, nil
}

func resolvedPathWithin(root, candidate string) bool {
	resolvedRoot, rootErr := filepath.EvalSymlinks(root)
	resolvedCandidate, candidateErr := filepath.EvalSymlinks(candidate)
	if rootErr != nil || candidateErr != nil {
		return false
	}
	return pathWithin(filepath.Clean(resolvedRoot), filepath.Clean(resolvedCandidate))
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func sourceRootDiagnostic(value string, err error) analysis.Diagnostic {
	return analysis.Diagnostic{
		Code:        "python_source_root_invalid",
		Severity:    "warning",
		Message:     fmt.Sprintf("Python source root %q was ignored: %v", value, err),
		Subject:     value,
		Recoverable: true,
	}
}

func containsSourceRoot(roots []SourceRoot, absolute string) bool {
	for _, root := range roots {
		if root.Absolute == absolute {
			return true
		}
	}
	return false
}

func supportedPythonVersion(value string) bool {
	return pythonVersionSelectorPattern.MatchString(strings.TrimSpace(value))
}

func optionString(options analysis.EffectiveOptions, name string) string {
	value, ok := options.Values[name]
	if !ok || value == nil {
		return ""
	}
	typed, _ := value.(string)
	return strings.TrimSpace(typed)
}

func optionStrings(options analysis.EffectiveOptions, name string) []string {
	value, _ := options.Values[name].([]string)
	return append([]string(nil), value...)
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
