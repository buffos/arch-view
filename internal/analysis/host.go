package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Host struct {
	registry *Registry
	runtime  RuntimeSelection
}

func NewHost(registry *Registry) *Host {
	return NewHostWithRuntime(registry, RuntimeSelection{})
}

// NewHostWithRuntime creates a host with explicit runtime provenance. The
// registry still owns analyzer selection; this value only records the trusted
// boundary through which the registry was assembled.
func NewHostWithRuntime(registry *Registry, runtimeSelection RuntimeSelection) *Host {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Host{registry: registry, runtime: runtimeSelection}
}

func (h *Host) ListManifests() []Manifest {
	if h == nil || h.registry == nil {
		return nil
	}
	return h.registry.ListManifests()
}

// RegistrySnapshot returns a defensive registry copy for application services
// such as multi-analyzer planning. The host retains ownership of its active
// registry; callers cannot mutate it through the snapshot.
func (h *Host) RegistrySnapshot() *Registry {
	if h == nil || h.registry == nil {
		return NewRegistry()
	}
	registry := NewRegistry()
	for _, analyzer := range h.registry.List() {
		_ = registry.Register(analyzer)
	}
	return registry
}

// Runtime returns the host's runtime provenance configuration.
func (h *Host) Runtime() RuntimeSelection {
	if h == nil {
		return RuntimeSelection{}
	}
	return h.runtime
}

// WithRuntime returns a host over the same analyzers with different runtime
// provenance. The registry copy keeps later registrations isolated.
func (h *Host) WithRuntime(selection RuntimeSelection) (*Host, error) {
	if h == nil || h.registry == nil {
		return nil, NewHostError(ErrHostFailure, "analysis host is not initialized", nil)
	}
	if err := validateRuntimeSelection(selection); err != nil {
		return nil, err
	}
	registry := NewRegistry()
	for _, analyzer := range h.registry.List() {
		if err := registry.Register(analyzer); err != nil {
			return nil, err
		}
	}
	return NewHostWithRuntime(registry, selection), nil
}

// Register adds an analyzer to the host registry. Callers that load optional
// analyzers can validate and register them before any selection or analysis
// request is run.
func (h *Host) Register(analyzer Analyzer) error {
	if h == nil {
		return NewHostError(ErrInvalidManifest, "cannot register an analyzer on a nil host", nil)
	}
	return h.registry.Register(analyzer)
}

