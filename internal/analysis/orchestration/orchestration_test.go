package orchestration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
)

type fixtureAnalyzer struct {
	manifest analysis.Manifest
	detect   func(context.Context, analysis.DetectRequest) (analysis.DetectionCandidate, error)
	analyze  func(context.Context, analysis.AnalyzeRequest) (analysis.AnalysisResult, error)
}

func (a *fixtureAnalyzer) Manifest() analysis.Manifest { return a.manifest }

func (a *fixtureAnalyzer) Detect(ctx context.Context, request analysis.DetectRequest) (analysis.DetectionCandidate, error) {
	if a.detect != nil {
		return a.detect(ctx, request)
	}
	return detectFixture(a.manifest, request), nil
}

func (a *fixtureAnalyzer) Analyze(ctx context.Context, request analysis.AnalyzeRequest) (analysis.AnalysisResult, error) {
	if a.analyze != nil {
		return a.analyze(ctx, request)
	}
	return fixtureResult(a.manifest), nil
}

func fixtureManifest(id, language, marker string, weight float64) analysis.Manifest {
	return analysis.Manifest{
		ID: id, Version: "1.0.0", Language: language, APIVersion: analysis.AnalyzerAPIVersion,
		DetectionMarkers: []analysis.DetectionMarker{{Kind: "file", Value: marker, Weight: weight}},
		Capabilities:     []string{"detect"}, Options: []analysis.OptionDescriptor{{Name: "exclude", Type: "string[]", Default: []string{}}},
	}
}

func detectFixture(manifest analysis.Manifest, request analysis.DetectRequest) analysis.DetectionCandidate {
	matched := []string{}
	for _, marker := range manifest.DetectionMarkers {
		if info, err := os.Stat(filepath.Join(request.ProjectRoot, marker.Value)); err == nil && !info.IsDir() {
			matched = append(matched, marker.Value)
		}
	}
	if len(matched) == 0 {
		return analysis.DetectionCandidate{AnalyzerID: manifest.ID, Reason: "fixture marker detection"}
	}
	return analysis.DetectionCandidate{AnalyzerID: manifest.ID, Confidence: manifest.DetectionMarkers[0].Weight, MatchedMarkers: matched, Reason: "fixture marker detection"}
}

func fixtureResult(manifest analysis.Manifest) analysis.AnalysisResult {
	return analysis.AnalysisResult{
		RunID: "fixture-run", Status: analysis.StatusComplete,
		Analyzer:         analysis.AnalyzerInfo{ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion},
		Project:          analysis.ProjectInfo{RootLabel: "fixture", Boundary: "fixture.marker"},
		Modules:          []analysis.ModuleObservation{{ID: "module", Language: manifest.Language, Kind: "module", Name: "module", DisplayName: "module", Hierarchy: []string{"module"}, SourceReferenceIDs: []string{"source"}}},
		SourceReferences: []analysis.SourceReference{{ID: "source", Path: "main.src", Kind: "definition"}},
	}
}

func TestPlanDiscoversMixedRootsNestedOwnershipAndSourceScope(t *testing.T) {
	repository := t.TempDir()
	writeFixtureFile(t, repository, "go.mod", "module fixture\n")
	writeFixtureFile(t, repository, "main.go", "package main\n")
	writeFixtureFile(t, repository, "excluded/kept.go", "package excluded\n")
	writeFixtureFile(t, repository, "vendor/go.mod", "module vendor\n")
	writeFixtureFile(t, repository, "nested/pyproject.toml", "[project]\nname='nested'\n")
	writeFixtureFile(t, repository, "nested/main.py", "VALUE = 1\n")
	writeFixtureFile(t, repository, "nested/ignored.py", "VALUE = 2\n")
	writeFixtureFile(t, repository, "package.json", "{}\n")

	goAnalyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.go", "go", "go.mod", 1)}
	pythonAnalyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.python", "python", "pyproject.toml", 1)}
	typeScriptAnalyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.typescript", "typescript", "package.json", 0.6)}
	registry := analysis.NewRegistry()
	for _, analyzer := range []*fixtureAnalyzer{goAnalyzer, pythonAnalyzer, typeScriptAnalyzer} {
		if err := registry.Register(analyzer); err != nil {
			t.Fatalf("register %s: %v", analyzer.manifest.ID, err)
		}
	}

	planner := NewAnalyzerJobPlanner(registry)
	policy := SourceScopePolicy{
		Exclude: []string{"excluded/**", "nested/ignored.py"},
		Include: []AnalyzerIncludeRule{{AnalyzerID: goAnalyzer.manifest.ID, Globs: []string{"**/*.go"}}},
	}
	plan, err := planner.PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: repository, SourceScopePolicy: policy})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(plan.Jobs) != 3 {
		t.Fatalf("jobs = %d, want Go + TypeScript at root and Python at nested root: %#v", len(plan.Jobs), plan.Jobs)
	}
	if plan.Jobs[0].RelativeProjectRoot != "." || plan.Jobs[1].RelativeProjectRoot != "." || plan.Jobs[2].RelativeProjectRoot != "nested" {
		t.Fatalf("job order = %#v", plan.Jobs)
	}
	if plan.Jobs[2].Language != "python" || plan.Jobs[2].SelectionSource != SelectionAutomatic {
		t.Fatalf("nested job = %#v", plan.Jobs[2])
	}
	for _, job := range plan.Jobs {
		if job.RelativeProjectRoot == "." && job.Language == "go" {
			if len(job.NestedRootExclusions) != 1 || job.NestedRootExclusions[0] != "nested" {
				t.Fatalf("Go nested exclusions = %#v", job.NestedRootExclusions)
			}
			if contains(job.EffectiveSourceScope.MatchedPaths, "nested/main.py") || contains(job.EffectiveSourceScope.MatchedPaths, "excluded/kept.go") {
				t.Fatalf("Go source scope owns excluded/nested content: %#v", job.EffectiveSourceScope)
			}
			if !contains(job.EffectiveSourceScope.MatchedLocalPaths, "main.go") {
				t.Fatalf("Go source scope omitted main.go: %#v", job.EffectiveSourceScope)
			}
		}
		if job.RelativeProjectRoot == "nested" {
			if contains(job.EffectiveSourceScope.MatchedPaths, "nested/ignored.py") || !contains(job.EffectiveSourceScope.MatchedPaths, "nested/main.py") {
				t.Fatalf("Python source scope = %#v", job.EffectiveSourceScope)
			}
		}
	}
	if plan.Jobs[0].ScopeID == plan.Jobs[1].ScopeID || plan.Jobs[0].JobID == plan.Jobs[1].JobID {
		t.Fatalf("scope/job identities collided: %#v", plan.Jobs)
	}
	dataA, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	planAgain, err := planner.PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: repository, SourceScopePolicy: policy})
	if err != nil {
		t.Fatalf("repeat plan: %v", err)
	}
	dataB, _ := json.Marshal(planAgain)
	if string(dataA) != string(dataB) {
		t.Fatal("repeated plan serialization is not deterministic")
	}
}

