package rustanalyzer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
	rustsyntax "github.com/buffo/arch-view/internal/analysis/syntax/rust"
)

type discoveryResult struct {
	Modules          map[string]*rustModule
	Uses             []rustUseObservation
	SourceReferences map[string]analysis.SourceReference
	Diagnostics      []analysis.Diagnostic
	Visited          map[string]struct{}
}

type rustModule struct {
	ID                 string
	Path               []string
	Kind               string
	Name               string
	FilePath           string
	ModuleDir          string
	SourceReferenceIDs map[string]struct{}
	Tags               map[string]struct{}
	CfgConditions      map[string]struct{}
	DeclaredModules    map[string]struct{}
	Paths              map[string]struct{}
	TargetKinds        map[string]struct{}
	MacroAttributes    map[string]struct{}
	Generated          bool
}

type rustUseObservation struct {
	FromModuleID  string
	Kind          string
	Path          string
	Alias         string
	ImportedName  string
	CfgConditions []string
	Source        analysis.SourceReference
}

type rustModuleDeclaration struct {
	Parent        *rustModule
	Name          string
	PathOverride  string
	CfgConditions []string
	Source        analysis.SourceReference
}

func Discover(ctx context.Context, project Project, options analysis.EffectiveOptions) (discoveryResult, error) {
	return DiscoverWithSyntaxProvider(ctx, project, options, rustsyntax.NewProvider())
}

func DiscoverWithSyntaxProvider(ctx context.Context, project Project, options analysis.EffectiveOptions, provider syntax.Provider) (discoveryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return discoveryResult{}, err
	}
	discovery := discoveryResult{
		Modules:          make(map[string]*rustModule),
		SourceReferences: make(map[string]analysis.SourceReference),
		Visited:          make(map[string]struct{}),
		Diagnostics:      append([]analysis.Diagnostic(nil), project.Diagnostics...),
	}
	if len(project.Targets) == 0 {
		discovery.Diagnostics = append(discovery.Diagnostics, analysis.Diagnostic{
			Code:        "rust_no_targets",
			Severity:    "warning",
			Message:     "The selected Cargo crate has no readable source targets.",
			Path:        project.RelativeManifestPath,
			Recoverable: true,
		})
	}
	for _, target := range project.Targets {
		if err := ctx.Err(); err != nil {
			return discoveryResult{}, err
		}
		root := ensureRustModule(discovery.Modules, project, nil, "crate", target.AbsolutePath, []string{}, target.Kind)
		root.TargetKinds[target.Kind] = struct{}{}
		if err := discoverRustFile(ctx, project, &discovery, root, target.AbsolutePath, target.Kind, provider); err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded || ctx.Err() != nil {
				return discoveryResult{}, err
			}
			return discoveryResult{}, err
		}
	}
	return discovery, nil
}