func (h *Host) Run(ctx context.Context, request RunRequest) (result AnalysisResult, err error) {
	if h == nil || h.registry == nil {
		return AnalysisResult{}, NewHostError(ErrHostFailure, "analysis host is not initialized", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateRuntimeSelection(h.runtime); err != nil {
		return AnalysisResult{}, err
	}
	if err := validateProjectRoot(request.ProjectRoot); err != nil {
		return AnalysisResult{}, err
	}
	root, err := normalizeProjectRoot(request.ProjectRoot)
	if err != nil {
		return AnalysisResult{}, err
	}
	if ctx.Err() != nil {
		return AnalysisResult{}, NewHostError(ErrCancelled, "analysis was cancelled before selection", nil)
	}

	var selected Analyzer
	var selection AnalyzerSelection
	defer func() {
		if recovered := recover(); recovered != nil {
			result = AnalysisResult{}
			err = NewHostError(ErrAnalyzerFailed, "analyzer panicked during host execution", map[string]any{"panic": fmt.Sprintf("%T: %v", recovered, recovered)})
		}
	}()

	selected, selection, err = h.selectAnalyzer(ctx, root, request)
	if err != nil {
		return AnalysisResult{}, err
	}
	if ctx.Err() != nil {
		return AnalysisResult{}, NewHostError(ErrCancelled, "analysis was cancelled before execution", nil)
	}
	manifest := selected.Manifest()
	effectiveOptions, err := ResolveOptions(manifest, request.ProjectOptions, request.CLIOptions)
	if err != nil {
		return AnalysisResult{}, err
	}
	return h.runSelected(ctx, root, selected, selection, effectiveOptions, request.SourceScope, request.SourceIndexRequest)
}

// RunPlanned executes a planner-resolved analyzer selection. It deliberately
// does not run Detect or resolve options again: doing so could change an
// immutable plan between scheduling and execution. All normal host validation
// and runtime provenance checks still apply.
func (h *Host) RunPlanned(ctx context.Context, request PlannedRunRequest) (result AnalysisResult, err error) {
	if h == nil || h.registry == nil {
		return AnalysisResult{}, NewHostError(ErrHostFailure, "analysis host is not initialized", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateRuntimeSelection(h.runtime); err != nil {
		return AnalysisResult{}, err
	}
	if err := validateProjectRoot(request.ProjectRoot); err != nil {
		return AnalysisResult{}, err
	}
	root, err := normalizeProjectRoot(request.ProjectRoot)
	if err != nil {
		return AnalysisResult{}, err
	}
	if strings.TrimSpace(request.AnalyzerID) == "" {
		return AnalysisResult{}, NewHostError(ErrInvalidRequest, "planned analyzer id is required", nil)
	}
	selected, ok := h.registry.Get(request.AnalyzerID)
	if !ok {
		return AnalysisResult{}, NewHostError(ErrNoAnalyzer, "planned analyzer is not registered", map[string]any{"analyzer_id": request.AnalyzerID})
	}
	if request.Selection.AnalyzerID != "" && request.Selection.AnalyzerID != request.AnalyzerID {
		return AnalysisResult{}, NewHostError(ErrInvalidRequest, "planned selection does not match the analyzer id", map[string]any{"analyzer_id": request.AnalyzerID, "selection_analyzer_id": request.Selection.AnalyzerID})
	}
	selection := request.Selection
	if selection.AnalyzerID == "" {
		selection.AnalyzerID = request.AnalyzerID
	}
	if selection.Mode == "" {
		selection.Mode = "planned"
	}
	selection = h.decorateSelection(selection)
	if request.Options.Fingerprint == "" {
		return AnalysisResult{}, NewHostError(ErrInvalidOptions, "planned analyzer options fingerprint is required", map[string]any{"analyzer_id": request.AnalyzerID})
	}
	return h.runSelected(ctx, root, selected, selection, request.Options, request.SourceScope, request.SourceIndexRequest)
}

func (h *Host) runSelected(ctx context.Context, root string, selected Analyzer, selection AnalyzerSelection, effectiveOptions EffectiveOptions, sourceScope *SourceScope, sourceIndexRequest *SourceIndexRequest) (result AnalysisResult, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			result = AnalysisResult{}
			err = NewHostError(ErrAnalyzerFailed, "analyzer panicked during host execution", map[string]any{"panic": fmt.Sprintf("%T: %v", recovered, recovered)})
		}
	}()
	if ctx.Err() != nil {
		return AnalysisResult{}, NewHostError(ErrCancelled, "analysis was cancelled before execution", nil)
	}
	manifest := selected.Manifest()
	analyzeRequest := AnalyzeRequest{
		ProjectRoot:        root,
		Selection:          selection,
		Options:            effectiveOptions,
		SourceScope:        sourceScope,
		SourceIndexRequest: cloneSourceIndexRequest(sourceIndexRequest),
	}
	result, err = selected.Analyze(ctx, analyzeRequest)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return AnalysisResult{}, NewHostError(ErrCancelled, "analysis was cancelled", nil)
		}
		var hostErr *HostError
		if errors.As(err, &hostErr) {
			return AnalysisResult{}, err
		}
		return AnalysisResult{}, WrapHostError(ErrAnalyzerFailed, "analyzer execution failed", err, map[string]any{"analyzer_id": manifest.ID})
	}
	if ctx.Err() != nil {
		return AnalysisResult{}, NewHostError(ErrCancelled, "analysis was cancelled", nil)
	}
	if err := validateResultRuntime(result.Analyzer, h.runtime); err != nil {
		return AnalysisResult{}, err
	}
	if result.Analyzer.ID == "" {
		result.Analyzer = analyzerInfo(manifest)
	}
	result.Analyzer.RuntimeMode = h.runtime.Mode
	result.Analyzer.RuntimeSource = h.runtime.Source
	result.Analyzer.RuntimePlatform = h.runtime.Platform
	if result.RunID == "" {
		result.RunID = deterministicRunID(root, manifest.ID, effectiveOptions.Fingerprint)
	}
	if result.OptionsFingerprint != "" && result.OptionsFingerprint != effectiveOptions.Fingerprint {
		return AnalysisResult{}, NewHostError(ErrResultInvalid, "analysis result options fingerprint does not match the host-resolved options", map[string]any{"expected": effectiveOptions.Fingerprint, "actual": result.OptionsFingerprint})
	}
	if result.OptionsFingerprint == "" {
		result.OptionsFingerprint = effectiveOptions.Fingerprint
	}
	if result.Project.RootLabel == "" {
		result.Project.RootLabel = projectRootLabel(root)
	}
	normalizeResultCollections(&result)
	if result.Summary == (AnalysisSummary{}) {
		result.Summary = ComputeSummary(result)
	}
	if err := ValidateAnalysisResult(result, manifest, root); err != nil {
		return AnalysisResult{}, err
	}
	result.Summary = ComputeSummary(result)
	return result, nil
}

