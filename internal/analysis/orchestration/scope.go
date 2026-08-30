package orchestration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

// NormalizeSourceScopePolicy validates and canonicalizes invocation-root
// source filters. It intentionally accepts a small, predictable glob subset;
// it is not a .gitignore parser.
func NormalizeSourceScopePolicy(policy SourceScopePolicy, invocationRoot string) (SourceScopePolicy, error) {
	root := normalizeRelativePath(invocationRoot)
	if root == "" {
		root = "."
	}
	if policy.PolicyVersion == "" {
		policy.PolicyVersion = SourceScopePolicyVersion
	}
	if policy.PolicyVersion != SourceScopePolicyVersion {
		return SourceScopePolicy{}, scopeFilterError("source-scope policy version is unsupported", map[string]any{"policy_version": policy.PolicyVersion})
	}
	if policy.InvocationRoot != "" {
		configured := normalizeRelativePath(policy.InvocationRoot)
		if configured != root {
			return SourceScopePolicy{}, scopeFilterError("source-scope policy invocation root does not match the plan invocation root", map[string]any{"expected": root, "actual": policy.InvocationRoot})
		}
	}
	policy.InvocationRoot = root

	exclude, err := normalizeGlobList(policy.Exclude)
	if err != nil {
		return SourceScopePolicy{}, err
	}
	policy.Exclude = exclude

	rules := make(map[string]map[string]struct{}, len(policy.Include))
	seenRule := make(map[string]struct{}, len(policy.Include))
	for _, rule := range policy.Include {
		analyzerID := strings.TrimSpace(rule.AnalyzerID)
		if analyzerID == "" || strings.ContainsAny(analyzerID, "\x00\r\n") {
			return SourceScopePolicy{}, scopeFilterError("source-scope include rule requires a safe analyzer id", map[string]any{"analyzer_id": rule.AnalyzerID})
		}
		if _, exists := seenRule[analyzerID]; exists {
			return SourceScopePolicy{}, scopeFilterError("source-scope include rules must contain one rule per analyzer id", map[string]any{"analyzer_id": analyzerID})
		}
		seenRule[analyzerID] = struct{}{}
		values, err := normalizeGlobList(rule.Globs)
		if err != nil {
			return SourceScopePolicy{}, err
		}
		if rules[analyzerID] == nil {
			rules[analyzerID] = map[string]struct{}{}
		}
		for _, value := range values {
			rules[analyzerID][value] = struct{}{}
		}
	}
	policy.Include = make([]AnalyzerIncludeRule, 0, len(rules))
	for analyzerID, values := range rules {
		globs := make([]string, 0, len(values))
		for value := range values {
			globs = append(globs, value)
		}
		sort.Strings(globs)
		policy.Include = append(policy.Include, AnalyzerIncludeRule{AnalyzerID: analyzerID, Globs: globs})
	}
	sort.Slice(policy.Include, func(i, j int) bool { return policy.Include[i].AnalyzerID < policy.Include[j].AnalyzerID })
	return policy, nil
}

func normalizeGlobList(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized, err := normalizeGlob(value)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeGlob(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return "", scopeFilterError("source-scope glob cannot be empty or contain surrounding whitespace", map[string]any{"glob": value})
	}
	if strings.ContainsAny(value, "\\\x00\r\n") || strings.HasPrefix(value, "!") || strings.HasPrefix(value, "#") {
		return "", scopeFilterError("source-scope glob contains unsupported syntax", map[string]any{"glob": value})
	}
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") || windowsAbsolute(value) {
		return "", scopeFilterError("source-scope glob must be relative to the invocation root", map[string]any{"glob": value})
	}
	for strings.HasPrefix(value, "./") {
		value = strings.TrimPrefix(value, "./")
	}
	if value == "" {
		return "", scopeFilterError("source-scope glob cannot resolve to an empty path", map[string]any{"glob": value})
	}
	segments := strings.Split(value, "/")
	canonicalSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == ".." || segment == "" {
			return "", scopeFilterError("source-scope glob cannot contain empty or parent-traversal segments", map[string]any{"glob": value})
		}
		if segment == "." {
			continue
		}
		if segment == "**" {
			canonicalSegments = append(canonicalSegments, segment)
			continue
		}
		if strings.Contains(segment, "**") {
			return "", scopeFilterError("recursive source-scope glob syntax must use a complete ** segment", map[string]any{"glob": value})
		}
		if _, err := path.Match(segment, "arch-view-glob-probe"); err != nil {
			return "", scopeFilterError("source-scope glob is malformed", map[string]any{"glob": value, "error": err.Error()})
		}
		canonicalSegments = append(canonicalSegments, segment)
	}
	if len(canonicalSegments) == 0 {
		return ".", nil
	}
	return strings.Join(canonicalSegments, "/"), nil
}