func discoverRustFile(ctx context.Context, project Project, discovery *discoveryResult, module *rustModule, filePath, targetKind string, provider syntax.Provider) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !safeRustFile(project, filePath) {
		discovery.Diagnostics = append(discovery.Diagnostics, analysis.Diagnostic{
			Code:        "rust_path_outside_root",
			Severity:    "warning",
			Message:     "Rust source path resolves outside the selected project and was ignored.",
			Path:        projectRelative(project.Root, filePath),
			Recoverable: true,
		})
		return nil
	}
	if rustExcluded(project, filePath) {
		return nil
	}
	visitKey := module.ID + "\x00" + filepath.Clean(filePath)
	if _, visited := discovery.Visited[visitKey]; visited {
		return nil
	}
	discovery.Visited[visitKey] = struct{}{}
	content, err := os.ReadFile(filePath)
	if err != nil {
		discovery.Diagnostics = append(discovery.Diagnostics, analysis.Diagnostic{
			Code:        "rust_unreadable_source",
			Severity:    "error",
			Message:     fmt.Sprintf("Rust source file could not be read: %v", err),
			Path:        projectRelative(project.Root, filePath),
			Recoverable: true,
		})
		return nil
	}
	relative := projectRelative(project.Root, filePath)
	if isRustGeneratedSource(content) {
		if len(module.Path) > 0 {
			delete(discovery.Modules, strings.Join(module.Path, "::"))
		}
		discovery.Diagnostics = append(discovery.Diagnostics, analysis.Diagnostic{
			Code:        "rust_generated_source",
			Severity:    "warning",
			Message:     "Rust source appears to be generated and was excluded from static discovery.",
			Path:        relative,
			Recoverable: true,
		})
		return nil
	}
	fileSource := rustSourceReference(relative, "file", module.DisplayName(project.PackageName), nil, nil)
	discovery.SourceReferences[fileSource.ID] = fileSource
	module.SourceReferenceIDs[fileSource.ID] = struct{}{}
	module.Paths[relative] = struct{}{}
	if targetKind != "" {
		module.TargetKinds[targetKind] = struct{}{}
	}

	if provider == nil {
		discovery.Diagnostics = append(discovery.Diagnostics, rustSyntaxBackendDiagnostic(relative, fmt.Errorf("rust syntax provider is not configured")))
		return nil
	}
	parsed, parseErr := provider.Parse(ctx, syntax.Source{Path: relative, Content: content})
	if parseErr != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		discovery.Diagnostics = append(discovery.Diagnostics, rustSyntaxBackendDiagnostic(relative, parseErr))
		return nil
	}
	if parsed.Tree == nil || parsed.Tree.Root() == nil {
		parsed.Close()
		discovery.Diagnostics = append(discovery.Diagnostics, rustSyntaxBackendDiagnostic(relative, fmt.Errorf("tree-sitter returned no Rust syntax tree")))
		return nil
	}
	for _, issue := range parsed.Issues {
		discovery.Diagnostics = append(discovery.Diagnostics, rustSyntaxDiagnostic(relative, issue.Message, issue.Range))
	}
	declarations, uses, parseDiagnostics := extractRustSyntaxItems(parsed.Tree.Root(), project, discovery.Modules, discovery.SourceReferences, module, fileSource, project.IncludeTests, targetKind)
	parsed.Close()
	discovery.Uses = append(discovery.Uses, uses...)
	discovery.Diagnostics = append(discovery.Diagnostics, parseDiagnostics...)
	for _, declaration := range declarations {
		if err := ctx.Err(); err != nil {
			return err
		}
		if declaration.Source.ID != "" {
			discovery.SourceReferences[declaration.Source.ID] = declaration.Source
			declaration.Parent.SourceReferenceIDs[declaration.Source.ID] = struct{}{}
		}
		childPath, resolveErr := resolveRustModulePath(project, declaration)
		if resolveErr != nil {
			discovery.Diagnostics = append(discovery.Diagnostics, rustModuleDiagnostic(declaration.Source, resolveErr.Error(), "rust_module_path_invalid"))
			continue
		}
		if childPath == "" {
			discovery.Diagnostics = append(discovery.Diagnostics, rustModuleDiagnostic(declaration.Source, fmt.Sprintf("Rust module %q has no readable source file.", declaration.Name), "rust_module_source_missing"))
			continue
		}
		child := ensureRustModule(discovery.Modules, project, declaration.Parent, "file_module", childPath, append(append([]string(nil), declaration.Parent.Path...), declaration.Name), targetKind)
		child.SourceReferenceIDs[declaration.Source.ID] = struct{}{}
		if len(declaration.CfgConditions) > 0 {
			for _, condition := range declaration.CfgConditions {
				child.CfgConditions[condition] = struct{}{}
			}
		}
		if err := discoverRustFile(ctx, project, discovery, child, childPath, targetKind, provider); err != nil {
			return err
		}
	}
	return nil
}