func (h *Host) selectAnalyzer(ctx context.Context, root string, request RunRequest) (Analyzer, AnalyzerSelection, error) {
	if request.AnalyzerID != "" && request.Language != "" {
		analyzer, ok := h.registry.Get(request.AnalyzerID)
		if !ok {
			return nil, AnalyzerSelection{}, NewHostError(ErrNoAnalyzer, "requested analyzer is not registered", map[string]any{"analyzer_id": request.AnalyzerID})
		}
		if analyzer.Manifest().Language != strings.ToLower(request.Language) {
			return nil, AnalyzerSelection{}, NewHostError(ErrInvalidRequest, "analyzer id and language select different analyzers", map[string]any{"analyzer_id": request.AnalyzerID, "language": request.Language})
		}
	}

	if request.AnalyzerID != "" {
		analyzer, ok := h.registry.Get(request.AnalyzerID)
		if !ok {
			return nil, AnalyzerSelection{}, NewHostError(ErrNoAnalyzer, "requested analyzer is not registered", map[string]any{"analyzer_id": request.AnalyzerID})
		}
		selected, selection, err := h.confirmExplicitSelection(ctx, root, analyzer, "explicit-id")
		return selected, h.decorateSelection(selection), err
	}
	if request.Language != "" {
		analyzers := h.registry.ByLanguage(request.Language)
		if len(analyzers) == 0 {
			return nil, AnalyzerSelection{}, NewHostError(ErrNoAnalyzer, "no analyzer is registered for the requested language", map[string]any{"language": request.Language})
		}
		if len(analyzers) > 1 {
			ids := make([]string, 0, len(analyzers))
			for _, analyzer := range analyzers {
				ids = append(ids, analyzer.Manifest().ID)
			}
			return nil, AnalyzerSelection{}, NewHostError(ErrAmbiguousAnalyzer, "language maps to more than one registered analyzer", map[string]any{"language": request.Language, "analyzer_ids": ids})
		}
		selected, selection, err := h.confirmExplicitSelection(ctx, root, analyzers[0], "explicit-language")
		return selected, h.decorateSelection(selection), err
	}

	type detected struct {
		analyzer  Analyzer
		candidate DetectionCandidate
	}
	candidates := make([]detected, 0)
	for _, analyzer := range h.registry.List() {
		candidate, err := analyzer.Detect(ctx, DetectRequest{ProjectRoot: root})
		if err != nil {
			if ctx.Err() != nil {
				return nil, AnalyzerSelection{}, NewHostError(ErrCancelled, "analysis was cancelled during detection", nil)
			}
			if h.runtime.Mode == RuntimeModePackaged && IsAnalyzerPackageError(err) {
				return nil, AnalyzerSelection{}, err
			}
			continue
		}
		candidate, err = ensureCandidate(candidate, analyzer.Manifest())
		if err != nil {
			return nil, AnalyzerSelection{}, err
		}
		if candidate.Confidence > 0 {
			candidates = append(candidates, detected{analyzer: analyzer, candidate: candidate})
		}
	}
	if len(candidates) == 0 {
		return nil, AnalyzerSelection{}, NewHostError(ErrNoAnalyzer, "no registered analyzer detected the project", map[string]any{"project_root": root})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].candidate.Confidence == candidates[j].candidate.Confidence {
			return candidates[i].analyzer.Manifest().ID < candidates[j].analyzer.Manifest().ID
		}
		return candidates[i].candidate.Confidence > candidates[j].candidate.Confidence
	})
	if len(candidates) > 1 && candidates[0].candidate.Confidence == candidates[1].candidate.Confidence {
		ids := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			if candidate.candidate.Confidence != candidates[0].candidate.Confidence {
				break
			}
			ids = append(ids, candidate.analyzer.Manifest().ID)
		}
		return nil, AnalyzerSelection{}, NewHostError(ErrAmbiguousAnalyzer, "multiple analyzers have the same highest detection confidence", map[string]any{"analyzer_ids": ids})
	}
	chosen := candidates[0]
	return chosen.analyzer, h.decorateSelection(AnalyzerSelection{
		AnalyzerID:     chosen.analyzer.Manifest().ID,
		Mode:           "auto",
		Confidence:     chosen.candidate.Confidence,
		MatchedMarkers: chosen.candidate.MatchedMarkers,
		Reason:         chosen.candidate.Reason,
		BoundaryHint:   chosen.candidate.BoundaryHint,
	}), nil
}

