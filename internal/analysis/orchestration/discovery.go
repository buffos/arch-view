package orchestration

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

var defaultFixedExclusions = []string{
	".cache", ".git", ".hg", ".mypy_cache", ".nox", ".pytest_cache", ".svn", ".tox",
	".venv", "bin", "build", "cache", "coverage", "dist", "env", "external", "generated",
	"node_modules", "out", "target", "tmp", "vendor", "venv",
}

// IsDefaultExcludedDirectory reports whether discovery excludes a directory
// name from every analyzer plan. Live input fingerprinting uses the same
// boundary so VCS/cache churn cannot create revisions for files no analyzer
// is eligible to observe.
func IsDefaultExcludedDirectory(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, excluded := range defaultFixedExclusions {
		if name == excluded {
			return true
		}
	}
	return false
}

const hostConfigurationFileName = ".archview.json"

func defaultDiscoveryPolicy() DiscoveryPolicy {
	return DiscoveryPolicy{
		Version:         DiscoveryPolicyVersion,
		FixedExclusions: append([]string(nil), defaultFixedExclusions...),
		MaxPlannedJobs:  MaxPlannedJobs,
		FollowSymlinks:  false,
	}
}

func normalizeDiscoveryPolicy(policy DiscoveryPolicy) DiscoveryPolicy {
	defaults := defaultDiscoveryPolicy()
	if policy.Version == "" {
		policy.Version = defaults.Version
	}
	merged := make(map[string]struct{}, len(defaults.FixedExclusions)+len(policy.FixedExclusions))
	for _, value := range append(defaults.FixedExclusions, policy.FixedExclusions...) {
		value = strings.ToLower(strings.Trim(filepath.ToSlash(filepath.Clean(value)), "/"))
		if value == "" || value == "." {
			continue
		}
		merged[value] = struct{}{}
	}
	policy.FixedExclusions = make([]string, 0, len(merged))
	for value := range merged {
		policy.FixedExclusions = append(policy.FixedExclusions, value)
	}
	sort.Strings(policy.FixedExclusions)
	if policy.MaxPlannedJobs <= 0 || policy.MaxPlannedJobs > MaxPlannedJobs {
		policy.MaxPlannedJobs = MaxPlannedJobs
	}
	// The v1 policy never follows symlinks. Keep a caller-supplied true value
	// from weakening repository containment.
	policy.FollowSymlinks = false
	return policy
}

func NewProjectDiscoveryService(registry *analysis.Registry, policy DiscoveryPolicy) *ProjectDiscoveryService {
	return &ProjectDiscoveryService{Registry: registry, Policy: normalizeDiscoveryPolicy(policy)}
}

