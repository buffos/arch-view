package live

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/quality"
)

const (
	ErrorLiveConfigInvalid             = "LiveConfigInvalid"
	ErrorWatchRootInvalid              = "WatchRootInvalid"
	ErrorWatchBackendUnavailable       = "WatchBackendUnavailable"
	ErrorSourceReconciliation          = "SourceReconciliationFailed"
	ErrorInputUnstable                 = "InputUnstable"
	ErrorSnapshotBuildFailed           = "SnapshotBuildFailed"
	ErrorSnapshotValidation            = "SnapshotValidationFailed"
	ErrorNoReadySnapshot               = "NoReadySnapshot"
	ErrorRevisionUnavailable           = "RevisionUnavailable"
	ErrorAnalysisWaitTimeout           = "AnalysisWaitTimeout"
	ErrorQueryBudget                   = "QueryBudgetExceeded"
	ErrorQueryCursor                   = "QueryCursorInvalid"
	ErrorSourceContextOutOfScope       = "SourceContextOutOfScope"
	ErrorQualityEvaluation             = "QualityEvaluationInvalid"
	ErrorQualityPolicyPermissionDenied = "quality_policy_permission_denied"
	ErrorQualityProfileConflict        = "quality_profile_conflict"
	ErrorBaselineRevisionStale         = "baseline_revision_stale"
	ErrorBaselineNotFound              = "quality_baseline_not_found"
	ErrorQualityPolicyIncompatible     = "quality_policy_incompatible"
)

type ValidationDependencies struct {
	AnalyzerRegistry *analysis.Registry
	QualityCatalog   *quality.Catalog
	Profiles         QualityProfileResolver
}

type ValidatedLiveSession struct {
	Config         LiveSessionConfig
	RepositoryRoot string
}

func newLiveError(code, message string, details map[string]any) *QueryError {
	return &QueryError{Code: code, Message: message, Details: details}
}

// ValidateLiveSession validates the versioned policy before a watcher,
// analyzer, source reader, or query surface receives session access.
func ValidateLiveSession(ctx context.Context, config LiveSessionConfig, openedRepositoryRoot string, dependencies ValidationDependencies) (ValidatedLiveSession, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ValidatedLiveSession{}, err
	}
	config = cloneLiveSessionConfig(config)
	if config.SchemaVersion != LiveSchemaVersion {
		return ValidatedLiveSession{}, newLiveError(ErrorLiveConfigInvalid, "live session schema version is unsupported", map[string]any{"schema_version": config.SchemaVersion, "expected": LiveSchemaVersion})
	}
	if !safeIdentifier(config.SessionID) {
		return ValidatedLiveSession{}, newLiveError(ErrorLiveConfigInvalid, "session_id is required and must be a safe opaque identifier", nil)
	}
	root, err := normalizeOpenedRoot(openedRepositoryRoot)
	if err != nil {
		return ValidatedLiveSession{}, err
	}
	repositoryRoot, err := resolveContainedRelative(root, config.RepositoryRoot, true)
	if err != nil {
		return ValidatedLiveSession{}, newLiveError(ErrorWatchRootInvalid, "repository_root must be a repository-relative path inside the opened repository", map[string]any{"path": config.RepositoryRoot})
	}
	config.RepositoryRoot = repositoryRoot.relative
	config.WatchRoots, err = normalizeWatchRoots(repositoryRoot.absolute, config.WatchRoots)
	if err != nil {
		return ValidatedLiveSession{}, err
	}
	if err := validateAnalyzerIDs(config.AnalyzerIDs, dependencies.AnalyzerRegistry); err != nil {
		return ValidatedLiveSession{}, err
	}
	if err := normalizeSourceRequest(&config.SourceIndexRequest); err != nil {
		return ValidatedLiveSession{}, err
	}
	if err := validateQualityReference(ctx, config.QualityRequest, dependencies.Profiles, dependencies.QualityCatalog); err != nil {
		return ValidatedLiveSession{}, err
	}
	normalizePolicies(&config)
	if err := validatePolicies(config); err != nil {
		return ValidatedLiveSession{}, err
	}
	if err := validateExtensions(config.Extensions); err != nil {
		return ValidatedLiveSession{}, err
	}
	return ValidatedLiveSession{Config: config, RepositoryRoot: repositoryRoot.absolute}, nil
}