func (h *Host) confirmExplicitSelection(ctx context.Context, root string, analyzer Analyzer, mode string) (Analyzer, AnalyzerSelection, error) {
	candidate, err := analyzer.Detect(ctx, DetectRequest{ProjectRoot: root})
	if err != nil {
		if ctx.Err() != nil {
			return nil, AnalyzerSelection{}, NewHostError(ErrCancelled, "analysis was cancelled during explicit detection", nil)
		}
		var hostErr *HostError
		if errors.As(err, &hostErr) {
			return nil, AnalyzerSelection{}, err
		}
		return nil, AnalyzerSelection{}, WrapHostError(ErrUnsupportedProject, "selected analyzer could not inspect the project", err, map[string]any{"analyzer_id": analyzer.Manifest().ID})
	}
	candidate, err = ensureCandidate(candidate, analyzer.Manifest())
	if err != nil {
		return nil, AnalyzerSelection{}, err
	}
	if candidate.Confidence <= 0 {
		return nil, AnalyzerSelection{}, NewHostError(ErrUnsupportedProject, "selected analyzer is incompatible with the project root", map[string]any{"analyzer_id": analyzer.Manifest().ID})
	}
	return analyzer, AnalyzerSelection{
		AnalyzerID:     analyzer.Manifest().ID,
		Mode:           mode,
		Confidence:     candidate.Confidence,
		MatchedMarkers: candidate.MatchedMarkers,
		Reason:         candidate.Reason,
		BoundaryHint:   candidate.BoundaryHint,
	}, nil
}

func (h *Host) decorateSelection(selection AnalyzerSelection) AnalyzerSelection {
	if h == nil {
		return selection
	}
	selection.RuntimeMode = h.runtime.Mode
	selection.RuntimeSource = h.runtime.Source
	selection.RuntimePlatform = h.runtime.Platform
	return selection
}