func ensureRustModule(modules map[string]*rustModule, project Project, parent *rustModule, kind, filePath string, modulePath []string, targetKind string) *rustModule {
	key := strings.Join(modulePath, "::")
	module := modules[key]
	if module == nil {
		name := project.PackageName
		if len(modulePath) > 0 {
			name = modulePath[len(modulePath)-1]
		}
		module = &rustModule{
			ID:                 rustModuleID(project.PackageName, modulePath),
			Path:               append([]string(nil), modulePath...),
			Kind:               kind,
			Name:               name,
			FilePath:           filePath,
			ModuleDir:          rustModuleDir(filePath, name, len(modulePath) == 0),
			SourceReferenceIDs: make(map[string]struct{}),
			Tags:               make(map[string]struct{}),
			CfgConditions:      make(map[string]struct{}),
			DeclaredModules:    make(map[string]struct{}),
			Paths:              make(map[string]struct{}),
			TargetKinds:        make(map[string]struct{}),
			MacroAttributes:    make(map[string]struct{}),
		}
		modules[key] = module
	} else if module.Kind == "file_module" && kind == "inline_module" {
		module.Kind = kind
	}
	if module.FilePath == "" {
		module.FilePath = filePath
		module.ModuleDir = rustModuleDir(filePath, module.Name, len(module.Path) == 0)
	}
	if targetKind != "" {
		module.TargetKinds[targetKind] = struct{}{}
	}
	if parent != nil {
		parent.DeclaredModules[module.Name] = struct{}{}
	}
	return module
}

func rustModuleDir(filePath, name string, root bool) string {
	directory := filepath.Dir(filePath)
	if root || strings.EqualFold(filepath.Base(filePath), "mod.rs") {
		return directory
	}
	return filepath.Join(directory, name)
}

func resolveRustModulePath(project Project, declaration rustModuleDeclaration) (string, error) {
	base := declaration.Parent.ModuleDir
	if len(declaration.Parent.Path) == 0 && declaration.Source.Path != "" {
		// A crate root can be shared by lib/bin targets, whose source files may
		// live in different directories. Resolve each root declaration from the
		// file that declared it rather than from the first target's module dir.
		base = filepath.Dir(fileSourcePathAbsolute(project, declaration.Source.Path))
	}
	if declaration.PathOverride != "" {
		candidate := filepath.Clean(filepath.Join(base, filepath.FromSlash(declaration.PathOverride)))
		if !pathWithin(project.CrateRoot, candidate) || !resolvedPathWithin(project.Root, candidate) {
			return "", fmt.Errorf("rust #[path] module %q escapes the selected crate root", declaration.PathOverride)
		}
		if fileExists(candidate) {
			return candidate, nil
		}
		return "", nil
	}
	for _, candidate := range []string{
		filepath.Join(base, declaration.Name+".rs"),
		filepath.Join(base, declaration.Name, "mod.rs"),
	} {
		if !pathWithin(project.CrateRoot, candidate) || !resolvedPathWithin(project.Root, candidate) {
			continue
		}
		if fileExists(candidate) {
			return candidate, nil
		}
	}
	return "", nil
}

func fileSourcePathAbsolute(project Project, relative string) string {
	return filepath.Join(project.Root, filepath.FromSlash(relative))
}

func rustCfgTestOnly(expression string) bool {
	expression = strings.TrimSpace(strings.ToLower(expression))
	if expression == "test" {
		return true
	}
	open := strings.IndexByte(expression, '(')
	if open <= 0 || !strings.HasSuffix(expression, ")") {
		return false
	}
	operator := strings.TrimSpace(expression[:open])
	args := splitRustCfgArguments(expression[open+1 : len(expression)-1])
	if len(args) == 0 {
		return false
	}
	switch operator {
	case "all":
		for _, argument := range args {
			if rustCfgTestOnly(argument) {
				return true
			}
		}
		return false
	case "any":
		for _, argument := range args {
			if !rustCfgTestOnly(argument) {
				return false
			}
		}
		return true
	case "not":
		return false
	default:
		return false
	}
}