func cloneLiveSessionConfig(value LiveSessionConfig) LiveSessionConfig {
	data, err := json.Marshal(value)
	if err == nil {
		var clone LiveSessionConfig
		if json.Unmarshal(data, &clone) == nil {
			return clone
		}
	}
	clone := value
	clone.WatchRoots = append([]WatchRoot(nil), value.WatchRoots...)
	clone.AnalyzerIDs = append([]string(nil), value.AnalyzerIDs...)
	clone.SourceIndexRequest.Capabilities = append([]string(nil), value.SourceIndexRequest.Capabilities...)
	clone.PermissionPolicy.AllowedOperations = append([]Operation(nil), value.PermissionPolicy.AllowedOperations...)
	clone.Extensions = append([]ExtensionBlock(nil), value.Extensions...)
	if value.QualityRequest != nil {
		qualityRequest := *value.QualityRequest
		clone.QualityRequest = &qualityRequest
	}
	return clone
}

func (session *LiveSession) requireOperation(operation Operation) error {
	if session == nil {
		return newLiveError(ErrorLiveConfigInvalid, "live session is nil", nil)
	}
	for _, allowed := range session.validated.Config.PermissionPolicy.AllowedOperations {
		if allowed == operation {
			return nil
		}
	}
	return newLiveError("mcp_permission_denied", "the requested live operation is not enabled for this session", map[string]any{"operation": operation})
}

func normalizeOpenedRoot(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", newLiveError(ErrorLiveConfigInvalid, "opened repository root is required", nil)
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", newLiveError(ErrorLiveConfigInvalid, "opened repository root could not be normalized", map[string]any{"error": err.Error()})
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", newLiveError(ErrorLiveConfigInvalid, "opened repository root must be a readable directory", map[string]any{"root": value})
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", newLiveError(ErrorLiveConfigInvalid, "opened repository root symlinks could not be resolved", map[string]any{"root": value, "error": err.Error()})
	}
	return filepath.Clean(resolved), nil
}

type containedPath struct {
	relative string
	absolute string
}

func resolveContainedRelative(root, value string, allowDot bool) (containedPath, error) {
	normalized, err := normalizeRelativePath(value, allowDot)
	if err != nil {
		return containedPath{}, err
	}
	candidate := filepath.Clean(filepath.Join(root, filepath.FromSlash(normalized)))
	if !pathWithin(root, candidate) {
		return containedPath{}, fmt.Errorf("path escapes root")
	}
	info, err := os.Stat(candidate)
	if err != nil || !info.IsDir() {
		return containedPath{}, fmt.Errorf("path is not a directory")
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return containedPath{}, fmt.Errorf("path symlinks could not be resolved: %w", err)
	}
	if !pathWithin(root, resolved) {
		return containedPath{}, fmt.Errorf("path symlink escapes root")
	}
	return containedPath{relative: normalized, absolute: filepath.Clean(resolved)}, nil
}