func scopeFilterError(message string, details map[string]any) error {
	return analysis.NewHostError(analysis.ErrAnalysisScopeFilterInvalid, message, details)
}

func windowsAbsolute(value string) bool {
	return len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':'
}

func policyFingerprint(policy SourceScopePolicy) string {
	data, _ := json.Marshal(policy)
	return sha256Fingerprint(data)
}

func matchedSourceFingerprint(paths []string, root ProjectRootCandidate, invocationRoot string) string {
	values := uniqueStrings(append([]string(nil), paths...))
	type sourceFingerprintEntry struct {
		Path          string `json:"path"`
		ContentDigest string `json:"content_digest"`
	}
	entries := make([]sourceFingerprintEntry, 0, len(values))
	for _, invocationPath := range values {
		repositoryPath := repositoryPathFromInvocation(invocationRoot, invocationPath)
		localPath := relativeToProject(root.RelativePath, repositoryPath)
		contentDigest := "unreadable"
		if root.AbsolutePath != "" {
			contentDigest = sourceContentDigest(filepath.Join(root.AbsolutePath, filepath.FromSlash(localPath)))
		}
		entries = append(entries, sourceFingerprintEntry{Path: invocationPath, ContentDigest: contentDigest})
	}
	data, _ := json.Marshal(entries)
	return sha256Fingerprint(data)
}