// DiscoverProjectRoots finds strong-marker roots and the repository-owned file
// set in one lexical traversal. It never follows symlinks and does not invoke
// analyzer detection; candidate evaluation is a separate planning step.
func (s *ProjectDiscoveryService) DiscoverProjectRoots(ctx context.Context, repositoryRoot string) (DiscoveryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, err := normalizeRepositoryRoot(repositoryRoot)
	if err != nil {
		return DiscoveryResult{}, err
	}
	policy := normalizeDiscoveryPolicy(safeDiscoveryPolicy(s))
	result := DiscoveryResult{
		RepositoryRoot:      root,
		RepositoryRootLabel: filepath.Base(root),
		Policy:              policy,
		Roots:               []ProjectRootCandidate{},
		AllSourcePaths:      []string{},
		Diagnostics:         []analysis.Diagnostic{},
	}
	if result.RepositoryRootLabel == "" || result.RepositoryRootLabel == string(filepath.Separator) {
		result.RepositoryRootLabel = root
	}

	strongMarkers := make(map[string]map[string]struct{})
	if s != nil && s.Registry != nil {
		for _, analyzer := range s.Registry.List() {
			manifest := analyzer.Manifest()
			for _, marker := range manifest.DetectionMarkers {
				if marker.Weight < 1 {
					continue
				}
				value := filepath.ToSlash(filepath.Clean(marker.Value))
				if value == "." || value == "" || strings.Contains(value, "/") {
					continue
				}
				key := marker.Kind + "\x00" + value
				if strongMarkers[key] == nil {
					strongMarkers[key] = map[string]struct{}{}
				}
				strongMarkers[key][manifest.ID] = struct{}{}
			}
		}
	}

	markerRoots := make(map[string]map[string]struct{})
	walkErr := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		relative, relativeErr := filepath.Rel(root, filePath)
		if relativeErr != nil {
			return relativeErr
		}
		relative = normalizeRelativePath(relative)
		if walkErr != nil {
			if relative == "." {
				return walkErr
			}
			result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
				Code:        "discovery_path_unreadable",
				Severity:    "warning",
				Message:     fmt.Sprintf("path could not be inspected during project discovery: %v", walkErr),
				Path:        relative,
				Recoverable: true,
			})
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if relative != "." && fixedExcluded(relative, policy.FixedExclusions) {
				return filepath.SkipDir
			}
			for key, analyzers := range strongMarkers {
				kind, value, ok := splitMarkerKey(key)
				if !ok || kind != "directory" || value != entry.Name() {
					continue
				}
				parent := normalizeRelativePath(filepath.Dir(relative))
				if markerRoots[parent] == nil {
					markerRoots[parent] = map[string]struct{}{}
				}
				for analyzerID := range analyzers {
					markerRoots[parent][analyzerID+"\x00"+value] = struct{}{}
				}
			}
			return nil
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			result.Diagnostics = append(result.Diagnostics, analysis.Diagnostic{
				Code:        "discovery_path_unreadable",
				Severity:    "warning",
				Message:     fmt.Sprintf("file metadata could not be read: %v", infoErr),
				Path:        relative,
				Recoverable: true,
			})
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if !resolvedPathWithin(root, filePath) {
			return nil
		}
		parent := normalizeRelativePath(filepath.Dir(relative))
		for key, analyzers := range strongMarkers {
			kind, value, ok := splitMarkerKey(key)
			if !ok || kind != "file" || value != entry.Name() {
				continue
			}
			if markerRoots[parent] == nil {
				markerRoots[parent] = map[string]struct{}{}
			}
			for analyzerID := range analyzers {
				markerRoots[parent][analyzerID+"\x00"+value] = struct{}{}
			}
		}
		// The host configuration controls planning and presentation. It is not
		// analyzer source input, so layout-only edits must not invalidate every
		// job cache entry. Marker discovery still runs above for completeness.
		if entry.Name() != hostConfigurationFileName {
			result.AllSourcePaths = append(result.AllSourcePaths, relative)
		}
		return nil
	})
	if walkErr != nil {
		if ctx.Err() != nil {
			return DiscoveryResult{}, analysis.NewHostError(analysis.ErrCancelled, "project discovery was cancelled", nil)
		}
		return DiscoveryResult{}, analysis.WrapHostError(analysis.ErrUnreadableProject, "repository discovery failed", walkErr, map[string]any{"repository_root": root})
	}
	sort.Strings(result.AllSourcePaths)
	result.AllSourcePaths = uniqueStrings(result.AllSourcePaths)

	rootSet := make(map[string]struct{})
	for relative := range markerRoots {
		rootSet[relative] = struct{}{}
	}
	for _, relative := range sortedExplicitRoots(nil, root) {
		rootSet[relative] = struct{}{}
	}
	result.Roots = candidatesFromMarkerRoots(root, rootSet, markerRoots)
	ResolveNestedProjectOwnership(&result)
	return result, nil
}

// ResolveNestedProjectOwnership carves every nested discovered root out of
// its ancestors and assigns each repository file to the deepest owner.
func ResolveNestedProjectOwnership(result *DiscoveryResult) {
	if result == nil {
		return
	}
	sort.Slice(result.Roots, func(i, j int) bool {
		return rootSortKey(result.Roots[i].RelativePath) < rootSortKey(result.Roots[j].RelativePath)
	})
	for index := range result.Roots {
		result.Roots[index].NestedRootExclusions = []string{}
		result.Roots[index].OwnedSourcePaths = []string{}
	}
	for parentIndex := range result.Roots {
		parent := result.Roots[parentIndex].RelativePath
		for childIndex := range result.Roots {
			if parentIndex == childIndex {
				continue
			}
			child := result.Roots[childIndex].RelativePath
			if !relativeDescendant(parent, child) {
				continue
			}
			relativeToParent := relativePathFrom(parent, child)
			result.Roots[parentIndex].NestedRootExclusions = append(result.Roots[parentIndex].NestedRootExclusions, relativeToParent)
		}
		sort.Strings(result.Roots[parentIndex].NestedRootExclusions)
		result.Roots[parentIndex].NestedRootExclusions = uniqueStrings(result.Roots[parentIndex].NestedRootExclusions)
	}
	for _, sourcePath := range result.AllSourcePaths {
		owner := -1
		for index := range result.Roots {
			if !relativeDescendantOrSame(result.Roots[index].RelativePath, sourcePath) {
				continue
			}
			if owner == -1 || relativeDepth(result.Roots[index].RelativePath) > relativeDepth(result.Roots[owner].RelativePath) {
				owner = index
			}
		}
		if owner >= 0 {
			result.Roots[owner].OwnedSourcePaths = append(result.Roots[owner].OwnedSourcePaths, sourcePath)
		}
	}
	for index := range result.Roots {
		sort.Strings(result.Roots[index].OwnedSourcePaths)
		result.Roots[index].OwnedSourcePaths = uniqueStrings(result.Roots[index].OwnedSourcePaths)
	}
}