func normalizeWatchRoots(repositoryRoot string, roots []WatchRoot) ([]WatchRoot, error) {
	if len(roots) == 0 {
		roots = []WatchRoot{{Path: ".", Recursive: true}}
	}
	result := make([]WatchRoot, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		normalized, err := normalizeRelativePath(root.Path, true)
		if err != nil {
			return nil, newLiveError(ErrorWatchRootInvalid, "watch root must be repository-relative", map[string]any{"path": root.Path})
		}
		candidate := filepath.Clean(filepath.Join(repositoryRoot, filepath.FromSlash(normalized)))
		if !pathWithin(repositoryRoot, candidate) {
			return nil, newLiveError(ErrorWatchRootInvalid, "watch root escapes repository root", map[string]any{"path": root.Path})
		}
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			return nil, newLiveError(ErrorWatchRootInvalid, "watch root must be an existing directory", map[string]any{"path": normalized})
		}
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil || !pathWithin(repositoryRoot, resolved) {
			return nil, newLiveError(ErrorWatchRootInvalid, "watch root symlink or junction escapes repository root", map[string]any{"path": normalized})
		}
		if _, exists := seen[normalized]; exists {
			return nil, newLiveError(ErrorLiveConfigInvalid, "watch roots must be unique", map[string]any{"path": normalized})
		}
		seen[normalized] = struct{}{}
		result = append(result, WatchRoot{Path: normalized, Recursive: root.Recursive})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

func normalizeRelativePath(value string, allowDot bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		if allowDot {
			return ".", nil
		}
		return "", fmt.Errorf("path is required")
	}
	if strings.ContainsAny(value, "\x00\r\n") || filepath.IsAbs(value) || path.IsAbs(strings.ReplaceAll(value, "\\", "/")) || filepath.VolumeName(value) != "" {
		return "", fmt.Errorf("path is absolute or contains unsafe characters")
	}
	normalized := path.Clean(strings.ReplaceAll(value, "\\", "/"))
	if normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", fmt.Errorf("path traverses outside root")
	}
	if normalized == "." && !allowDot {
		return "", fmt.Errorf("path is not allowed")
	}
	return normalized, nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func validateAnalyzerIDs(ids []string, registry *analysis.Registry) error {
	if len(ids) > 0 && registry == nil {
		return newLiveError(ErrorLiveConfigInvalid, "analyzer_ids require an analyzer registry for validation", nil)
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if !safeIdentifier(id) {
			return newLiveError(ErrorLiveConfigInvalid, "analyzer_ids must contain safe analyzer identities", map[string]any{"analyzer_id": id})
		}
		if _, exists := seen[id]; exists {
			return newLiveError(ErrorLiveConfigInvalid, "analyzer_ids must be unique", map[string]any{"analyzer_id": id})
		}
		seen[id] = struct{}{}
		if registry != nil {
			if _, ok := registry.Get(id); !ok {
				return newLiveError(ErrorLiveConfigInvalid, "requested analyzer is not registered", map[string]any{"analyzer_id": id})
			}
		}
	}
	return nil
}

func normalizeSourceRequest(request *SourceIndexRequest) error {
	if request == nil {
		return newLiveError(ErrorLiveConfigInvalid, "source_index_request is required", nil)
	}
	if request.Enabled && len(request.Capabilities) == 0 {
		request.Capabilities = []string{"source:files", "source:size", "source:declarations", "source:documentation"}
	}
	seen := make(map[string]struct{}, len(request.Capabilities))
	for index, capability := range request.Capabilities {
		capability = strings.TrimSpace(capability)
		if !safeNamespaced(capability) {
			return newLiveError(ErrorLiveConfigInvalid, "source capability requests must be safe namespaced identifiers", map[string]any{"index": index, "capability": capability})
		}
		if _, exists := seen[capability]; exists {
			return newLiveError(ErrorLiveConfigInvalid, "source capability requests must be unique", map[string]any{"capability": capability})
		}
		seen[capability] = struct{}{}
		request.Capabilities[index] = capability
	}
	sort.Strings(request.Capabilities)
	return nil
}

