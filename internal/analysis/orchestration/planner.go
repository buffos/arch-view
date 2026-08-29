package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
)

func NewAnalyzerJobPlanner(registry *analysis.Registry) *AnalyzerJobPlanner {
	return &AnalyzerJobPlanner{
		Registry:  registry,
		Discovery: NewProjectDiscoveryService(registry, DiscoveryPolicy{}),
	}
}

// PlanAnalyzerJobs is the package-level convenience entrypoint for callers
// that do not need to retain a planner instance.
func PlanAnalyzerJobs(ctx context.Context, registry *analysis.Registry, request PlanRequest) (JobPlan, error) {
	return NewAnalyzerJobPlanner(registry).PlanAnalyzerJobs(ctx, request)
}

// PlanAnalyzerJobs resolves roots, selections, options, and source scopes
// before any analyzer is executed. The returned plan is safe to serialize and
// is immutable by convention once handed to the scheduler.
func (p *AnalyzerJobPlanner) PlanAnalyzerJobs(ctx context.Context, request PlanRequest) (JobPlan, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if p == nil || p.Registry == nil {
		return JobPlan{}, analysis.NewHostError(analysis.ErrHostFailure, "analyzer job planner is not initialized", nil)
	}
	repositoryRoot, err := normalizeRepositoryRoot(request.RepositoryRoot)
	if err != nil {
		return JobPlan{}, err
	}
	invocationInput := request.InvocationRoot
	configuredInvocation := request.SourceScopePolicy.InvocationRoot
	if strings.TrimSpace(invocationInput) != "" && strings.TrimSpace(configuredInvocation) != "" {
		_, requestedRelative, requestErr := normalizeInvocationRoot(repositoryRoot, invocationInput)
		if requestErr != nil {
			return JobPlan{}, requestErr
		}
		_, configuredRelative, configuredErr := normalizeInvocationRoot(repositoryRoot, configuredInvocation)
		if configuredErr != nil {
			return JobPlan{}, configuredErr
		}
		if requestedRelative != configuredRelative {
			return JobPlan{}, scopeFilterError("source-scope policy invocation root does not match the plan invocation root", map[string]any{
				"expected": requestedRelative,
				"actual":   configuredRelative,
			})
		}
	}
	if strings.TrimSpace(invocationInput) == "" {
		invocationInput = configuredInvocation
	}
	_, invocationRelative, err := normalizeInvocationRoot(repositoryRoot, invocationInput)
	if err != nil {
		return JobPlan{}, err
	}
	policy := request.SourceScopePolicy
	policy.InvocationRoot = invocationRelative
	policy, err = NormalizeSourceScopePolicy(policy, invocationRelative)
	if err != nil {
		return JobPlan{}, err
	}

	discoveryService := p.Discovery
	if discoveryService == nil || discoveryService.Registry != p.Registry {
		discoveryService = NewProjectDiscoveryService(p.Registry, request.DiscoveryPolicy)
	} else if discoveryPolicyProvided(request.DiscoveryPolicy) {
		discoveryService.Policy = normalizeDiscoveryPolicy(request.DiscoveryPolicy)
	}
	discovery, err := discoveryService.DiscoverProjectRoots(ctx, repositoryRoot)
	if err != nil {
		return JobPlan{}, err
	}

	rootByRelative := make(map[string]ProjectRootCandidate, len(discovery.Roots))
	for _, root := range discovery.Roots {
		rootByRelative[root.RelativePath] = root
	}
	for _, assignment := range request.Assignments {
		relative, absolute, rootErr := normalizeSelectionRoot(repositoryRoot, assignment.ProjectRoot)
		if rootErr != nil {
			return JobPlan{}, rootErr
		}
		if _, exists := rootByRelative[relative]; !exists {
			rootByRelative[relative] = ProjectRootCandidate{AbsolutePath: absolute, RelativePath: relative}
		}
	}
	if request.CLISelection != nil && request.CLISelection.ProjectRoot != "" {
		relative, absolute, rootErr := normalizeSelectionRoot(repositoryRoot, request.CLISelection.ProjectRoot)
		if rootErr != nil {
			return JobPlan{}, rootErr
		}
		if _, exists := rootByRelative[relative]; !exists {
			rootByRelative[relative] = ProjectRootCandidate{AbsolutePath: absolute, RelativePath: relative}
		}
	}
	discovery.Roots = make([]ProjectRootCandidate, 0, len(rootByRelative))
	for _, root := range rootByRelative {
		discovery.Roots = append(discovery.Roots, root)
	}
	ResolveNestedProjectOwnership(&discovery)
	request, err = normalizePlanSelections(repositoryRoot, request)
	if err != nil {
		return JobPlan{}, err
	}

	plan := JobPlan{
		PlanVersion:            JobPlanSchemaVersion,
		RepositoryRoot:         repositoryRoot,
		InvocationRoot:         invocationRelative,
		DiscoveryPolicyVersion: discovery.Policy.Version,
		SourceScopePolicy:      policy,
		Jobs:                   []AnalyzerJob{},
		DiscoveryDiagnostics:   append([]analysis.Diagnostic(nil), discovery.Diagnostics...),
	}
	evaluations, evaluationDiagnostics, err := p.evaluateCandidates(ctx, discovery.Roots, request)
	if err != nil {
		return JobPlan{}, err
	}
	plan.DiscoveryDiagnostics = append(plan.DiscoveryDiagnostics, evaluationDiagnostics...)
	for _, evaluation := range evaluations {
		job, jobErr := p.buildJob(policy, evaluation, request.Runtime)
		if jobErr != nil {
			return JobPlan{}, jobErr
		}
		plan.Jobs = append(plan.Jobs, job)
	}
	sort.SliceStable(plan.Jobs, func(i, j int) bool {
		left, right := plan.Jobs[i], plan.Jobs[j]
		if left.RelativeProjectRoot != right.RelativeProjectRoot {
			return rootSortKey(left.RelativeProjectRoot) < rootSortKey(right.RelativeProjectRoot)
		}
		if left.Language != right.Language {
			return left.Language < right.Language
		}
		return left.LogicalAnalyzerID < right.LogicalAnalyzerID
	})

	maxJobs := discovery.Policy.MaxPlannedJobs
	if request.DiscoveryPolicy.MaxPlannedJobs > 0 && request.DiscoveryPolicy.MaxPlannedJobs < maxJobs {
		maxJobs = request.DiscoveryPolicy.MaxPlannedJobs
	}
	if maxJobs <= 0 || maxJobs > MaxPlannedJobs {
		maxJobs = MaxPlannedJobs
	}
	if len(plan.Jobs) > maxJobs {
		dropped := len(plan.Jobs) - maxJobs
		plan.Jobs = plan.Jobs[:maxJobs]
		plan.DiscoveryDiagnostics = append(plan.DiscoveryDiagnostics, analysis.Diagnostic{
			Code:        "discovery_limit_exceeded",
			Severity:    "error",
			Message:     fmt.Sprintf("project discovery produced more than the maximum of %d planned analyzer jobs; %d jobs were not scheduled", maxJobs, dropped),
			Recoverable: true,
			Metadata: map[string]any{
				"maximum_planned_jobs": maxJobs,
				"dropped_job_count":    dropped,
			},
		})
	}
	sortDiagnostics(plan.DiscoveryDiagnostics)
	return plan, nil
}