func TestPlanWeakMarkersDoNotCreateRootsAndAssignmentsOverrideAutomaticSelection(t *testing.T) {
	weakRoot := t.TempDir()
	writeFixtureFile(t, weakRoot, "package.json", "{}\n")
	weakAnalyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.weak", "typescript", "package.json", 0.6)}
	registry := analysis.NewRegistry()
	if err := registry.Register(weakAnalyzer); err != nil {
		t.Fatal(err)
	}
	plan, err := NewAnalyzerJobPlanner(registry).PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: weakRoot})
	if err != nil {
		t.Fatalf("weak plan: %v", err)
	}
	if len(plan.Jobs) != 0 {
		t.Fatalf("weak marker created automatic jobs: %#v", plan.Jobs)
	}

	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module fixture\n")
	assigned := &fixtureAnalyzer{manifest: fixtureManifest("org.example.assigned", "go", "go.mod", 1)}
	if err := registry.Register(assigned); err != nil {
		t.Fatal(err)
	}
	plan, err = NewAnalyzerJobPlanner(registry).PlanAnalyzerJobs(context.Background(), PlanRequest{
		RepositoryRoot: root,
		Assignments:    []AnalyzerAssignment{{ProjectRoot: ".", AnalyzerID: assigned.manifest.ID}},
	})
	if err != nil {
		t.Fatalf("assignment plan: %v", err)
	}
	if len(plan.Jobs) != 1 || plan.Jobs[0].SelectionSource != SelectionAssignment {
		t.Fatalf("assignment selection = %#v", plan.Jobs)
	}
}

func TestPlanChoosesOneAutomaticAnalyzerPerLanguageAndRejectsTies(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module fixture\n")

	makeAnalyzer := func(id string, confidence float64) *fixtureAnalyzer {
		analyzer := &fixtureAnalyzer{manifest: fixtureManifest(id, "go", "go.mod", 1)}
		analyzer.detect = func(context.Context, analysis.DetectRequest) (analysis.DetectionCandidate, error) {
			return analysis.DetectionCandidate{AnalyzerID: id, Confidence: confidence, Reason: "confidence fixture"}, nil
		}
		return analyzer
	}
	high := makeAnalyzer("org.example.go-high", 0.9)
	low := makeAnalyzer("org.example.go-low", 0.7)
	registry := analysis.NewRegistry()
	for _, analyzer := range []*fixtureAnalyzer{low, high} {
		if err := registry.Register(analyzer); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := NewAnalyzerJobPlanner(registry).PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: root})
	if err != nil {
		t.Fatalf("confidence-ranked plan: %v", err)
	}
	if len(plan.Jobs) != 1 || plan.Jobs[0].LogicalAnalyzerID != high.manifest.ID {
		t.Fatalf("automatic language selection = %#v", plan.Jobs)
	}

	tieRoot := t.TempDir()
	writeFixtureFile(t, tieRoot, "go.mod", "module fixture\n")
	tieA := makeAnalyzer("org.example.go-a", 0.8)
	tieB := makeAnalyzer("org.example.go-b", 0.8)
	tieRegistry := analysis.NewRegistry()
	for _, analyzer := range []*fixtureAnalyzer{tieB, tieA} {
		if err := tieRegistry.Register(analyzer); err != nil {
			t.Fatal(err)
		}
	}
	_, err = NewAnalyzerJobPlanner(tieRegistry).PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: tieRoot})
	if analysis.ErrorCodeOf(err) != analysis.ErrAmbiguousAnalyzer {
		t.Fatalf("automatic tie error = %v, want %s", err, analysis.ErrAmbiguousAnalyzer)
	}
}

func TestPlanRejectsProjectRootSymlinkOutsideRepository(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "linked-project")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}
	registry := analysis.NewRegistry()
	analyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.symlink", "fixture", "fixture.marker", 1)}
	if err := registry.Register(analyzer); err != nil {
		t.Fatal(err)
	}
	_, err := NewAnalyzerJobPlanner(registry).PlanAnalyzerJobs(context.Background(), PlanRequest{
		RepositoryRoot: root,
		Assignments:    []AnalyzerAssignment{{ProjectRoot: "linked-project", AnalyzerID: analyzer.manifest.ID}},
	})
	if analysis.ErrorCodeOf(err) != analysis.ErrInvalidRequest {
		t.Fatalf("outside symlink error = %v, want %s", err, analysis.ErrInvalidRequest)
	}
}