func validateQualityReference(ctx context.Context, request *QualityRequest, profiles QualityProfileResolver, catalog *quality.Catalog) error {
	if request == nil {
		return nil
	}
	if !safeNamespaced(request.ProfileID) || !strings.HasPrefix(request.ProfileID, "profile:") || !safeVersion(request.ProfileVersion) {
		return newLiveError(ErrorLiveConfigInvalid, "quality profile reference is invalid", map[string]any{"profile_id": request.ProfileID, "profile_version": request.ProfileVersion})
	}
	if profiles == nil || catalog == nil {
		return newLiveError(ErrorLiveConfigInvalid, "quality profile references require a profile resolver and quality catalog", nil)
	}
	profile, err := profiles.ResolveProfile(ctx, request.ProfileID, request.ProfileVersion)
	if err != nil {
		return newLiveError(ErrorLiveConfigInvalid, "quality profile reference could not be resolved", map[string]any{"profile_id": request.ProfileID, "profile_version": request.ProfileVersion, "error": err.Error()})
	}
	if _, err := quality.ValidateQualityProfile(profile, catalog); err != nil {
		return newLiveError(ErrorLiveConfigInvalid, "quality profile reference is invalid", map[string]any{"profile_id": request.ProfileID, "profile_version": request.ProfileVersion, "error": err.Error()})
	}
	return nil
}

func normalizePolicies(config *LiveSessionConfig) {
	if config.WatchPolicy.DebounceMS < 0 {
		return
	}
	if config.WatchPolicy.MaxPendingEvents == 0 {
		config.WatchPolicy.MaxPendingEvents = DefaultMaxPendingEvents
	}
	if config.WatchPolicy.MaxParallelScopes == 0 {
		config.WatchPolicy.MaxParallelScopes = DefaultMaxParallelScopes
	}
	if config.FreshnessPolicy.DefaultConsistency == "" {
		config.FreshnessPolicy.DefaultConsistency = ConsistencyLatestReady
	}
	if config.FreshnessPolicy.MaxWaitMS == 0 {
		config.FreshnessPolicy.MaxWaitMS = int(DefaultFreshnessMaxWait / time.Millisecond)
	}
	if config.FreshnessPolicy.MaxStabilityRetries == 0 {
		config.FreshnessPolicy.MaxStabilityRetries = DefaultStabilityRetries
	}
	if config.QueryPolicy.DefaultMaxBytes == 0 {
		config.QueryPolicy.DefaultMaxBytes = DefaultQueryMaxBytes
	}
	if config.QueryPolicy.DefaultMaxItems == 0 {
		config.QueryPolicy.DefaultMaxItems = DefaultQueryMaxItems
	}
	if config.QueryPolicy.HardMaxBytes == 0 {
		config.QueryPolicy.HardMaxBytes = DefaultQueryHardMaxBytes
	}
	if config.QueryPolicy.HardMaxItems == 0 {
		config.QueryPolicy.HardMaxItems = DefaultQueryHardMaxItems
	}
	if config.QueryPolicy.DefaultContextLines == 0 {
		config.QueryPolicy.DefaultContextLines = DefaultContextLines
	}
	if config.QueryPolicy.HardContextLines == 0 {
		config.QueryPolicy.HardContextLines = DefaultContextHardLines
	}
	if config.PermissionPolicy.DefaultMode == "" {
		config.PermissionPolicy.DefaultMode = "read_only"
	}
	if len(config.PermissionPolicy.AllowedOperations) == 0 {
		config.PermissionPolicy.AllowedOperations = []Operation{OperationStatus, OperationSearch, OperationEvidence, OperationSourceContext, OperationEnsureCurrent, OperationQualityEvaluate, OperationQualityProfileRead, OperationBaselineRead}
	}
	if config.PermissionPolicy.Transport == "" {
		config.PermissionPolicy.Transport = "stdio"
	}
	config.PermissionPolicy.AuditPolicyWrites = true
}

