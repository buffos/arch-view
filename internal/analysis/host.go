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
}

func NewHost(registry *Registry) *Host {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Host{registry: registry}
}

func (h *Host) ListManifests() []Manifest {
	return h.registry.ListManifests()
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
	if ctx == nil {
		ctx = context.Background()
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
	analyzeRequest := AnalyzeRequest{
		ProjectRoot: root,
		Selection:   selection,
		Options:     effectiveOptions,
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
	if result.Analyzer.ID == "" {
		result.Analyzer = analyzerInfo(manifest)
	}
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
		return h.confirmExplicitSelection(ctx, root, analyzer, "explicit-id")
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
		return h.confirmExplicitSelection(ctx, root, analyzers[0], "explicit-language")
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
	return chosen.analyzer, AnalyzerSelection{
		AnalyzerID:     chosen.analyzer.Manifest().ID,
		Mode:           "auto",
		Confidence:     chosen.candidate.Confidence,
		MatchedMarkers: chosen.candidate.MatchedMarkers,
		Reason:         chosen.candidate.Reason,
		BoundaryHint:   chosen.candidate.BoundaryHint,
	}, nil
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