func TestPlanSourceContentChangesInvalidateMatchedSourceIdentity(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module fixture\n")
	writeFixtureFile(t, root, "main.go", "package main\n")
	analyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.content", "go", "go.mod", 1)}
	registry := analysis.NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatal(err)
	}
	planner := NewAnalyzerJobPlanner(registry)
	first, err := planner.PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: root})
	if err != nil {
		t.Fatalf("first plan: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := planner.PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: root})
	if err != nil {
		t.Fatalf("second plan: %v", err)
	}
	if len(first.Jobs) != 1 || len(second.Jobs) != 1 {
		t.Fatalf("content plans = %#v %#v", first.Jobs, second.Jobs)
	}
	if first.Jobs[0].MatchedSourceSetFingerprint == second.Jobs[0].MatchedSourceSetFingerprint || first.Jobs[0].CacheKey == second.Jobs[0].CacheKey || first.Jobs[0].JobID == second.Jobs[0].JobID {
		t.Fatalf("source content did not invalidate job identity: first=%#v second=%#v", first.Jobs[0], second.Jobs[0])
	}
}

func TestPlanIgnoresHostConfigurationContentForSourceIdentity(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module fixture\n")
	writeFixtureFile(t, root, "main.go", "package main\n")
	writeFixtureFile(t, root, ".archview.json", `{"schema_version":"arch-view.config/v2","layout":{"algorithm":"layered","options":{}},"analysis":{}}`)
	analyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.config-content", "go", "go.mod", 1)}
	registry := analysis.NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatal(err)
	}
	planner := NewAnalyzerJobPlanner(registry)
	first, err := planner.PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: root})
	if err != nil {
		t.Fatalf("first plan: %v", err)
	}
	writeFixtureFile(t, root, ".archview.json", `{"schema_version":"arch-view.config/v2","layout":{"algorithm":"layered","options":{"org.eclipse.elk.direction":"RIGHT"}},"analysis":{}}`)
	second, err := planner.PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: root})
	if err != nil {
		t.Fatalf("second plan: %v", err)
	}
	if len(first.Jobs) != 1 || len(second.Jobs) != 1 {
		t.Fatalf("configuration plans = %#v %#v", first.Jobs, second.Jobs)
	}
	if contains(first.Jobs[0].EffectiveSourceScope.MatchedPaths, ".archview.json") {
		t.Fatalf("host configuration leaked into source scope: %#v", first.Jobs[0].EffectiveSourceScope.MatchedPaths)
	}
	if first.Jobs[0].MatchedSourceSetFingerprint != second.Jobs[0].MatchedSourceSetFingerprint || first.Jobs[0].CacheKey != second.Jobs[0].CacheKey {
		t.Fatalf("layout-only configuration edit invalidated analysis identity: first=%#v second=%#v", first.Jobs[0], second.Jobs[0])
	}
}

func TestCacheIdentityIncludesNormalizedDiscoveryPolicy(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module fixture\n")
	writeFixtureFile(t, root, "main.go", "package main\n")
	analyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.discovery-policy", "go", "go.mod", 1)}
	registry := analysis.NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatal(err)
	}
	planner := NewAnalyzerJobPlanner(registry)
	first, err := planner.PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: root, DiscoveryPolicy: DiscoveryPolicy{RepositoryScope: "first"}})
	if err != nil {
		t.Fatalf("first plan: %v", err)
	}
	second, err := planner.PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: root, DiscoveryPolicy: DiscoveryPolicy{RepositoryScope: "second"}})
	if err != nil {
		t.Fatalf("second plan: %v", err)
	}
	if len(first.Jobs) != 1 || len(second.Jobs) != 1 || first.Jobs[0].CacheKey == second.Jobs[0].CacheKey {
		t.Fatalf("discovery policy did not change cache identity: first=%#v second=%#v", first.Jobs, second.Jobs)
	}
}

func TestCacheIdentityIncludesAnalyzerPackageIdentity(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module fixture\n")
	writeFixtureFile(t, root, "main.go", "package main\n")
	manifest := fixtureManifest("org.example.package-identity", "go", "go.mod", 1)
	manifest.RuntimeIdentity = "sha256:package-a"
	firstRegistry := analysis.NewRegistry()
	if err := firstRegistry.Register(&fixtureAnalyzer{manifest: manifest}); err != nil {
		t.Fatal(err)
	}
	first, err := PlanAnalyzerJobs(context.Background(), firstRegistry, PlanRequest{RepositoryRoot: root})
	if err != nil {
		t.Fatalf("first plan: %v", err)
	}
	manifest.RuntimeIdentity = "sha256:package-b"
	secondRegistry := analysis.NewRegistry()
	if err := secondRegistry.Register(&fixtureAnalyzer{manifest: manifest}); err != nil {
		t.Fatal(err)
	}
	second, err := PlanAnalyzerJobs(context.Background(), secondRegistry, PlanRequest{RepositoryRoot: root})
	if err != nil {
		t.Fatalf("second plan: %v", err)
	}
	if len(first.Jobs) != 1 || len(second.Jobs) != 1 || first.Jobs[0].CacheKey == second.Jobs[0].CacheKey {
		t.Fatalf("package identity did not invalidate cache key: first=%#v second=%#v", first.Jobs, second.Jobs)
	}
}