func discoveryPolicyProvided(policy DiscoveryPolicy) bool {
	return policy.Version != "" || len(policy.FixedExclusions) > 0 || policy.FollowSymlinks || policy.MaxPlannedJobs > 0 || policy.RepositoryScope != ""
}

func (p *AnalyzerJobPlanner) evaluateCandidates(ctx context.Context, roots []ProjectRootCandidate, request PlanRequest) ([]CandidateEvaluation, []analysis.Diagnostic, error) {
	assignments := append([]AnalyzerAssignment(nil), request.Assignments...)
	sort.SliceStable(assignments, func(i, j int) bool {
		leftRoot := normalizeRelativePath(assignments[i].ProjectRoot)
		rightRoot := normalizeRelativePath(assignments[j].ProjectRoot)
		if leftRoot != rightRoot {
			return rootSortKey(leftRoot) < rootSortKey(rightRoot)
		}
		if assignments[i].AnalyzerID != assignments[j].AnalyzerID {
			return assignments[i].AnalyzerID < assignments[j].AnalyzerID
		}
		left, _ := json.Marshal(assignments[i].Options)
		right, _ := json.Marshal(assignments[j].Options)
		return string(left) < string(right)
	})

	evaluations := []CandidateEvaluation{}
	diagnostics := []analysis.Diagnostic{}
	for _, root := range roots {
		selected := make(map[string]CandidateEvaluation)
		for _, assignment := range assignments {
			assignmentRoot := normalizeRelativePath(assignment.ProjectRoot)
			if assignmentRoot == "" {
				assignmentRoot = root.RelativePath
			}
			if assignmentRoot != root.RelativePath || !assignmentAllowedByCLI(assignment, root, request.CLISelection) {
				continue
			}
			evaluation, evaluationErr := p.evaluateExplicit(ctx, root, assignment.AnalyzerID, assignment.Language, SelectionAssignment, assignment.Options, request)
			if evaluationErr != nil {
				return nil, nil, evaluationErr
			}
			if request.CLISelection != nil && selectionAppliesToRoot(*request.CLISelection, root) && !evaluationMatchesCLI(evaluation, *request.CLISelection) {
				continue
			}
			addCandidateEvaluation(selected, evaluation)
		}

		if request.CLISelection != nil && selectionAppliesToRoot(*request.CLISelection, root) {
			selection := *request.CLISelection
			if selection.AnalyzerID != "" || selection.Language != "" {
				evaluation, evaluationErr := p.evaluateExplicit(ctx, root, selection.AnalyzerID, selection.Language, SelectionCLI, selection.Options, request)
				if evaluationErr != nil {
					return nil, nil, evaluationErr
				}
				addCandidateEvaluation(selected, evaluation)
			}
		}

		var consolidationErr error
		selected, consolidationErr = consolidateExplicitSelections(selected, root)
		if consolidationErr != nil {
			return nil, nil, consolidationErr
		}

		constrainAutomatic := request.CLISelection != nil &&
			selectionAppliesToRoot(*request.CLISelection, root) &&
			(request.CLISelection.AnalyzerID != "" || request.CLISelection.Language != "")
		if !constrainAutomatic {
			explicitLanguages := make(map[string]struct{})
			for _, evaluation := range selected {
				if evaluation.Source == SelectionAssignment || evaluation.Source == SelectionCLI {
					if language := evaluationLanguage(evaluation); language != "" {
						explicitLanguages[language] = struct{}{}
					}
				}
			}
			automaticByLanguage := make(map[string][]CandidateEvaluation)
			for _, analyzer := range p.Registry.List() {
				manifest := analyzer.Manifest()
				if _, alreadySelected := selected[manifest.ID]; alreadySelected {
					continue
				}
				candidate, detectErr := analyzer.Detect(ctx, analysis.DetectRequest{ProjectRoot: root.AbsolutePath})
				if detectErr != nil {
					if ctx.Err() != nil {
						return nil, nil, analysis.NewHostError(analysis.ErrCancelled, "analyzer discovery was cancelled", nil)
					}
					diagnostics = append(diagnostics, analysis.Diagnostic{
						Code:        "analyzer_detection_failed",
						Severity:    "warning",
						Message:     fmt.Sprintf("analyzer %s could not inspect candidate root: %v", manifest.ID, detectErr),
						Subject:     manifest.ID,
						Path:        root.RelativePath,
						Recoverable: true,
					})
					continue
				}
				if candidate.AnalyzerID != manifest.ID || candidate.Confidence <= 0 {
					continue
				}
				if candidate.Confidence > 1 {
					return nil, nil, analysis.NewHostError(analysis.ErrResultInvalid, "analyzer detection confidence is invalid", map[string]any{"analyzer_id": manifest.ID, "confidence": candidate.Confidence})
				}
				language := strings.ToLower(strings.TrimSpace(manifest.Language))
				if language == "" {
					continue
				}
				if _, explicitlySelected := explicitLanguages[language]; explicitlySelected {
					continue
				}
				effective, optionsErr := resolveAnalyzerOptions(manifest, request, manifest.ID, nil)
				if optionsErr != nil {
					return nil, nil, optionsErr
				}
				automaticByLanguage[language] = append(automaticByLanguage[language], CandidateEvaluation{
					Root: root, Analyzer: analyzer, Manifest: manifest, Language: language, Candidate: candidate,
					Source: SelectionAutomatic, Options: effective,
				})
			}
			languages := make([]string, 0, len(automaticByLanguage))
			for language := range automaticByLanguage {
				languages = append(languages, language)
			}
			sort.Strings(languages)
			for _, language := range languages {
				candidates := automaticByLanguage[language]
				sort.SliceStable(candidates, func(i, j int) bool {
					if candidates[i].Candidate.Confidence != candidates[j].Candidate.Confidence {
						return candidates[i].Candidate.Confidence > candidates[j].Candidate.Confidence
					}
					return candidates[i].Manifest.ID < candidates[j].Manifest.ID
				})
				if len(candidates) > 1 && candidates[0].Candidate.Confidence == candidates[1].Candidate.Confidence {
					ids := make([]string, 0, len(candidates))
					for _, candidate := range candidates {
						if candidate.Candidate.Confidence != candidates[0].Candidate.Confidence {
							break
						}
						ids = append(ids, candidate.Manifest.ID)
					}
					return nil, nil, analysis.NewHostError(analysis.ErrAmbiguousAnalyzer, "multiple analyzers have the same highest detection confidence", map[string]any{
						"project_root": root.RelativePath,
						"language":     language,
						"analyzer_ids": ids,
					})
				}
				addCandidateEvaluation(selected, candidates[0])
			}
		}

		keys := make([]string, 0, len(selected))
		for key := range selected {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			evaluations = append(evaluations, selected[key])
		}
	}
	return evaluations, diagnostics, nil
}