func validateRuntimeSelection(selection RuntimeSelection) error {
	if selection.Mode == "" {
		if selection.Source != "" || selection.Platform != "" {
			return NewHostError(ErrInvalidRequest, "analyzer runtime mode is required when provenance is set", nil)
		}
		return nil
	}
	switch selection.Mode {
	case RuntimeModePackaged, RuntimeModeInProcess, RuntimeModeExplicit:
		if strings.TrimSpace(selection.Source) == "" || strings.TrimSpace(selection.Source) != selection.Source ||
			strings.TrimSpace(selection.Platform) == "" || strings.TrimSpace(selection.Platform) != selection.Platform {
			return NewHostError(ErrInvalidRequest, "analyzer runtime source and platform are required", map[string]any{"runtime_mode": selection.Mode})
		}
		if strings.ContainsAny(selection.Source+selection.Platform, "\x00\r\n") {
			return NewHostError(ErrInvalidRequest, "analyzer runtime provenance contains unsafe control characters", map[string]any{"runtime_mode": selection.Mode})
		}
		return nil
	default:
		return NewHostError(ErrInvalidRequest, "analyzer runtime mode is unsupported", map[string]any{"runtime_mode": selection.Mode})
	}
}

func validateResultRuntime(info AnalyzerInfo, expected RuntimeSelection) error {
	if info.RuntimeMode != "" && info.RuntimeMode != expected.Mode {
		return NewHostError(ErrResultInvalid, "analysis result runtime mode does not match the selected runtime", map[string]any{
			"expected": expected.Mode,
			"actual":   info.RuntimeMode,
		})
	}
	if info.RuntimeSource != "" && info.RuntimeSource != expected.Source {
		return NewHostError(ErrResultInvalid, "analysis result runtime source does not match the selected runtime", map[string]any{
			"expected": expected.Source,
			"actual":   info.RuntimeSource,
		})
	}
	if info.RuntimePlatform != "" && info.RuntimePlatform != expected.Platform {
		return NewHostError(ErrResultInvalid, "analysis result runtime platform does not match the selected runtime", map[string]any{
			"expected": expected.Platform,
			"actual":   info.RuntimePlatform,
		})
	}
	return nil
}

func validateProjectRoot(root string) error {
	if strings.TrimSpace(root) == "" {
		return NewHostError(ErrInvalidRequest, "project root is required", nil)
	}
	return nil
}

func normalizeProjectRoot(input string) (string, error) {
	absolute, err := filepath.Abs(input)
	if err != nil {
		return "", WrapHostError(ErrInvalidRequest, "project root could not be normalized", err, nil)
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		if os.IsNotExist(err) {
			return "", NewHostError(ErrUnreadableProject, "project root does not exist", map[string]any{"project_root": input})
		}
		return "", WrapHostError(ErrUnreadableProject, "project root could not be read", err, map[string]any{"project_root": input})
	}
	if !info.IsDir() {
		return "", NewHostError(ErrInvalidRequest, "project root must be a directory", map[string]any{"project_root": input})
	}
	return absolute, nil
}

func projectRootLabel(root string) string {
	clean := filepath.Clean(root)
	label := filepath.Base(clean)
	if label == string(filepath.Separator) || label == "." || label == "" {
		return clean
	}
	return label
}

func deterministicRunID(root, analyzerID, fingerprint string) string {
	payload := root + "\x00" + analyzerID + "\x00" + fingerprint
	sum := sha256.Sum256([]byte(payload))
	return "run-" + hex.EncodeToString(sum[:8])
}

func MarshalError(err error) ([]byte, error) {
	var hostErr *HostError
	if errors.As(err, &hostErr) {
		data, marshalErr := json.MarshalIndent(struct {
			Error *HostError `json:"error"`
		}{Error: hostErr}, "", "  ")
		if marshalErr == nil {
			return data, nil
		}
		return json.MarshalIndent(struct {
			Error *HostError `json:"error"`
		}{Error: &HostError{Code: hostErr.Code, Message: hostErr.Message}}, "", "  ")
	}
	return json.MarshalIndent(struct {
		Error map[string]any `json:"error"`
	}{Error: map[string]any{"code": string(ErrorCodeOf(err)), "message": err.Error()}}, "", "  ")
}