func TestPlanNestedInvocationRootContentChangesInvalidateMatchedSourceIdentity(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "apps/service/go.mod", "module fixture\n")
	writeFixtureFile(t, root, "apps/service/main.go", "package main\n")
	analyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.nested-content", "go", "go.mod", 1)}
	registry := analysis.NewRegistry()
	if err := registry.Register(analyzer); err != nil {
		t.Fatal(err)
	}
	planner := NewAnalyzerJobPlanner(registry)
	request := PlanRequest{RepositoryRoot: root, InvocationRoot: "apps"}
	first, err := planner.PlanAnalyzerJobs(context.Background(), request)
	if err != nil {
		t.Fatalf("first nested plan: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "apps", "service", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := planner.PlanAnalyzerJobs(context.Background(), request)
	if err != nil {
		t.Fatalf("second nested plan: %v", err)
	}
	if len(first.Jobs) != 1 || len(second.Jobs) != 1 {
		t.Fatalf("nested content plans = %#v %#v", first.Jobs, second.Jobs)
	}
	if first.Jobs[0].MatchedSourceSetFingerprint == second.Jobs[0].MatchedSourceSetFingerprint || first.Jobs[0].JobID == second.Jobs[0].JobID {
		t.Fatalf("nested invocation content did not invalidate job identity: first=%#v second=%#v", first.Jobs[0], second.Jobs[0])
	}
}

func TestPlanRejectsUnsafeGlobsAndEnforcesLimit(t *testing.T) {
	registry := analysis.NewRegistry()
	if err := registry.Register(&fixtureAnalyzer{manifest: fixtureManifest("org.example.go", "go", "go.mod", 1)}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for index := 0; index < MaxPlannedJobs+1; index++ {
		// Use a stable numeric directory name while keeping the fixture creation
		// independent of platform path separators.
		directory := filepath.Join(root, "project-"+itoa(index))
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFixtureFile(t, directory, "go.mod", "module fixture\n")
	}
	plan, err := NewAnalyzerJobPlanner(registry).PlanAnalyzerJobs(context.Background(), PlanRequest{RepositoryRoot: root})
	if err != nil {
		t.Fatalf("limited plan: %v", err)
	}
	if len(plan.Jobs) != MaxPlannedJobs || !hasDiagnostic(plan.DiscoveryDiagnostics, "discovery_limit_exceeded") {
		t.Fatalf("limit plan jobs=%d diagnostics=%#v", len(plan.Jobs), plan.DiscoveryDiagnostics)
	}
	for _, glob := range []string{"", "../outside", "/absolute", `C:\\outside`, "!src/**", "#comment", "[broken"} {
		_, err := NormalizeSourceScopePolicy(SourceScopePolicy{Exclude: []string{glob}}, ".")
		if analysis.ErrorCodeOf(err) != analysis.ErrAnalysisScopeFilterInvalid {
			t.Fatalf("glob %q error = %v, want scope-filter-invalid", glob, err)
		}
	}
}

func TestSchedulerBoundsConcurrencyAndRetainsIndependentResults(t *testing.T) {
	jobs := make([]AnalyzerJob, 8)
	manifest := fixtureManifest("org.example.fixture", "fixture", "fixture.marker", 1)
	for index := range jobs {
		jobs[index] = plannedFixtureJob(t, manifest, "job-"+itoa(index), "scope-"+itoa(index))
	}
	var active int32
	var maximum int32
	executor := func(ctx context.Context, job AnalyzerJob) (analysis.AnalysisResult, error) {
		current := atomic.AddInt32(&active, 1)
		for {
			previous := atomic.LoadInt32(&maximum)
			if current <= previous || atomic.CompareAndSwapInt32(&maximum, previous, current) {
				break
			}
		}
		select {
		case <-time.After(25 * time.Millisecond):
		case <-ctx.Done():
		}
		atomic.AddInt32(&active, -1)
		if ctx.Err() != nil {
			return analysis.AnalysisResult{}, ctx.Err()
		}
		return fixtureResult(manifest), nil
	}
	scheduler := NewAnalyzerJobScheduler(executor, SchedulerOptions{WorkerCount: 99})
	snapshot := scheduler.ExecuteAnalyzerPlan(context.Background(), JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: t.TempDir(), Jobs: jobs})
	if maximum > HardMaxWorkerCount {
		t.Fatalf("maximum concurrency = %d, exceeds hard cap", maximum)
	}
	if maximum > HardMaxWorkerCount || maximum == 0 {
		t.Fatalf("maximum concurrency = %d", maximum)
	}
	for _, job := range snapshot.Jobs {
		if job.Status != JobComplete || job.Result == nil {
			t.Fatalf("job did not complete independently: %#v", job)
		}
	}
	if snapshot.Events[0].Type != "run.started" || snapshot.Events[len(snapshot.Events)-1].Type != "run.completed" {
		t.Fatalf("lifecycle boundary events = %#v", snapshot.Events)
	}
	for index, event := range snapshot.Events {
		if event.Sequence != index+1 {
			t.Fatalf("event sequence at %d = %d", index, event.Sequence)
		}
	}
	secondSnapshot := scheduler.ExecuteAnalyzerPlan(context.Background(), JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: snapshot.Plan.RepositoryRoot, Jobs: jobs})
	if snapshot.RunID == secondSnapshot.RunID {
		t.Fatalf("retried runs reused id %q", snapshot.RunID)
	}
}

func TestSchedulerDoesNotExecuteJobsWithPlanningErrors(t *testing.T) {
	manifest := fixtureManifest("org.example.invalid-plan", "fixture", "fixture.marker", 1)
	job := plannedFixtureJob(t, manifest, "job-invalid", "scope-invalid")
	job.Diagnostics = []analysis.Diagnostic{{Code: "assignment_project_unsupported", Severity: "error", Message: "assignment does not match this root", Recoverable: true}}
	var calls atomic.Int32
	snapshot := NewAnalyzerJobScheduler(func(context.Context, AnalyzerJob) (analysis.AnalysisResult, error) {
		calls.Add(1)
		return fixtureResult(manifest), nil
	}, SchedulerOptions{WorkerCount: 1}).ExecuteAnalyzerPlan(context.Background(), JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: t.TempDir(), Jobs: []AnalyzerJob{job}})
	if calls.Load() != 0 || snapshot.Jobs[0].Status != JobFailed || snapshot.Jobs[0].Result != nil {
		t.Fatalf("invalid planned job executed: calls=%d job=%#v", calls.Load(), snapshot.Jobs[0])
	}
}

func TestSchedulerKeepsPlanInputsImmutableFromExecutorMutation(t *testing.T) {
	manifest := fixtureManifest("org.example.immutable", "fixture", "fixture.marker", 1)
	job := plannedFixtureJob(t, manifest, "job-immutable", "scope-immutable")
	job.Options.Values["exclude"] = []string{"keep/**"}
	plan := JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: t.TempDir(), Jobs: []AnalyzerJob{job}}
	snapshot := NewAnalyzerJobScheduler(func(_ context.Context, received AnalyzerJob) (analysis.AnalysisResult, error) {
		received.Options.Values["exclude"].([]string)[0] = "changed/**"
		received.Manifest.ID = "org.example.changed"
		return fixtureResult(manifest), nil
	}, SchedulerOptions{WorkerCount: 1}).ExecuteAnalyzerPlan(context.Background(), plan)
	for label, value := range map[string]AnalyzerJob{
		"input":         plan.Jobs[0],
		"snapshot plan": snapshot.Plan.Jobs[0],
		"snapshot job":  snapshot.Jobs[0],
	} {
		if got := value.Options.Values["exclude"].([]string)[0]; got != "keep/**" {
			t.Fatalf("%s options mutated to %q", label, got)
		}
		if value.Manifest.ID != manifest.ID {
			t.Fatalf("%s manifest mutated to %q", label, value.Manifest.ID)
		}
	}
	if snapshot.Jobs[0].Status != JobComplete {
		t.Fatalf("immutable fixture job status = %q", snapshot.Jobs[0].Status)
	}
}