func validatePolicies(config LiveSessionConfig) error {
	if config.WatchPolicy.DebounceMS < 0 || config.WatchPolicy.DebounceMS > 60_000 || config.WatchPolicy.MaxPendingEvents < 1 || config.WatchPolicy.MaxPendingEvents > 1_000_000 || config.WatchPolicy.MaxParallelScopes < 1 || config.WatchPolicy.MaxParallelScopes > 64 || config.WatchPolicy.RescanIntervalMS < 0 {
		return newLiveError(ErrorLiveConfigInvalid, "watch policy is outside the supported bounds", nil)
	}
	freshness := config.FreshnessPolicy
	if freshness.DefaultConsistency != ConsistencyLatestReady && freshness.DefaultConsistency != ConsistencyRequireCurrent || freshness.SettleMS < 0 || freshness.SettleMS > 60_000 || freshness.MaxWaitMS < 0 || freshness.MaxWaitMS > 120_000 || freshness.MaxStabilityRetries < 1 || freshness.MaxStabilityRetries > 10 || freshness.ReconcileIntervalMS < 0 {
		return newLiveError(ErrorLiveConfigInvalid, "freshness policy is outside the supported bounds", nil)
	}
	query := config.QueryPolicy
	if query.DefaultMaxBytes < 1 || query.DefaultMaxItems < 1 || query.HardMaxBytes < query.DefaultMaxBytes || query.HardMaxBytes > 16<<20 || query.HardMaxItems < query.DefaultMaxItems || query.HardMaxItems > 50_000 || query.DefaultContextLines < 1 || query.HardContextLines < query.DefaultContextLines || query.HardContextLines > 10_000 {
		return newLiveError(ErrorLiveConfigInvalid, "query policy is outside the supported bounds", nil)
	}
	permission := config.PermissionPolicy
	if permission.DefaultMode != "read_only" || permission.Transport != "stdio" && permission.Transport != "local_http" && permission.Transport != "authenticated_http" || permission.AllowShell || permission.AllowTargetExecution {
		return newLiveError(ErrorLiveConfigInvalid, "permission policy must remain read-only and transport-safe", nil)
	}
	seen := make(map[Operation]struct{}, len(permission.AllowedOperations))
	for _, operation := range permission.AllowedOperations {
		if !validOperation(operation) {
			return newLiveError(ErrorLiveConfigInvalid, "permission policy contains an unsupported operation", map[string]any{"operation": operation})
		}
		if _, exists := seen[operation]; exists {
			return newLiveError(ErrorLiveConfigInvalid, "permission policy operations must be unique", map[string]any{"operation": operation})
		}
		seen[operation] = struct{}{}
	}
	return nil
}

func validateExtensions(values []ExtensionBlock) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !safeNamespaced(value.Namespace) || !safeVersion(value.SchemaVersion) || !safeNamespaced(value.Capability) {
			return newLiveError(ErrorLiveConfigInvalid, "extension identity is invalid", nil)
		}
		if _, err := json.Marshal(value.Payload); err != nil {
			return newLiveError(ErrorLiveConfigInvalid, "extension payload must be JSON-serializable", map[string]any{"namespace": value.Namespace})
		}
		key := value.Namespace + "\x00" + value.SchemaVersion + "\x00" + value.Capability
		if _, exists := seen[key]; exists {
			return newLiveError(ErrorLiveConfigInvalid, "extensions must be unique", map[string]any{"key": key})
		}
		seen[key] = struct{}{}
	}
	return nil
}

func safeIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && value == strings.TrimSpace(value) && !strings.ContainsAny(value, "\x00\r\n/\\")
}

func safeNamespaced(value string) bool {
	if !safeIdentifier(value) || !strings.Contains(value, ":") || strings.ContainsAny(value, "/\\") {
		return false
	}
	return true
}

func safeVersion(value string) bool {
	return safeIdentifier(value) && !strings.ContainsAny(value, " /\\")
}

func validOperation(value Operation) bool {
	switch value {
	case OperationStatus, OperationSearch, OperationEvidence, OperationSourceContext, OperationEnsureCurrent, OperationQualityEvaluate, OperationQualityProfileRead, OperationQualityProfileWrite, OperationBaselineRead, OperationBaselineWrite:
		return true
	default:
		return false
	}
}