func sourceContentDigest(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return "unreadable"
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		_ = file.Close()
		return "unreadable"
	}
	if err := file.Close(); err != nil {
		return "unreadable"
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func sha256Fingerprint(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func scopeID(relativeRoot, analyzerID string) string {
	sum := sha256.Sum256([]byte(normalizeRelativePath(relativeRoot) + "\x00" + analyzerID))
	return "scope-" + hex.EncodeToString(sum[:])
}

func jobID(job AnalyzerJob) string {
	identity := job.CacheKey
	if identity == "" {
		identity = job.ScopeID + "\x00" + job.AnalyzerVersion + "\x00" + job.EffectiveOptionsFingerprint + "\x00" + job.SourceScopeFingerprint + "\x00" + job.MatchedSourceSetFingerprint
	}
	sum := sha256.Sum256([]byte(identity))
	return "job-" + hex.EncodeToString(sum[:12])
}

// cacheKey is the canonical identity for one executable analyzer scope. It
// includes every input that can change observations while keeping ScopeID
// stable for projection and selection across reanalysis.
func cacheKey(job AnalyzerJob, repositoryRoot, invocationRoot string, discoveryPolicy DiscoveryPolicy) string {
	manifestAPIVersion := job.Manifest.APIVersion
	if manifestAPIVersion == "" && job.LogicalAnalyzerID != "" {
		manifestAPIVersion = analysis.AnalyzerAPIVersion
	}
	identity := struct {
		SchemaVersion           string                     `json:"schema_version"`
		DiscoveryPolicy         DiscoveryPolicy            `json:"discovery_policy"`
		RepositoryRoot          string                     `json:"repository_root"`
		InvocationRoot          string                     `json:"invocation_root"`
		ProjectRoot             string                     `json:"project_root"`
		LogicalAnalyzerID       string                     `json:"logical_analyzer_id"`
		AnalyzerVersion         string                     `json:"analyzer_version"`
		AnalyzerAPIVersion      string                     `json:"analyzer_api_version"`
		AnalyzerRuntimeIdentity string                     `json:"analyzer_runtime_identity,omitempty"`
		RuntimeMode             string                     `json:"runtime_mode"`
		RuntimeSource           string                     `json:"runtime_source"`
		RuntimePlatform         string                     `json:"runtime_platform"`
		SelectionSource         SelectionSource            `json:"selection_source"`
		AssignmentPath          string                     `json:"assignment_path,omitempty"`
		Selection               analysis.AnalyzerSelection `json:"selection"`
		NestedRootExclusions    []string                   `json:"nested_root_exclusions"`
		Options                 map[string]any             `json:"options"`
		SourceScopeFingerprint  string                     `json:"source_scope_fingerprint"`
		MatchedSourceSet        string                     `json:"matched_source_set_fingerprint"`
		MatchedPaths            []string                   `json:"matched_paths"`
	}{
		SchemaVersion:           JobPlanSchemaVersion,
		DiscoveryPolicy:         normalizeDiscoveryPolicy(discoveryPolicy),
		RepositoryRoot:          filepath.Clean(repositoryRoot),
		InvocationRoot:          normalizeRelativePath(invocationRoot),
		ProjectRoot:             normalizeRelativePath(job.RelativeProjectRoot),
		LogicalAnalyzerID:       job.LogicalAnalyzerID,
		AnalyzerVersion:         job.AnalyzerVersion,
		AnalyzerAPIVersion:      manifestAPIVersion,
		AnalyzerRuntimeIdentity: job.Manifest.RuntimeIdentity,
		RuntimeMode:             job.RuntimeMode,
		RuntimeSource:           job.RuntimeSource,
		RuntimePlatform:         job.RuntimePlatform,
		SelectionSource:         job.SelectionSource,
		AssignmentPath:          normalizeRelativePath(job.AssignmentPath),
		Selection:               job.Selection,
		NestedRootExclusions:    uniqueStrings(append([]string(nil), job.NestedRootExclusions...)),
		Options:                 job.Options.Values,
		SourceScopeFingerprint:  job.SourceScopeFingerprint,
		MatchedSourceSet:        job.MatchedSourceSetFingerprint,
		MatchedPaths:            uniqueStrings(append([]string(nil), job.EffectiveSourceScope.MatchedPaths...)),
	}
	data, _ := json.Marshal(identity)
	return sha256Fingerprint(data)
}

func namespacedID(scope, local string) string {
	return scope + "::" + urlPathEscape(local)
}

func urlPathEscape(value string) string {
	// url.PathEscape is intentionally wrapped so all callers use one identity
	// rule and local empty IDs remain visibly invalid to result validation.
	return pathEscape(value)
}

func pathEscape(value string) string {
	const hexDigits = "0123456789ABCDEF"
	var builder strings.Builder
	for _, character := range []byte(value) {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || strings.ContainsRune("-._~", rune(character)) {
			builder.WriteByte(character)
			continue
		}
		builder.WriteByte('%')
		builder.WriteByte(hexDigits[character>>4])
		builder.WriteByte(hexDigits[character&0x0f])
	}
	return builder.String()
}

func globMatches(pattern, value string) bool {
	pattern = normalizeRelativePath(pattern)
	value = normalizeRelativePath(value)
	if pattern == "." {
		return true
	}
	patternParts := strings.Split(pattern, "/")
	valueParts := strings.Split(value, "/")
	if matchGlobParts(patternParts, valueParts, 0, 0, map[string]bool{}) {
		return true
	}
	// A directory match applies to all descendants. Source candidates are
	// files, so accepting a matched prefix does not make a file glob escape its
	// intended path shape.
	for prefixLength := len(valueParts) - 1; prefixLength > 0; prefixLength-- {
		if matchGlobParts(patternParts, valueParts[:prefixLength], 0, 0, map[string]bool{}) {
			return true
		}
	}
	return false
}

func matchGlobParts(pattern, value []string, patternIndex, valueIndex int, memo map[string]bool) bool {
	key := fmt.Sprintf("%d:%d", patternIndex, valueIndex)
	if cached, ok := memo[key]; ok {
		return cached
	}
	var matched bool
	switch {
	case patternIndex == len(pattern):
		matched = valueIndex == len(value)
	case pattern[patternIndex] == "**":
		matched = matchGlobParts(pattern, value, patternIndex+1, valueIndex, memo) || (valueIndex < len(value) && matchGlobParts(pattern, value, patternIndex, valueIndex+1, memo))
	case valueIndex < len(value):
		segmentMatched, err := path.Match(pattern[patternIndex], value[valueIndex])
		matched = err == nil && segmentMatched && matchGlobParts(pattern, value, patternIndex+1, valueIndex+1, memo)
	}
	memo[key] = matched
	return matched
}

func includeGlobs(policy SourceScopePolicy, analyzerID string) []string {
	for _, rule := range policy.Include {
		if rule.AnalyzerID == analyzerID {
			return append([]string(nil), rule.Globs...)
		}
	}
	return nil
}

func relativeToInvocation(invocationRoot, repositoryPath string) string {
	if invocationRoot == "." || invocationRoot == "" {
		return normalizeRelativePath(repositoryPath)
	}
	if repositoryPath == invocationRoot {
		return "."
	}
	return strings.TrimPrefix(normalizeRelativePath(repositoryPath), normalizeRelativePath(invocationRoot)+"/")
}

func repositoryPathFromInvocation(invocationRoot, invocationPath string) string {
	invocationRoot = normalizeRelativePath(invocationRoot)
	invocationPath = normalizeRelativePath(invocationPath)
	if invocationRoot == "." {
		return invocationPath
	}
	if invocationPath == "." {
		return invocationRoot
	}
	return normalizeRelativePath(invocationRoot + "/" + invocationPath)
}

func relativeToProject(projectRoot, repositoryPath string) string {
	if projectRoot == "." || projectRoot == "" {
		return normalizeRelativePath(repositoryPath)
	}
	if repositoryPath == projectRoot {
		return "."
	}
	return strings.TrimPrefix(normalizeRelativePath(repositoryPath), normalizeRelativePath(projectRoot)+"/")
}

func buildEffectiveSourceScope(policy SourceScopePolicy, root ProjectRootCandidate, analyzerID string, options analysis.EffectiveOptions) (analysis.SourceScope, string, error) {
	include := includeGlobs(policy, analyzerID)
	analyzerExcludes, err := optionGlobValues(options)
	if err != nil {
		return analysis.SourceScope{}, "", err
	}
	allExcludes := append([]string(nil), policy.Exclude...)
	allExcludes = append(allExcludes, analyzerExcludes...)
	allExcludes = uniqueStrings(allExcludes)
	matched := []string{}
	matchedLocal := []string{}
	excluded := []string{}
	for _, repositoryPath := range root.OwnedSourcePaths {
		invocationPath := relativeToInvocation(policy.InvocationRoot, repositoryPath)
		localPath := relativeToProject(root.RelativePath, repositoryPath)
		if len(include) > 0 && !matchesAnyGlob(include, invocationPath) {
			excluded = append(excluded, invocationPath)
			continue
		}
		if matchesAnyGlob(policy.Exclude, invocationPath) || matchesAnyGlob(analyzerExcludes, localPath) || matchesAnyGlob(analyzerExcludes, invocationPath) {
			excluded = append(excluded, invocationPath)
			continue
		}
		matched = append(matched, invocationPath)
		matchedLocal = append(matchedLocal, localPath)
	}
	matched = uniqueStrings(matched)
	matchedLocal = uniqueStrings(matchedLocal)
	excluded = uniqueStrings(excluded)
	policyHash := policyFingerprint(policy)
	matchedHash := matchedSourceFingerprint(matched, root, policy.InvocationRoot)
	scope := analysis.SourceScope{
		PolicyVersion:               policy.PolicyVersion,
		InvocationRoot:              policy.InvocationRoot,
		ProjectRoot:                 root.RelativePath,
		NestedRootExclusions:        append([]string(nil), root.NestedRootExclusions...),
		IncludeGlobs:                append([]string(nil), include...),
		ExcludeGlobs:                allExcludes,
		MatchedPaths:                matched,
		MatchedLocalPaths:           matchedLocal,
		ExcludedPaths:               excluded,
		PolicyFingerprint:           policyHash,
		MatchedSourceSetFingerprint: matchedHash,
	}
	relevantExcludes := []string{}
	for _, candidate := range append(append([]string(nil), policy.Exclude...), analyzerExcludes...) {
		for _, repositoryPath := range root.OwnedSourcePaths {
			invocationPath := relativeToInvocation(policy.InvocationRoot, repositoryPath)
			localPath := relativeToProject(root.RelativePath, repositoryPath)
			if matchesAnyGlob([]string{candidate}, invocationPath) || matchesAnyGlob([]string{candidate}, localPath) {
				relevantExcludes = append(relevantExcludes, candidate)
				break
			}
		}
	}
	identityData, _ := json.Marshal(struct {
		PolicyVersion    string   `json:"policy_version"`
		AnalyzerID       string   `json:"analyzer_id"`
		Root             string   `json:"root"`
		InvocationRoot   string   `json:"invocation_root"`
		Include          []string `json:"include"`
		Matched          []string `json:"matched"`
		Excluded         []string `json:"excluded"`
		RelevantExcludes []string `json:"relevant_excludes"`
	}{policy.PolicyVersion, analyzerID, root.RelativePath, policy.InvocationRoot, include, matched, excluded, uniqueStrings(relevantExcludes)})
	return scope, sha256Fingerprint(identityData), nil
}

func optionGlobValues(options analysis.EffectiveOptions) ([]string, error) {
	value, exists := options.Values["exclude"]
	if !exists || value == nil {
		return []string{}, nil
	}
	values, ok := value.([]string)
	if !ok {
		return nil, scopeFilterError("analyzer exclusion option must be a string array", map[string]any{"type": fmt.Sprintf("%T", value)})
	}
	return normalizeGlobList(values)
}

func matchesAnyGlob(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if globMatches(pattern, value) {
			return true
		}
	}
	return false
}