func TestSchedulerRetainsIndependentSuccessWhenOneJobFails(t *testing.T) {
	manifest := fixtureManifest("org.example.fixture", "fixture", "fixture.marker", 1)
	success := plannedFixtureJob(t, manifest, "job-success", "scope-success")
	failure := plannedFixtureJob(t, manifest, "job-failure", "scope-failure")
	executor := func(ctx context.Context, job AnalyzerJob) (analysis.AnalysisResult, error) {
		if job.ScopeID == failure.ScopeID {
			return analysis.AnalysisResult{}, analysis.NewHostError(analysis.ErrAnalyzerFailed, "fixture analyzer failed", nil)
		}
		return fixtureResult(manifest), nil
	}
	snapshot := NewAnalyzerJobScheduler(executor, SchedulerOptions{WorkerCount: 2}).ExecuteAnalyzerPlan(context.Background(), JobPlan{
		PlanVersion:    JobPlanSchemaVersion,
		RepositoryRoot: t.TempDir(),
		Jobs:           []AnalyzerJob{failure, success},
	})
	var successJob, failedJob AnalyzerJob
	for _, job := range snapshot.Jobs {
		switch job.ScopeID {
		case success.ScopeID:
			successJob = job
		case failure.ScopeID:
			failedJob = job
		}
	}
	if successJob.Status != JobComplete || successJob.Result == nil {
		t.Fatalf("independent success = %#v", successJob)
	}
	if failedJob.Status != JobFailed || !hasDiagnostic(failedJob.Diagnostics, string(analysis.ErrAnalyzerFailed)) {
		t.Fatalf("failed job = %#v", failedJob)
	}
	if snapshot.Events[len(snapshot.Events)-1].Type != "run.completed" || snapshot.Events[len(snapshot.Events)-1].Status != JobPartial {
		t.Fatalf("run completion event = %#v", snapshot.Events[len(snapshot.Events)-1])
	}
}

func TestSchedulerFiltersAnalyzerOutputToEffectiveSourceScope(t *testing.T) {
	manifest := fixtureManifest("org.example.fixture", "fixture", "fixture.marker", 1)
	job := plannedFixtureJob(t, manifest, "job-filter", "scope-filter")
	job.EffectiveSourceScope.MatchedLocalPaths = []string{"allowed.src"}
	executor := func(context.Context, AnalyzerJob) (analysis.AnalysisResult, error) {
		return analysis.AnalysisResult{
			RunID:    "filter-run",
			Status:   analysis.StatusComplete,
			Analyzer: analysis.AnalyzerInfo{ID: manifest.ID, Version: manifest.Version, Language: manifest.Language, APIVersion: manifest.APIVersion},
			Project:  analysis.ProjectInfo{RootLabel: "fixture", Boundary: "fixture.marker"},
			Modules: []analysis.ModuleObservation{
				{ID: "allowed", Language: manifest.Language, Kind: "module", Name: "allowed", DisplayName: "allowed", Hierarchy: []string{"allowed"}, SourceReferenceIDs: []string{"source-allowed"}},
				{ID: "removed", Language: manifest.Language, Kind: "module", Name: "removed", DisplayName: "removed", Hierarchy: []string{"removed"}, SourceReferenceIDs: []string{"source-removed"}},
			},
			SourceReferences: []analysis.SourceReference{
				{ID: "source-allowed", Path: "allowed.src", Kind: "definition"},
				{ID: "source-removed", Path: "removed.src", Kind: "definition"},
			},
			References:    []analysis.Reference{{ID: "removed-ref", Name: "removed", Scope: "external", Language: manifest.Language}},
			Relationships: []analysis.RelationshipObservation{{ID: "removed-edge", Type: "depends_on", FromModuleID: "removed", ToReferenceID: "removed-ref", SourceReferenceIDs: []string{"source-removed"}}},
		}, nil
	}
	snapshot := NewAnalyzerJobScheduler(executor, SchedulerOptions{WorkerCount: 1}).ExecuteAnalyzerPlan(context.Background(), JobPlan{
		PlanVersion:    JobPlanSchemaVersion,
		RepositoryRoot: t.TempDir(),
		Jobs:           []AnalyzerJob{job},
	})
	if snapshot.Jobs[0].Status != JobPartial || snapshot.Jobs[0].Result == nil {
		t.Fatalf("filtered job = %#v", snapshot.Jobs[0])
	}
	result := snapshot.Jobs[0].Result
	if len(result.SourceReferences) != 1 || result.SourceReferences[0].Path != "allowed.src" || len(result.Modules) != 1 || result.Modules[0].ID != "allowed" || len(result.Relationships) != 0 || len(result.References) != 0 {
		t.Fatalf("filtered result = %#v", result)
	}
	if !hasDiagnostic(result.Diagnostics, "analysis_scope_filtered") {
		t.Fatalf("filtered result diagnostics = %#v", result.Diagnostics)
	}
}

func TestSchedulerCancelsQueuedAndActiveJobs(t *testing.T) {
	manifest := fixtureManifest("org.example.fixture", "fixture", "fixture.marker", 1)
	jobs := make([]AnalyzerJob, 6)
	for index := range jobs {
		jobs[index] = plannedFixtureJob(t, manifest, "job-"+itoa(index), "scope-"+itoa(index))
	}
	started := make(chan struct{}, 2)
	executor := func(ctx context.Context, job AnalyzerJob) (analysis.AnalysisResult, error) {
		started <- struct{}{}
		<-ctx.Done()
		return analysis.AnalysisResult{}, ctx.Err()
	}
	scheduler := NewAnalyzerJobScheduler(executor, SchedulerOptions{WorkerCount: 2})
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan ExecutionSnapshot, 1)
	go func() {
		result <- scheduler.ExecuteAnalyzerPlan(ctx, JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: t.TempDir(), Jobs: jobs})
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not start a job")
	}
	cancel()
	snapshot := <-result
	for _, job := range snapshot.Jobs {
		if !isTerminal(job.Status) {
			t.Fatalf("job remained non-terminal after cancellation: %#v", job)
		}
		if job.Status != JobCancelled {
			t.Fatalf("job status after cancellation = %q, want cancelled", job.Status)
		}
	}
	if snapshot.Events[len(snapshot.Events)-1].Status != JobCancelled {
		t.Fatalf("run completion status = %q, want cancelled", snapshot.Events[len(snapshot.Events)-1].Status)
	}
}