func (p *AnalyzerJobPlanner) evaluateExplicit(ctx context.Context, root ProjectRootCandidate, analyzerID, language string, source SelectionSource, assignmentOptions map[string]any, request PlanRequest) (CandidateEvaluation, error) {
	analyzerID = strings.TrimSpace(analyzerID)
	language = strings.ToLower(strings.TrimSpace(language))
	if analyzerID == "" && language != "" {
		analyzers := p.Registry.ByLanguage(language)
		switch len(analyzers) {
		case 0:
			analyzerID = "<language:" + language + ">"
		case 1:
			analyzerID = analyzers[0].Manifest().ID
		default:
			ids := make([]string, 0, len(analyzers))
			for _, analyzer := range analyzers {
				ids = append(ids, analyzer.Manifest().ID)
			}
			return CandidateEvaluation{}, analysis.NewHostError(analysis.ErrAmbiguousAnalyzer, "language maps to more than one registered analyzer", map[string]any{"language": language, "analyzer_ids": ids})
		}
	}
	evaluation := CandidateEvaluation{
		Root:     root,
		Source:   source,
		Language: language,
		Candidate: analysis.DetectionCandidate{
			AnalyzerID: analyzerID,
			Confidence: 1,
			Reason:     string(source) + " analyzer selection",
		},
	}
	analyzer, exists := p.Registry.Get(analyzerID)
	if !exists {
		evaluation.InitialDiagnostics = []analysis.Diagnostic{{
			Code:        "assignment_analyzer_unavailable",
			Severity:    "error",
			Message:     fmt.Sprintf("analyzer %q is not registered for this planned scope", analyzerID),
			Subject:     analyzerID,
			Path:        root.RelativePath,
			Recoverable: true,
		}}
		if source == SelectionCLI {
			evaluation.InitialDiagnostics[0].Code = "selection_analyzer_unavailable"
		}
		return evaluation, nil
	}
	evaluation.Analyzer = analyzer
	evaluation.Manifest = analyzer.Manifest()
	if language != "" && evaluation.Manifest.Language != language {
		evaluation.InitialDiagnostics = append(evaluation.InitialDiagnostics, analysis.Diagnostic{
			Code:        "selection_language_mismatch",
			Severity:    "error",
			Message:     fmt.Sprintf("analyzer %q is language %q, not %q", analyzerID, evaluation.Manifest.Language, language),
			Subject:     analyzerID,
			Path:        root.RelativePath,
			Recoverable: true,
		})
	}
	candidate, detectErr := analyzer.Detect(ctx, analysis.DetectRequest{ProjectRoot: root.AbsolutePath})
	if detectErr != nil {
		if ctx.Err() != nil {
			return CandidateEvaluation{}, analysis.NewHostError(analysis.ErrCancelled, "explicit analyzer discovery was cancelled", nil)
		}
		evaluation.InitialDiagnostics = append(evaluation.InitialDiagnostics, analysis.Diagnostic{
			Code:        "assignment_detection_failed",
			Severity:    "error",
			Message:     fmt.Sprintf("analyzer %s could not inspect the explicitly selected root: %v", analyzerID, detectErr),
			Subject:     analyzerID,
			Path:        root.RelativePath,
			Recoverable: true,
		})
	} else if candidate.AnalyzerID == analyzerID && candidate.Confidence > 0 {
		evaluation.Candidate = candidate
	} else {
		evaluation.InitialDiagnostics = append(evaluation.InitialDiagnostics, analysis.Diagnostic{
			Code:        "assignment_project_unsupported",
			Severity:    "error",
			Message:     fmt.Sprintf("analyzer %s did not detect the explicitly selected root", analyzerID),
			Subject:     analyzerID,
			Path:        root.RelativePath,
			Recoverable: true,
		})
	}
	effective, err := resolveAnalyzerOptions(evaluation.Manifest, request, analyzerID, assignmentOptions)
	if err != nil {
		return CandidateEvaluation{}, err
	}
	evaluation.Options = effective
	return evaluation, nil
}