func splitRustCfgArguments(value string) []string {
	arguments := make([]string, 0)
	start := 0
	depth := 0
	quote := byte(0)
	for index := 0; index < len(value); index++ {
		char := value[index]
		if quote != 0 {
			if char == quote && (index == 0 || value[index-1] != '\\') {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		switch char {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				arguments = append(arguments, strings.TrimSpace(value[start:index]))
				start = index + 1
			}
		}
	}
	if tail := strings.TrimSpace(value[start:]); tail != "" {
		arguments = append(arguments, tail)
	}
	return arguments
}

func rustCfgKnown(conditions []string) bool {
	for _, condition := range conditions {
		lower := strings.ToLower(strings.TrimSpace(condition))
		if lower == "test" || lower == "true" || lower == "false" {
			continue
		}
		return false
	}
	return true
}

func mapUseKind(public bool) string {
	if public {
		return "pub_use"
	}
	return "use"
}

func useAliasSuffix(alias string) string {
	if alias == "" {
		return ""
	}
	return " as " + alias
}

func useImportedName(value string) string {
	value = strings.TrimSuffix(value, "::*")
	if index := strings.LastIndex(value, "::"); index >= 0 {
		return value[index+2:]
	}
	return value
}

func rustSourceReference(relative, kind, symbol string, start, end *analysis.Position) analysis.SourceReference {
	id := stableID("source", kind, relative, symbol, positionKey(start), positionKey(end))
	return analysis.SourceReference{ID: id, Path: relative, Start: start, End: end, Symbol: symbol, Kind: kind}
}

func positionKey(position *analysis.Position) string {
	if position == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", position.Line, position.Column)
}

func stableID(kind string, parts ...string) string {
	payload := kind + "\x00" + strings.Join(parts, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "rust:" + kind + ":" + hex.EncodeToString(sum[:8])
}

func rustModuleID(packageName string, modulePath []string) string {
	if len(modulePath) == 0 {
		return "rust:crate:" + normalizeRustIdent(packageName)
	}
	return "rust:module:" + normalizeRustIdent(packageName) + "::" + strings.Join(modulePath, "::")
}

func (module *rustModule) DisplayName(packageName string) string {
	if len(module.Path) == 0 {
		return packageName
	}
	return packageName + "::" + strings.Join(module.Path, "::")
}

func normalizeRustIdent(value string) string {
	return strings.TrimPrefix(value, "r#")
}

func rustModuleDiagnostic(source analysis.SourceReference, message, code string) analysis.Diagnostic {
	return analysis.Diagnostic{Code: code, Severity: "warning", Message: message, Path: source.Path, Location: source.Start, Recoverable: true, Metadata: map[string]any{"source_reference_id": source.ID}}
}

func safeRustFile(project Project, filePath string) bool {
	return pathWithin(project.CrateRoot, filePath) && resolvedPathWithin(project.Root, filePath) && fileExists(filePath)
}

func rustExcluded(project Project, filePath string) bool {
	relativeProject := projectRelative(project.Root, filePath)
	relativeCrate := projectRelative(project.CrateRoot, filePath)
	for _, value := range []string{relativeProject, relativeCrate} {
		if matchesRustDefaultExclusion(value, project.IncludeTests, project.IncludeExamples) || matchesAnyRustExclude(value, project.ExcludePatterns) {
			return true
		}
	}
	return false
}

func matchesRustDefaultExclusion(relative string, includeTests, includeExamples bool) bool {
	clean := filepath.ToSlash(filepath.Clean(relative))
	segments := strings.Split(clean, "/")
	for _, segment := range segments {
		switch strings.ToLower(segment) {
		case ".git", "target", "build", "cache", ".cache", "dist", "out", "tmp", "vendor", "external", "generated":
			return true
		case "tests":
			if !includeTests {
				return true
			}
		case "examples", "benches":
			if !includeExamples {
				return true
			}
		}
	}
	if !includeTests && strings.HasSuffix(strings.ToLower(filepath.Base(clean)), "_test.rs") {
		return true
	}
	return false
}

func matchesAnyRustExclude(relative string, patterns []string) bool {
	clean := filepath.ToSlash(filepath.Clean(relative))
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(filepath.Clean(strings.TrimSpace(pattern)))
		pattern = strings.TrimPrefix(pattern, "./")
		if pattern == "" || pattern == "." {
			continue
		}
		if matched, _ := path.Match(pattern, clean); matched {
			return true
		}
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
				return true
			}
		}
		if strings.HasPrefix(pattern, "**/") {
			if matched, _ := path.Match(strings.TrimPrefix(pattern, "**/"), path.Base(clean)); matched {
				return true
			}
		}
	}
	return false
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func isRustGeneratedSource(content []byte) bool {
	lines := strings.Split(string(content), "\n")
	if len(lines) > 20 {
		lines = lines[:20]
	}
	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(lower, "generated") && (strings.HasPrefix(lower, "//") || strings.HasPrefix(lower, "/*") || strings.HasPrefix(lower, "*") || strings.HasPrefix(lower, "#")) {
			return true
		}
	}
	return false
}