func TestSessionCacheReusesTerminalScopeWithoutExecutingAnalyzer(t *testing.T) {
	manifest := fixtureManifest("org.example.fixture", "fixture", "fixture.marker", 1)
	job := plannedFixtureJob(t, manifest, "job-cache", "scope-cache")
	job.CacheKey = "sha256:cache-key"
	var calls atomic.Int32
	executor := func(context.Context, AnalyzerJob) (analysis.AnalysisResult, error) {
		calls.Add(1)
		return fixtureResult(manifest), nil
	}
	scheduler := NewAnalyzerJobScheduler(executor, SchedulerOptions{WorkerCount: 1})
	cache := NewSessionCache()
	first := ExecuteAnalyzerPlanWithCache(context.Background(), scheduler, JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: job.ProjectRoot, Jobs: []AnalyzerJob{job}}, cache)
	if calls.Load() != 1 || first.Jobs[0].Status != JobComplete {
		t.Fatalf("first execution calls/status = %d/%s", calls.Load(), first.Jobs[0].Status)
	}
	first.Jobs[0].Result.Modules[0].Hierarchy[0] = "caller-mutated"
	second := ExecuteAnalyzerPlanWithCache(context.Background(), scheduler, JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: job.ProjectRoot, Jobs: []AnalyzerJob{job}}, cache)
	if calls.Load() != 1 || !second.Jobs[0].CacheHit || second.Jobs[0].Status != JobComplete {
		t.Fatalf("cached execution calls/job = %d/%#v", calls.Load(), second.Jobs[0])
	}
	if second.Jobs[0].Result == nil || second.Jobs[0].Result.Modules[0].Hierarchy[0] != "module" {
		t.Fatalf("cache entry was mutated through the first snapshot: %#v", second.Jobs[0].Result)
	}
	run, err := AggregateScopeResults(second)
	if err != nil {
		t.Fatalf("aggregate cached execution: %v", err)
	}
	if len(run.Scopes) != 1 || !run.Scopes[0].CacheHit || run.Scopes[0].CacheKey != job.CacheKey {
		t.Fatalf("cached scope metadata = %#v", run.Scopes)
	}
}

func TestSessionCacheRerunsOnlyScopeWithChangedInputs(t *testing.T) {
	manifest := fixtureManifest("org.example.fixture", "fixture", "fixture.marker", 1)
	firstJob := plannedFixtureJob(t, manifest, "job-cache-a", "scope-cache-a")
	firstJob.CacheKey = "sha256:cache-a"
	secondJob := plannedFixtureJob(t, manifest, "job-cache-b", "scope-cache-b")
	secondJob.CacheKey = "sha256:cache-b"
	var calls atomic.Int32
	executor := func(context.Context, AnalyzerJob) (analysis.AnalysisResult, error) {
		calls.Add(1)
		return fixtureResult(manifest), nil
	}
	scheduler := NewAnalyzerJobScheduler(executor, SchedulerOptions{WorkerCount: 2})
	cache := NewSessionCache()
	initialPlan := JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: firstJob.ProjectRoot, Jobs: []AnalyzerJob{firstJob, secondJob}}
	initial := ExecuteAnalyzerPlanWithCache(context.Background(), scheduler, initialPlan, cache)
	if calls.Load() != 2 {
		t.Fatalf("initial execution calls = %d, want 2", calls.Load())
	}
	if initialPlan.Jobs[0].Status != JobPlanned || initialPlan.Jobs[0].CacheHit || initialPlan.Jobs[0].Result != nil {
		t.Fatalf("cache execution mutated caller plan: %#v", initialPlan.Jobs[0])
	}

	changed := secondJob
	changed.CacheKey = "sha256:cache-b-options-changed"
	changed.EffectiveOptionsFingerprint = "changed-options"
	changed.Options.Fingerprint = "changed-options"
	changed.JobID = jobID(changed)
	second := ExecuteAnalyzerPlanWithCache(context.Background(), scheduler, JobPlan{
		PlanVersion:    JobPlanSchemaVersion,
		RepositoryRoot: firstJob.ProjectRoot,
		Jobs:           []AnalyzerJob{firstJob, changed},
	}, cache)
	if calls.Load() != 3 {
		t.Fatalf("selective reanalysis calls = %d, want 3", calls.Load())
	}
	var cached, rerun AnalyzerJob
	for _, job := range second.Jobs {
		switch job.ScopeID {
		case firstJob.ScopeID:
			cached = job
		case changed.ScopeID:
			rerun = job
		}
	}
	if !cached.CacheHit || cached.Status != JobComplete || rerun.CacheHit || rerun.InvalidationReason != "options_changed" || rerun.Status != JobComplete {
		t.Fatalf("selective cache state = cached=%#v rerun=%#v", cached, rerun)
	}
	if initial.Jobs[0].Result == nil || initial.Jobs[1].Result == nil {
		t.Fatal("initial cache snapshot did not retain both results")
	}
}