// consolidateExplicitSelections enforces the one-logical-analyzer-per-
// root/language rule before automatic detection is considered. A CLI choice
// outranks an assignment, and an assignment outranks automatic detection.
// Conflicting choices at the same priority are ambiguous rather than being
// resolved by registry order.
func consolidateExplicitSelections(values map[string]CandidateEvaluation, root ProjectRootCandidate) (map[string]CandidateEvaluation, error) {
	byLanguage := make(map[string][]CandidateEvaluation)
	for _, evaluation := range values {
		language := evaluationLanguage(evaluation)
		if language == "" {
			continue
		}
		byLanguage[language] = append(byLanguage[language], evaluation)
	}
	for language, candidates := range byLanguage {
		highest := 0
		for _, candidate := range candidates {
			if priority := selectionPriority(candidate.Source); priority > highest {
				highest = priority
			}
		}
		winners := make([]CandidateEvaluation, 0, len(candidates))
		for _, candidate := range candidates {
			if selectionPriority(candidate.Source) == highest {
				winners = append(winners, candidate)
			}
		}
		if len(winners) <= 1 {
			continue
		}
		sort.SliceStable(winners, func(i, j int) bool {
			return candidateAnalyzerID(winners[i]) < candidateAnalyzerID(winners[j])
		})
		ids := make([]string, 0, len(winners))
		for _, winner := range winners {
			ids = append(ids, candidateAnalyzerID(winner))
		}
		return nil, analysis.NewHostError(analysis.ErrAmbiguousAnalyzer, "multiple explicitly selected analyzers target the same project language", map[string]any{
			"project_root": root.RelativePath,
			"language":     language,
			"analyzer_ids": ids,
		})
	}

	for language, candidates := range byLanguage {
		if len(candidates) == 0 {
			continue
		}
		highest := 0
		for _, candidate := range candidates {
			if priority := selectionPriority(candidate.Source); priority > highest {
				highest = priority
			}
		}
		for key, candidate := range values {
			if evaluationLanguage(candidate) == language && selectionPriority(candidate.Source) < highest {
				delete(values, key)
			}
		}
	}
	return values, nil
}