func safeDiscoveryPolicy(service *ProjectDiscoveryService) DiscoveryPolicy {
	if service == nil {
		return defaultDiscoveryPolicy()
	}
	return service.Policy
}

func candidatesFromMarkerRoots(repositoryRoot string, roots map[string]struct{}, markers map[string]map[string]struct{}) []ProjectRootCandidate {
	values := make([]string, 0, len(roots))
	for relative := range roots {
		values = append(values, normalizeRelativePath(relative))
	}
	sort.Slice(values, func(i, j int) bool { return rootSortKey(values[i]) < rootSortKey(values[j]) })
	result := make([]ProjectRootCandidate, 0, len(values))
	for _, relative := range values {
		markerValues := []string{}
		for key := range markers[relative] {
			_, value, ok := splitMarkerKey(key)
			if ok {
				markerValues = append(markerValues, value)
			}
		}
		sort.Strings(markerValues)
		markerValues = uniqueStrings(markerValues)
		absolute := repositoryRoot
		if relative != "." {
			absolute = filepath.Join(repositoryRoot, filepath.FromSlash(relative))
		}
		result = append(result, ProjectRootCandidate{AbsolutePath: absolute, RelativePath: relative, StrongMarkers: markerValues})
	}
	return result
}

func splitMarkerKey(value string) (string, string, bool) {
	parts := strings.SplitN(value, "\x00", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func normalizeRepositoryRoot(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "repository root is required", nil)
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "repository root could not be normalized", err, map[string]any{"repository_root": value})
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrUnreadableProject, "repository root could not be read", err, map[string]any{"repository_root": value})
	}
	if !info.IsDir() {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "repository root must be a directory", map[string]any{"repository_root": value})
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err == nil {
		absolute = filepath.Clean(resolved)
	}
	return absolute, nil
}

func normalizeRelativePath(value string) string {
	value = filepath.ToSlash(filepath.Clean(value))
	if value == "" || value == "." {
		return "."
	}
	return strings.TrimPrefix(value, "./")
}

func fixedExcluded(relative string, exclusions []string) bool {
	clean := normalizeRelativePath(relative)
	if clean == "." {
		return false
	}
	for _, segment := range strings.Split(clean, "/") {
		for _, exclusion := range exclusions {
			if strings.EqualFold(segment, exclusion) {
				return true
			}
		}
	}
	return false
}

func resolvedPathWithin(root, candidate string) bool {
	resolvedRoot, rootErr := filepath.EvalSymlinks(root)
	resolvedCandidate, candidateErr := filepath.EvalSymlinks(candidate)
	if rootErr != nil || candidateErr != nil {
		return false
	}
	return pathWithin(resolvedRoot, resolvedCandidate)
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func relativeDescendantOrSame(parent, candidate string) bool {
	parent = normalizeRelativePath(parent)
	candidate = normalizeRelativePath(candidate)
	return parent == "." || candidate == parent || strings.HasPrefix(candidate, parent+"/")
}

func relativeDescendant(parent, candidate string) bool {
	return parent != candidate && relativeDescendantOrSame(parent, candidate)
}

func relativePathFrom(parent, candidate string) string {
	if parent == "." {
		return normalizeRelativePath(candidate)
	}
	return strings.TrimPrefix(candidate, parent+"/")
}

func relativeDepth(value string) int {
	value = normalizeRelativePath(value)
	if value == "." {
		return 0
	}
	return len(strings.Split(value, "/"))
}

func rootSortKey(value string) string {
	return fmt.Sprintf("%04d:%s", relativeDepth(value), normalizeRelativePath(value))
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	sort.Strings(values)
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

// sortedExplicitRoots is kept as a tiny seam for the planner, which adds
// assignment-created roots after discovery without making the traversal
// depend on assignment persistence.
func sortedExplicitRoots(values []string, _ string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, normalizeRelativePath(value))
	}
	sort.Slice(result, func(i, j int) bool { return rootSortKey(result[i]) < rootSortKey(result[j]) })
	return uniqueStrings(result)
}