func TestSessionCacheReportsAssignmentChangeWhenAnalyzerChanges(t *testing.T) {
	firstManifest := fixtureManifest("org.example.first", "fixture", "fixture.marker", 1)
	first := plannedFixtureJob(t, firstManifest, "job-first", "scope-first")
	first.SelectionSource = SelectionAssignment
	first.AssignmentPath = "."
	first.CacheKey = "sha256:first"
	secondManifest := fixtureManifest("org.example.second", "fixture", "fixture.marker", 1)
	second := plannedFixtureJob(t, secondManifest, "job-second", "scope-second")
	second.SelectionSource = SelectionAssignment
	second.AssignmentPath = "."
	second.CacheKey = "sha256:second"
	var calls atomic.Int32
	scheduler := NewAnalyzerJobScheduler(func(_ context.Context, job AnalyzerJob) (analysis.AnalysisResult, error) {
		calls.Add(1)
		return fixtureResult(job.Manifest), nil
	}, SchedulerOptions{WorkerCount: 1})
	cache := NewSessionCache()
	_ = ExecuteAnalyzerPlanWithCache(context.Background(), scheduler, JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: first.ProjectRoot, Jobs: []AnalyzerJob{first}}, cache)
	next := ExecuteAnalyzerPlanWithCache(context.Background(), scheduler, JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: second.ProjectRoot, Jobs: []AnalyzerJob{second}}, cache)
	if calls.Load() != 2 || len(next.Jobs) != 1 || next.Jobs[0].CacheHit || next.Jobs[0].InvalidationReason != "assignment_changed" {
		t.Fatalf("assignment analyzer change = calls:%d jobs:%#v", calls.Load(), next.Jobs)
	}
}

func TestDeepestAssignmentWinsAndCLIBypassesAssignments(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module fixture\n")
	writeFixtureFile(t, root, "frontend/go.mod", "module frontend\n")
	rootAnalyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.root", "go", "go.mod", 1)}
	frontendAnalyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.frontend", "go", "go.mod", 1)}
	registry := analysis.NewRegistry()
	for _, analyzer := range []*fixtureAnalyzer{rootAnalyzer, frontendAnalyzer} {
		if err := registry.Register(analyzer); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := PlanAnalyzerJobs(context.Background(), registry, PlanRequest{
		RepositoryRoot: root,
		Assignments: []AnalyzerAssignment{
			{ProjectRoot: ".", AnalyzerID: rootAnalyzer.manifest.ID},
			{ProjectRoot: "frontend", AnalyzerID: frontendAnalyzer.manifest.ID},
		},
	})
	if err != nil {
		t.Fatalf("deepest assignment plan: %v", err)
	}
	for _, job := range plan.Jobs {
		if job.RelativeProjectRoot == "frontend" && (job.LogicalAnalyzerID != frontendAnalyzer.manifest.ID || job.AssignmentPath != "frontend") {
			t.Fatalf("frontend assignment did not win: %#v", job)
		}
	}
	cliPlan, err := PlanAnalyzerJobs(context.Background(), registry, PlanRequest{
		RepositoryRoot: root,
		Assignments:    []AnalyzerAssignment{{ProjectRoot: ".", AnalyzerID: rootAnalyzer.manifest.ID}},
		CLISelection:   &ExplicitSelection{ProjectRoot: "frontend", AnalyzerID: rootAnalyzer.manifest.ID},
	})
	if err != nil {
		t.Fatalf("CLI override plan: %v", err)
	}
	for _, job := range cliPlan.Jobs {
		if job.RelativeProjectRoot == "frontend" && (job.SelectionSource != SelectionCLI || job.LogicalAnalyzerID != rootAnalyzer.manifest.ID) {
			t.Fatalf("CLI did not override assignment: %#v", job)
		}
	}
}

func TestUnavailableOrNonMatchingAssignmentDoesNotFallBackToAutomaticDetection(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module fixture\n")
	goAnalyzer := &fixtureAnalyzer{manifest: fixtureManifest("org.example.go", "go", "go.mod", 1)}
	registry := analysis.NewRegistry()
	if err := registry.Register(goAnalyzer); err != nil {
		t.Fatal(err)
	}

	for _, assignment := range []AnalyzerAssignment{
		{ProjectRoot: ".", AnalyzerID: "org.example.missing"},
		{ProjectRoot: ".", AnalyzerID: "org.example.go", Language: "python"},
	} {
		plan, err := PlanAnalyzerJobs(context.Background(), registry, PlanRequest{
			RepositoryRoot: root,
			Assignments:    []AnalyzerAssignment{assignment},
		})
		if err != nil {
			t.Fatalf("assignment %#v: %v", assignment, err)
		}
		if len(plan.Jobs) != 1 || plan.Jobs[0].SelectionSource != SelectionAssignment {
			t.Fatalf("assignment %#v fell back to automatic detection: %#v", assignment, plan.Jobs)
		}
		if plan.Jobs[0].Result != nil || len(plan.Jobs[0].Diagnostics) == 0 {
			t.Fatalf("assignment %#v did not retain a diagnostic-ready scope: %#v", assignment, plan.Jobs[0])
		}
	}
}

func TestAggregateNamespacesObservationsAndPreservesScopeParity(t *testing.T) {
	manifest := fixtureManifest("org.example.fixture", "fixture", "fixture.marker", 1)
	first := plannedFixtureJob(t, manifest, "job-a", "scope-a")
	second := plannedFixtureJob(t, manifest, "job-b", "scope-b")
	firstResult := namespacedFixtureResult(manifest)
	secondResult := namespacedFixtureResult(manifest)
	first.Result = &firstResult
	second.Result = &secondResult
	first.Status = JobComplete
	second.Status = JobComplete
	run, err := AggregateScopeResults(ExecutionSnapshot{RunID: "run-fixture", Plan: JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: t.TempDir(), Jobs: []AnalyzerJob{first, second}}, Jobs: []AnalyzerJob{first, second}})
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	if run.Status != analysis.StatusComplete || run.Model == nil || len(run.Model.Modules) != 4 {
		t.Fatalf("aggregate status/model = %q %#v", run.Status, run.Model)
	}
	moduleIDs := map[string]bool{}
	for _, module := range run.Model.Modules {
		moduleIDs[module.ID] = true
	}
	if !moduleIDs["scope-a::module"] || !moduleIDs["scope-b::module"] || !moduleIDs["scope-a::target"] || !moduleIDs["scope-b::target"] {
		t.Fatalf("namespaced module IDs = %#v", moduleIDs)
	}
	for _, relationship := range run.Model.Relationships {
		if relationship.FromModuleID[:len("scope-")] != "scope-" || relationship.ToModuleID[:len("scope-")] != "scope-" {
			t.Fatalf("relationship was not namespaced: %#v", relationship)
		}
		if relationship.FromModuleID[:8] != relationship.ToModuleID[:8] {
			t.Fatalf("relationship crossed scopes: %#v", relationship)
		}
	}
	selected, err := run.SelectAnalysisScope("scope-a")
	if err != nil {
		t.Fatalf("select scope: %v", err)
	}
	if len(selected.Model.Modules) != 2 || selected.Model.Project.Language != "fixture" {
		t.Fatalf("selected scope model = %#v", selected.Model)
	}
	cached, err := run.ScopeResult("scope-a")
	if err != nil {
		t.Fatalf("cached scope result: %v", err)
	}
	if cached.Modules[0].ID == "scope-a::module" || strings.HasPrefix(cached.Modules[0].ID, "scope-a::") {
		t.Fatalf("aggregate namespacing mutated cached scope result: %#v", cached.Modules)
	}
	if _, err := run.SelectAnalysisScope("scope-missing"); analysis.ErrorCodeOf(err) != analysis.ErrAnalysisScopeNotFound {
		t.Fatalf("missing scope error = %v", err)
	}
	subset, err := run.CombinedCanonicalModelForScopes([]string{"scope-b", "scope-a"})
	if err != nil {
		t.Fatalf("combine selected scopes: %v", err)
	}
	if len(subset.Modules) != 4 {
		t.Fatalf("selected-scope aggregate modules = %#v", subset.Modules)
	}
	if _, err := run.CombinedCanonicalModelForScopes([]string{"scope-missing"}); analysis.ErrorCodeOf(err) != analysis.ErrAnalysisScopeNotFound {
		t.Fatalf("missing combined scope error = %v", err)
	}
}