func evaluationLanguage(evaluation CandidateEvaluation) string {
	if evaluation.Manifest.Language != "" {
		return strings.ToLower(strings.TrimSpace(evaluation.Manifest.Language))
	}
	return strings.ToLower(strings.TrimSpace(evaluation.Language))
}

func candidateAnalyzerID(evaluation CandidateEvaluation) string {
	if evaluation.Manifest.ID != "" {
		return evaluation.Manifest.ID
	}
	return evaluation.Candidate.AnalyzerID
}

func addCandidateEvaluation(values map[string]CandidateEvaluation, evaluation CandidateEvaluation) {
	if evaluation.Manifest.ID == "" {
		key := evaluation.Candidate.AnalyzerID
		if key == "" {
			key = "<unavailable>"
		}
		if existing, ok := values[key]; !ok || selectionPriority(evaluation.Source) > selectionPriority(existing.Source) {
			values[key] = evaluation
		}
		return
	}
	key := evaluation.Manifest.ID
	if existing, ok := values[key]; !ok || selectionPriority(evaluation.Source) > selectionPriority(existing.Source) {
		values[key] = evaluation
	}
}

func selectionPriority(source SelectionSource) int {
	switch source {
	case SelectionCLI:
		return 3
	case SelectionAssignment:
		return 2
	default:
		return 1
	}
}

