package goanalyzer

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

type Project struct {
	Root                  string
	ModuleRoot            string
	ModulePath            string
	WorkspacePath         string
	RelativeModuleRoot    string
	RelativeWorkspacePath string
	Boundary              string
}

type moduleCandidate struct {
	Root       string
	ModulePath string
}

func ResolveProject(root string, options analysis.EffectiveOptions) (Project, error) {
	root = filepath.Clean(root)
	workspacePath := filepath.Join(root, "go.work")
	if existsAsFile(workspacePath) {
		candidates, err := workspaceModules(root, workspacePath)
		if err != nil {
			return Project{}, err
		}
		if len(candidates) == 0 {
			if existsAsFile(filepath.Join(root, "go.mod")) {
				candidate, err := readModule(filepath.Join(root, "go.mod"))
				if err != nil {
					return Project{}, err
				}
				candidates = []moduleCandidate{candidate}
			} else {
				return Project{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "go.work does not declare a readable module", map[string]any{"project_root": root})
			}
		}
		selected, err := chooseModule(candidates, optionString(options, "module"), root)
		if err != nil {
			return Project{}, err
		}
		return makeProject(root, selected, workspacePath), nil
	}

	modulePath := filepath.Join(root, "go.mod")
	if !existsAsFile(modulePath) {
		return Project{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "Go project requires go.mod or go.work at the selected root", map[string]any{"project_root": root})
	}
	candidate, err := readModule(modulePath)
	if err != nil {
		return Project{}, err
	}
	if selector := optionString(options, "module"); selector != "" && !moduleMatches(candidate, selector, root) {
		return Project{}, analysis.NewHostError(analysis.ErrModuleSelection, "requested module does not match the selected go.mod", map[string]any{"module": selector, "module_path": candidate.ModulePath})
	}
	return makeProject(root, candidate, ""), nil
}

func workspaceModules(root, workspacePath string) ([]moduleCandidate, error) {
	content, err := os.ReadFile(workspacePath)
	if err != nil {
		return nil, analysis.WrapHostError(analysis.ErrUnreadableProject, "go.work could not be read", err, map[string]any{"path": workspacePath})
	}
	uses := parseUseDirectives(string(content))
	candidates := make([]moduleCandidate, 0, len(uses))
	seen := make(map[string]struct{}, len(uses))
	for _, usePath := range uses {
		moduleRoot := usePath
		if !filepath.IsAbs(moduleRoot) {
			moduleRoot = filepath.Join(root, moduleRoot)
		}
		moduleRoot, err = filepath.Abs(filepath.Clean(moduleRoot))
		if err != nil {
			return nil, analysis.WrapHostError(analysis.ErrUnsupportedProject, "workspace module path could not be normalized", err, map[string]any{"module": usePath})
		}
		if _, exists := seen[moduleRoot]; exists {
			continue
		}
		seen[moduleRoot] = struct{}{}
		if !pathWithin(root, moduleRoot) {
			return nil, analysis.NewHostError(analysis.ErrUnsupportedProject, "go.work references a module outside the selected project root", map[string]any{"module": usePath})
		}
		info, statErr := os.Stat(moduleRoot)
		if statErr != nil || !info.IsDir() {
			return nil, analysis.NewHostError(analysis.ErrUnreadableProject, "go.work references a missing module directory", map[string]any{"module": usePath})
		}
		candidate, readErr := readModule(filepath.Join(moduleRoot, "go.mod"))
		if readErr != nil {
			return nil, readErr
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func parseUseDirectives(content string) []string {
	var uses []string
	inBlock := false
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := stripLineComment(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			if token := firstToken(line); token != "" {
				uses = append(uses, token)
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "use" {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
		if rest == "(" {
			inBlock = true
			continue
		}
		if strings.HasPrefix(rest, "(") {
			inBlock = true
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "("))
		}
		if token := firstToken(rest); token != "" && token != ")" {
			uses = append(uses, token)
		}
	}
	return uses
}

func firstToken(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], " \t\"'()")
}

func stripLineComment(line string) string {
	if index := strings.Index(line, "//"); index >= 0 {
		return strings.TrimSpace(line[:index])
	}
	return line
}

func readModule(path string) (moduleCandidate, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return moduleCandidate{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "go.mod could not be found", map[string]any{"path": path})
		}
		return moduleCandidate{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "go.mod could not be read", err, map[string]any{"path": path})
	}
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := stripLineComment(strings.TrimSpace(scanner.Text()))
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return moduleCandidate{Root: filepath.Dir(path), ModulePath: fields[1]}, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return moduleCandidate{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "go.mod could not be scanned", err, map[string]any{"path": path})
	}
	return moduleCandidate{}, analysis.NewHostError(analysis.ErrUnsupportedProject, "go.mod does not declare a module path", map[string]any{"path": path})
}

func chooseModule(candidates []moduleCandidate, selector, workspaceRoot string) (moduleCandidate, error) {
	if selector != "" {
		for _, candidate := range candidates {
			if moduleMatches(candidate, selector, workspaceRoot) {
				return candidate, nil
			}
		}
		return moduleCandidate{}, analysis.NewHostError(analysis.ErrModuleSelection, "requested module was not found in go.work", map[string]any{"module": selector})
	}
	if len(candidates) != 1 {
		return moduleCandidate{}, analysis.NewHostError(analysis.ErrModuleSelection, "go.work exposes multiple modules; an explicit module selection is required", map[string]any{"module_count": len(candidates)})
	}
	return candidates[0], nil
}

func moduleMatches(candidate moduleCandidate, selector, workspaceRoot string) bool {
	if selector == candidate.ModulePath {
		return true
	}
	cleanSelector := selector
	if !filepath.IsAbs(cleanSelector) {
		cleanSelector = filepath.Join(workspaceRoot, cleanSelector)
	}
	cleanSelector, err := filepath.Abs(filepath.Clean(cleanSelector))
	if err != nil {
		return false
	}
	candidateRoot, err := filepath.Abs(filepath.Clean(candidate.Root))
	return err == nil && candidateRoot == cleanSelector
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func makeProject(root string, candidate moduleCandidate, workspacePath string) Project {
	relativeModuleRoot, _ := filepath.Rel(root, candidate.Root)
	if relativeModuleRoot == "." {
		relativeModuleRoot = "."
	}
	relativeWorkspacePath := ""
	boundary := "go.mod"
	if workspacePath != "" {
		relativeWorkspacePath, _ = filepath.Rel(root, workspacePath)
		boundary = "go.work"
	}
	return Project{
		Root:                  root,
		ModuleRoot:            candidate.Root,
		ModulePath:            candidate.ModulePath,
		WorkspacePath:         workspacePath,
		RelativeModuleRoot:    filepath.ToSlash(relativeModuleRoot),
		RelativeWorkspacePath: filepath.ToSlash(relativeWorkspacePath),
		Boundary:              boundary,
	}
}

func optionString(options analysis.EffectiveOptions, name string) string {
	value, ok := options.Values[name]
	if !ok || value == nil {
		return ""
	}
	typed, _ := value.(string)
	return typed
}