func TestAggregateStatusRulesAndSourceIdentity(t *testing.T) {
	manifest := fixtureManifest("org.example.fixture", "fixture", "fixture.marker", 1)
	success := plannedFixtureJob(t, manifest, "job-success", "scope-success")
	successResult := fixtureResult(manifest)
	success.Result = &successResult
	success.Status = JobComplete
	success.EffectiveSourceScope.PolicyFingerprint = "sha256:policy-a"
	success.MatchedSourceSetFingerprint = "sha256:sources-a"
	failed := plannedFixtureJob(t, manifest, "job-failed", "scope-failed")
	failed.Status = JobFailed
	failed.Diagnostics = []analysis.Diagnostic{{Code: "job_failed", Severity: "error", Message: "fixture failure", Recoverable: true}}
	run, err := AggregateScopeResults(ExecutionSnapshot{RunID: "run-status", Plan: JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: t.TempDir(), Jobs: []AnalyzerJob{success, failed}}, Jobs: []AnalyzerJob{success, failed}})
	if err != nil {
		t.Fatalf("partial aggregate: %v", err)
	}
	if run.Status != analysis.StatusPartial || run.Model == nil || run.Summary.UsableScopeCount != 1 || run.Summary.FailedScopeCount != 1 {
		t.Fatalf("partial aggregate = %#v", run)
	}
	for _, scope := range run.Scopes {
		if scope.ScopeID == success.ScopeID && scope.SourceScope.MatchedSourceSetFingerprint != "sha256:sources-a" {
			t.Fatalf("source identity lost: %#v", scope)
		}
	}
	cancelled := success
	cancelled.Status = JobCancelled
	cancelled.Result = nil
	run, err = AggregateScopeResults(ExecutionSnapshot{RunID: "run-cancelled", Plan: JobPlan{PlanVersion: JobPlanSchemaVersion, RepositoryRoot: t.TempDir(), Jobs: []AnalyzerJob{cancelled}}, Jobs: []AnalyzerJob{cancelled}})
	if err != nil {
		t.Fatalf("cancelled aggregate: %v", err)
	}
	if run.Status != analysis.StatusCancelled || run.Model != nil {
		t.Fatalf("cancelled aggregate = %#v", run)
	}
}

func plannedFixtureJob(t *testing.T, manifest analysis.Manifest, jobID, scopeID string) AnalyzerJob {
	t.Helper()
	root := t.TempDir()
	writeFixtureFile(t, root, "fixture.marker", "fixture\n")
	return AnalyzerJob{
		JobID: jobID, ScopeID: scopeID, ProjectRoot: root, RelativeProjectRoot: ".", LogicalAnalyzerID: manifest.ID,
		AnalyzerVersion: manifest.Version, Language: manifest.Language, EffectiveOptionsFingerprint: "fixture-options", Status: JobPlanned,
		Options:              analysis.EffectiveOptions{Values: map[string]any{"exclude": []string{}}, Fingerprint: "fixture-options"},
		EffectiveSourceScope: analysis.SourceScope{PolicyFingerprint: "sha256:policy", MatchedSourceSetFingerprint: "sha256:sources", MatchedPaths: []string{}},
		Manifest:             manifest, Diagnostics: []analysis.Diagnostic{},
	}
}

func namespacedFixtureResult(manifest analysis.Manifest) analysis.AnalysisResult {
	result := fixtureResult(manifest)
	result.Modules = append(result.Modules, analysis.ModuleObservation{ID: "target", Language: manifest.Language, Kind: "module", Name: "target", DisplayName: "target", Hierarchy: []string{"target"}, SourceReferenceIDs: []string{"source-target"}})
	result.SourceReferences = append(result.SourceReferences, analysis.SourceReference{ID: "source-target", Path: "target.src", Kind: "definition"})
	result.Relationships = []analysis.RelationshipObservation{{ID: "relationship", Type: "depends_on", FromModuleID: "module", ToModuleID: "target", SourceReferenceIDs: []string{"source"}}}
	return result
}

func writeFixtureFile(t *testing.T, root, relative, value string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func hasDiagnostic(values []analysis.Diagnostic, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}

func itoa(value int) string {
	data, _ := json.Marshal(value)
	return string(data)
}