func resolveAnalyzerOptions(manifest analysis.Manifest, request PlanRequest, analyzerID string, assignmentOptions map[string]any) (analysis.EffectiveOptions, error) {
	project := cloneMap(request.ProjectOptions)
	for key, value := range request.ProjectOptionsByID[analyzerID] {
		project[key] = value
	}
	for key, value := range assignmentOptions {
		project[key] = value
	}
	cli := cloneMap(request.CLIOptions)
	for key, value := range request.CLIOptionsByID[analyzerID] {
		cli[key] = value
	}
	if request.CLISelection != nil {
		for key, value := range request.CLISelection.Options {
			cli[key] = value
		}
	}
	return analysis.ResolveOptions(manifest, project, cli)
}

func (p *AnalyzerJobPlanner) buildJob(policy SourceScopePolicy, evaluation CandidateEvaluation, runtime analysis.RuntimeSelection) (AnalyzerJob, error) {
	analyzerID := evaluation.Manifest.ID
	if analyzerID == "" {
		analyzerID = evaluation.Candidate.AnalyzerID
	}
	language := evaluation.Manifest.Language
	if language == "" {
		language = evaluation.Language
	}
	selection := analysis.AnalyzerSelection{
		AnalyzerID:      analyzerID,
		Mode:            string(evaluation.Source),
		Confidence:      evaluation.Candidate.Confidence,
		MatchedMarkers:  append([]string(nil), evaluation.Candidate.MatchedMarkers...),
		Reason:          evaluation.Candidate.Reason,
		BoundaryHint:    evaluation.Candidate.BoundaryHint,
		RuntimeMode:     runtime.Mode,
		RuntimeSource:   runtime.Source,
		RuntimePlatform: runtime.Platform,
	}
	if selection.Reason == "" {
		selection.Reason = string(evaluation.Source) + " analyzer selection"
	}
	sourceScope, sourceFingerprint, err := buildEffectiveSourceScope(policy, evaluation.Root, analyzerID, evaluation.Options)
	if err != nil {
		return AnalyzerJob{}, err
	}
	job := AnalyzerJob{
		ScopeID:                     scopeID(evaluation.Root.RelativePath, analyzerID),
		ProjectRoot:                 evaluation.Root.AbsolutePath,
		RelativeProjectRoot:         evaluation.Root.RelativePath,
		LogicalAnalyzerID:           analyzerID,
		AnalyzerVersion:             evaluation.Manifest.Version,
		Language:                    language,
		RuntimeMode:                 runtime.Mode,
		RuntimeSource:               runtime.Source,
		RuntimePlatform:             runtime.Platform,
		SelectionSource:             evaluation.Source,
		Selection:                   selection,
		NestedRootExclusions:        append([]string(nil), evaluation.Root.NestedRootExclusions...),
		EffectiveSourceScope:        sourceScope,
		SourceScopeFingerprint:      sourceFingerprint,
		MatchedSourceSetFingerprint: sourceScope.MatchedSourceSetFingerprint,
		EffectiveOptionsFingerprint: evaluation.Options.Fingerprint,
		Options:                     cloneEffectiveOptions(evaluation.Options),
		Status:                      JobPlanned,
		Diagnostics:                 append([]analysis.Diagnostic(nil), evaluation.InitialDiagnostics...),
		Manifest:                    evaluation.Manifest,
	}
	job.JobID = jobID(job)
	return job, nil
}

func normalizeInvocationRoot(repositoryRoot, value string) (string, string, error) {
	if strings.TrimSpace(value) == "" {
		return repositoryRoot, ".", nil
	}
	relative, absolute, err := normalizeSelectionRoot(repositoryRoot, value)
	return absolute, relative, err
}

func normalizePlanSelections(repositoryRoot string, request PlanRequest) (PlanRequest, error) {
	assignments := make([]AnalyzerAssignment, len(request.Assignments))
	for index, assignment := range request.Assignments {
		assignmentCopy := assignment
		if strings.TrimSpace(assignmentCopy.ProjectRoot) == "" {
			assignmentCopy.ProjectRoot = "."
		} else {
			relative, _, err := normalizeSelectionRoot(repositoryRoot, assignmentCopy.ProjectRoot)
			if err != nil {
				return PlanRequest{}, err
			}
			assignmentCopy.ProjectRoot = relative
		}
		assignmentCopy.AnalyzerID = strings.TrimSpace(assignmentCopy.AnalyzerID)
		assignmentCopy.Language = strings.ToLower(strings.TrimSpace(assignmentCopy.Language))
		assignments[index] = assignmentCopy
	}
	request.Assignments = assignments
	if request.CLISelection != nil {
		selection := *request.CLISelection
		if selection.ProjectRoot != "" {
			relative, _, err := normalizeSelectionRoot(repositoryRoot, selection.ProjectRoot)
			if err != nil {
				return PlanRequest{}, err
			}
			selection.ProjectRoot = relative
		}
		selection.AnalyzerID = strings.TrimSpace(selection.AnalyzerID)
		selection.Language = strings.ToLower(strings.TrimSpace(selection.Language))
		request.CLISelection = &selection
	}
	return request, nil
}

func normalizeSelectionRoot(repositoryRoot, value string) (string, string, error) {
	if strings.TrimSpace(value) == "" {
		return ".", repositoryRoot, nil
	}
	absolute := value
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(repositoryRoot, filepath.FromSlash(value))
	}
	absolute, err := filepath.Abs(absolute)
	if err != nil {
		return "", "", analysis.WrapHostError(analysis.ErrInvalidRequest, "selection root could not be normalized", err, map[string]any{"project_root": value})
	}
	absolute = filepath.Clean(absolute)
	if !pathWithin(repositoryRoot, absolute) {
		return "", "", analysis.NewHostError(analysis.ErrInvalidRequest, "selection root must remain inside the opened repository", map[string]any{"project_root": value})
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", "", analysis.WrapHostError(analysis.ErrUnreadableProject, "selection root could not be read", err, map[string]any{"project_root": value})
	}
	if !info.IsDir() {
		return "", "", analysis.NewHostError(analysis.ErrInvalidRequest, "selection root must be a directory", map[string]any{"project_root": value})
	}
	if !resolvedPathWithin(repositoryRoot, absolute) {
		return "", "", analysis.NewHostError(analysis.ErrInvalidRequest, "selection root must remain inside the opened repository after symlink resolution", map[string]any{"project_root": value})
	}
	relative, err := filepath.Rel(repositoryRoot, absolute)
	if err != nil {
		return "", "", analysis.WrapHostError(analysis.ErrInvalidRequest, "selection root relationship could not be calculated", err, nil)
	}
	return normalizeRelativePath(relative), absolute, nil
}

func relativePathForRoot(repositoryRoot, candidate string) (string, error) {
	absolute, err := filepath.Abs(candidate)
	if err != nil || !pathWithin(repositoryRoot, absolute) {
		return "", analysis.NewHostError(analysis.ErrInvalidRequest, "invocation root must remain inside the opened repository", map[string]any{"invocation_root": candidate})
	}
	relative, err := filepath.Rel(repositoryRoot, filepath.Clean(absolute))
	if err != nil {
		return "", analysis.WrapHostError(analysis.ErrInvalidRequest, "invocation root relationship could not be calculated", err, nil)
	}
	return normalizeRelativePath(relative), nil
}

func selectionAppliesToRoot(selection ExplicitSelection, root ProjectRootCandidate) bool {
	if selection.ProjectRoot == "" {
		return true
	}
	return normalizeRelativePath(selection.ProjectRoot) == root.RelativePath || filepath.Clean(selection.ProjectRoot) == filepath.Clean(root.AbsolutePath)
}

func assignmentAllowedByCLI(assignment AnalyzerAssignment, root ProjectRootCandidate, selection *ExplicitSelection) bool {
	if selection == nil {
		return true
	}
	if !selectionAppliesToRoot(*selection, root) {
		return false
	}
	if selection.AnalyzerID != "" && assignment.AnalyzerID != selection.AnalyzerID {
		return false
	}
	if selection.Language != "" && strings.ToLower(assignment.Language) != strings.ToLower(selection.Language) && assignment.Language != "" {
		return false
	}
	return true
}

func evaluationMatchesCLI(evaluation CandidateEvaluation, selection ExplicitSelection) bool {
	if selection.AnalyzerID != "" && candidateAnalyzerID(evaluation) != selection.AnalyzerID {
		return false
	}
	if selection.Language != "" && evaluationLanguage(evaluation) != strings.ToLower(strings.TrimSpace(selection.Language)) {
		return false
	}
	return true
}

func cloneMap(values map[string]any) map[string]any {
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = cloneOptionValue(value)
	}
	return result
}

func cloneOptionValue(value any) any {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]any, len(typed))
		for index := range typed {
			result[index] = cloneOptionValue(typed[index])
		}
		return result
	case map[string]any:
		return cloneMap(typed)
	case map[string]string:
		return cloneStringMap(typed)
	default:
		return value
	}
}

func cloneEffectiveOptions(value analysis.EffectiveOptions) analysis.EffectiveOptions {
	return analysis.EffectiveOptions{Values: cloneMap(value.Values), Sources: cloneStringMap(value.Sources), Fingerprint: value.Fingerprint}
}

func cloneStringMap(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func sortDiagnostics(values []analysis.Diagnostic) {
	sort.Slice(values, func(i, j int) bool {
		left, _ := json.Marshal(values[i])
		right, _ := json.Marshal(values[j])
		return string(left) < string(right)
	})
}
