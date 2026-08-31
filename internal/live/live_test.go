package live

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/analysis/sourceindex"
	gosyntax "github.com/buffo/arch-view/internal/analysis/syntax/go"
	goanalyzer "github.com/buffo/arch-view/internal/analyzers/go"
	"github.com/buffo/arch-view/internal/analyzers/go/sourcefacts"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/quality"
)

func TestValidateLiveSessionRejectsUnsafeConfiguration(t *testing.T) {
	root := t.TempDir()
	config := testLiveConfig("session:validation")
	config.WatchRoots = []WatchRoot{{Path: "../outside", Recursive: true}}
	_, err := ValidateLiveSession(context.Background(), config, root, ValidationDependencies{})
	assertLiveErrorCode(t, err, ErrorWatchRootInvalid)

	config = testLiveConfig("session:permissions")
	config.PermissionPolicy.AllowShell = true
	_, err = ValidateLiveSession(context.Background(), config, root, ValidationDependencies{})
	assertLiveErrorCode(t, err, ErrorLiveConfigInvalid)

	config = testLiveConfig("session:absolute")
	config.RepositoryRoot = filepath.Join(root, "child")
	_, err = ValidateLiveSession(context.Background(), config, root, ValidationDependencies{})
	assertLiveErrorCode(t, err, ErrorWatchRootInvalid)

	config = testLiveConfig("session:unknown-analyzer")
	config.AnalyzerIDs = []string{"analyzer:missing"}
	_, err = ValidateLiveSession(context.Background(), config, root, ValidationDependencies{})
	assertLiveErrorCode(t, err, ErrorLiveConfigInvalid)

	config = testLiveConfig("session:unknown-profile")
	config.QualityRequest = &QualityRequest{ProfileID: "profile:missing", ProfileVersion: "1.0.0"}
	_, err = ValidateLiveSession(context.Background(), config, root, ValidationDependencies{})
	assertLiveErrorCode(t, err, ErrorLiveConfigInvalid)
}

func TestValidateLiveSessionNormalizesDefaultsAndReferences(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := testLiveConfig("session:defaults")
	config.WatchRoots = []WatchRoot{{Path: "src", Recursive: true}}
	validated, err := ValidateLiveSession(context.Background(), config, root, ValidationDependencies{})
	if err != nil {
		t.Fatalf("ValidateLiveSession() error = %v", err)
	}
	if validated.Config.WatchPolicy.MaxPendingEvents != DefaultMaxPendingEvents || validated.Config.QueryPolicy.DefaultMaxItems != DefaultQueryMaxItems {
		t.Fatalf("defaults were not applied: %#v", validated.Config)
	}
	if validated.Config.PermissionPolicy.DefaultMode != "read_only" || validated.Config.PermissionPolicy.AllowShell || validated.Config.PermissionPolicy.AllowTargetExecution {
		t.Fatalf("unsafe defaults: %#v", validated.Config.PermissionPolicy)
	}
}

func TestMultiAnalyzerScannerHonorsSourceIndexRequest(t *testing.T) {
	root := t.TempDir()
	writeLiveFixture(t, filepath.Join(root, "go.mod"), "module example.com/live\ngo 1.22\n")
	writeLiveFixture(t, filepath.Join(root, "main.go"), "package main\n\nfunc Main() {}\n")

	registry := analysis.NewRegistry()
	if err := registry.Register(goanalyzer.New()); err != nil {
		t.Fatalf("register Go analyzer: %v", err)
	}
	config := testLiveConfig("session:scanner-policy")
	config.SourceIndexRequest = SourceIndexRequest{Enabled: false, Capabilities: []string{sourceindex.CapabilityFiles, sourceindex.CapabilityDeclarations}}
	disabledScanner := NewMultiAnalyzerScanner(analysis.NewHost(registry), nil, nil)
	disabled, err := disabledScanner.Scan(context.Background(), ScanRequest{SessionID: config.SessionID, RepositoryRoot: root, Config: config})
	if err != nil {
		t.Fatalf("disabled live scan: %v", err)
	}
	if disabled.Model.SourceIndex != nil {
		t.Fatalf("disabled live source index = %#v, want nil", disabled.Model.SourceIndex)
	}

	config.SourceIndexRequest = SourceIndexRequest{Enabled: true, Capabilities: []string{sourceindex.CapabilityFiles, sourceindex.CapabilitySize}}
	enabledScanner := NewMultiAnalyzerScanner(analysis.NewHost(registry), nil, nil)
	enabled, err := enabledScanner.Scan(context.Background(), ScanRequest{SessionID: config.SessionID, RepositoryRoot: root, Config: config})
	if err != nil {
		t.Fatalf("enabled live scan: %v", err)
	}
	if enabled.Model.SourceIndex == nil || len(enabled.Model.SourceIndex.Snapshots) != 1 {
		t.Fatalf("enabled live source index = %#v, want one snapshot", enabled.Model.SourceIndex)
	}
	requested := enabled.Model.SourceIndex.Snapshots[0].Input.RequestedCapabilities
	if len(requested) != 2 || requested[0] != sourceindex.CapabilityFiles || requested[1] != sourceindex.CapabilitySize {
		t.Fatalf("live requested capabilities = %#v", requested)
	}
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeLiveFixture(t, filepath.Join(root, "src", "watched.go"), "package watched\n\nfunc Watched() {}\n")
	config.WatchRoots = []WatchRoot{{Path: "src", Recursive: true}}
	restrictedInput, err := (FileSystemFingerprinter{}).Fingerprint(context.Background(), root, config.WatchRoots)
	if err != nil {
		t.Fatal(err)
	}
	restricted, err := enabledScanner.Scan(context.Background(), ScanRequest{SessionID: config.SessionID, RepositoryRoot: root, Config: config, InputFingerprint: restrictedInput, Invalidation: InvalidationPlan{Mode: "full_rescan"}})
	if err != nil {
		t.Fatalf("watch-root-restricted live scan: %v", err)
	}
	files := restricted.Model.SourceIndex.Snapshots[0].Files
	if len(files) != 1 || files[0].Path != "src/watched.go" {
		t.Fatalf("watch roots and analyzer scope diverged: %#v", files)
	}
	if err := os.Mkdir(filepath.Join(root, "src", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeLiveFixture(t, filepath.Join(root, "src", "nested", "ignored.go"), "package nested\n")
	config.WatchRoots[0].Recursive = false
	nonRecursiveInput, err := (FileSystemFingerprinter{}).Fingerprint(context.Background(), root, config.WatchRoots)
	if err != nil {
		t.Fatal(err)
	}
	nonRecursive, err := enabledScanner.Scan(context.Background(), ScanRequest{SessionID: config.SessionID, RepositoryRoot: root, Config: config, InputFingerprint: nonRecursiveInput, Invalidation: InvalidationPlan{Mode: "full_rescan"}})
	if err != nil {
		t.Fatalf("non-recursive live scan: %v", err)
	}
	files = nonRecursive.Model.SourceIndex.Snapshots[0].Files
	if len(files) != 1 || files[0].Path != "src/watched.go" {
		t.Fatalf("non-recursive watch root leaked descendants: %#v", files)
	}
}

func TestNormalizeWatchEventAndCoalesceEvents(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	roots := []WatchRoot{{Path: "src", Recursive: true}}
	event, err := NormalizeWatchEvent(WatchEvent{BackendID: "fake", Kind: "modify", Path: "src\\main.go", Sequence: "1"}, root, roots)
	if err != nil || event.Path != "src/main.go" || event.Kind != "modify" || event.Sequence != "1" {
		t.Fatalf("normalized event = %#v, err = %v", event, err)
	}
	_, err = NormalizeWatchEvent(WatchEvent{Kind: "modify", Path: "../secret.go"}, root, roots)
	assertLiveErrorCode(t, err, "WatchEventOutOfScope")
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = NormalizeWatchEvent(WatchEvent{Kind: "modify", Path: "nested/deep.go"}, root, []WatchRoot{{Path: ".", Recursive: false}})
	assertLiveErrorCode(t, err, "WatchEventOutOfScope")

	coalescer := NewEventCoalescer(WatchPolicy{DebounceMS: 100, MaxPendingEvents: 8})
	now := time.Unix(0, 0)
	coalescer.Add(NormalizedEvent{BackendID: "fake", Kind: "create", Path: "src/main.go", Sequence: "1"}, now)
	coalescer.Add(NormalizedEvent{BackendID: "fake", Kind: "modify", Path: "src/main.go", Sequence: "2"}, now.Add(time.Millisecond))
	if group := coalescer.FlushIfReady(now.Add(99 * time.Millisecond)); group != nil {
		t.Fatal("coalescer flushed before debounce window")
	}
	group := coalescer.FlushIfReady(now.Add(101 * time.Millisecond))
	if group == nil || len(group.Events) != 1 || group.Events[0].Kind != "create" || group.ChangedPaths[0] != "src/main.go" || group.FullRescan {
		t.Fatalf("coalesced group = %#v", group)
	}
	if group.Events[0].EventGroupID != group.ID {
		t.Fatalf("event group identity was not propagated: %#v", group)
	}

	coalescer.Add(NormalizedEvent{BackendID: "fake", Kind: "modify", Path: "src/a.go", Sequence: "4"}, now)
	coalescer.Add(NormalizedEvent{BackendID: "fake", Kind: "modify", Path: "src/b.go", Sequence: "6"}, now)
	group = coalescer.Flush()
	if group == nil || !group.FullRescan || group.Reason != "watch_sequence_gap" {
		t.Fatalf("sequence gap did not force rescan: %#v", group)
	}

	coalescer.Add(NormalizedEvent{BackendID: "fake", Kind: "rename", Path: "src/new.go", Sequence: "7"}, now)
	group = coalescer.Flush()
	if group == nil || !group.FullRescan || group.Reason != "ambiguous_rename" {
		t.Fatalf("ambiguous rename did not force rescan: %#v", group)
	}

	bounded := NewEventCoalescer(WatchPolicy{DebounceMS: 100, MaxPendingEvents: 1})
	bounded.Add(NormalizedEvent{BackendID: "fake", Kind: "modify", Path: "src/a.go", Sequence: "1"}, now)
	bounded.Add(NormalizedEvent{BackendID: "fake", Kind: "modify", Path: "src/b.go", Sequence: "2"}, now.Add(time.Millisecond))
	group = bounded.FlushIfReady(now.Add(101 * time.Millisecond))
	if group == nil || !group.FullRescan || group.Reason != "pending_event_limit_exceeded" {
		t.Fatalf("pending event bound did not force a flushable rescan: %#v", group)
	}
	numeric := NewEventCoalescer(WatchPolicy{MaxPendingEvents: 4})
	numeric.Add(NormalizedEvent{Kind: "modify", Path: "src/ten.go", Sequence: "10"}, now)
	numeric.Add(NormalizedEvent{Kind: "modify", Path: "src/two.go", Sequence: "2"}, now)
	group = numeric.Flush()
	if group.FirstSequence != "2" || group.LastSequence != "10" {
		t.Fatalf("numeric sequence bounds = %q..%q", group.FirstSequence, group.LastSequence)
	}
}

func TestFileSystemFingerprintExcludesAnalyzerIneligibleDirectories(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "objects"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeLiveFixture(t, filepath.Join(root, "main.go"), "package main\n")
	writeLiveFixture(t, filepath.Join(root, ".git", "objects", "noise"), "first")
	fingerprinter := FileSystemFingerprinter{}
	first, err := fingerprinter.Fingerprint(context.Background(), root, []WatchRoot{{Path: ".", Recursive: true}})
	if err != nil {
		t.Fatal(err)
	}
	writeLiveFixture(t, filepath.Join(root, ".git", "objects", "noise"), "second")
	second, err := fingerprinter.Fingerprint(context.Background(), root, []WatchRoot{{Path: ".", Recursive: true}})
	if err != nil {
		t.Fatal(err)
	}
	if !inputEqual(first, second) || len(second.Files) != 1 || second.Files[0].Path != "main.go" {
		t.Fatalf("excluded directory changed fingerprint: first=%#v second=%#v", first, second)
	}
}

func TestPlanInvalidationIsConservative(t *testing.T) {
	group := EventGroup{ID: "group:1", ChangedPaths: []string{"go/main.go"}}
	plan := PlanInvalidation(group, []ScopeIdentity{{ScopeID: "scope:go", ProjectRoot: "go"}, {ScopeID: "scope:python", ProjectRoot: "python"}})
	if plan.Mode != "selective" || len(plan.AffectedScopeIDs) != 1 || plan.AffectedScopeIDs[0] != "scope:go" {
		t.Fatalf("selective plan = %#v", plan)
	}
	plan = PlanInvalidation(EventGroup{ID: "group:2", ChangedPaths: []string{"shared/main.go"}}, []ScopeIdentity{{ScopeID: "scope:a", ProjectRoot: "."}, {ScopeID: "scope:b", ProjectRoot: "."}})
	if plan.Mode != "full_rescan" || plan.Reason != "scope_or_dependency_impact_uncertain" {
		t.Fatalf("ambiguous plan = %#v", plan)
	}
}

func TestMemorySnapshotStorePublishesMonotonicImmutableRevisions(t *testing.T) {
	store := NewMemorySnapshotStore()
	first := &RevisionRecord{SessionID: "session:test", Snapshot: LiveSnapshot{SnapshotID: "snapshot:one", SemanticDigest: digestJSON("one")}}
	published, err := store.PublishAtomically(first)
	if err != nil || published.Snapshot.Revision != 1 {
		t.Fatalf("first publish = %#v, err = %v", published, err)
	}
	first.Snapshot.SnapshotID = "mutated-after-publish"
	second := &RevisionRecord{SessionID: "session:test", Snapshot: LiveSnapshot{SnapshotID: "snapshot:two", SemanticDigest: digestJSON("two")}}
	published, err = store.PublishAtomically(second)
	if err != nil || published.Snapshot.Revision != 2 {
		t.Fatalf("second publish = %#v, err = %v", published, err)
	}
	equal := &RevisionRecord{SessionID: "session:test", Snapshot: LiveSnapshot{SnapshotID: "snapshot:ignored", SemanticDigest: digestJSON("two")}}
	published, err = store.PublishAtomically(equal)
	if err != nil || published.Snapshot.Revision != 2 || published.Snapshot.SnapshotID != "snapshot:two" {
		t.Fatalf("equal publish created noise: %#v, err = %v", published, err)
	}
	read, ok := store.ReadRevision("session:test", 1)
	if !ok || read.Snapshot.SnapshotID != "snapshot:one" {
		t.Fatalf("immutable revision was changed: %#v", read)
	}
}

func TestRevisionValidationRejectsCrossArtifactMismatch(t *testing.T) {
	record := buildRevisionRecord("session:coherence", ScanResult{Model: testModel()}, testInput("coherent"), 1, false)
	if err := validateRevisionCandidate(record); err != nil {
		t.Fatalf("coherent candidate rejected: %v", err)
	}
	record.Snapshot.ModelRef.ID = "model:other"
	assertLiveErrorCode(t, validateRevisionCandidate(record), ErrorSnapshotValidation)
}

func TestMemorySnapshotStoreWakesWaiterStartedBeforeFirstPublish(t *testing.T) {
	store := NewMemorySnapshotStore()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan *RevisionRecord, 1)
	errors := make(chan error, 1)
	go func() {
		record, err := store.WaitForRevision(ctx, "session:wait", 1)
		result <- record
		errors <- err
	}()

	candidate := &RevisionRecord{SessionID: "session:wait", Snapshot: LiveSnapshot{SnapshotID: "snapshot:first", SemanticDigest: digestJSON("first")}}
	if _, err := store.PublishAtomically(candidate); err != nil {
		t.Fatal(err)
	}
	if err := <-errors; err != nil {
		t.Fatalf("WaitForRevision() error = %v", err)
	}
	if record := <-result; record == nil || record.Snapshot.Revision != 1 {
		t.Fatalf("WaitForRevision() = %#v", record)
	}
}

func TestValidatedConfigurationIsDefensivelyCopied(t *testing.T) {
	root := t.TempDir()
	config := testLiveConfig("session:immutable-config")
	config.SourceIndexRequest.Capabilities = []string{"source:files"}
	validated, err := ValidateLiveSession(context.Background(), config, root, ValidationDependencies{})
	if err != nil {
		t.Fatal(err)
	}
	config.SourceIndexRequest.Capabilities[0] = "source:mutated"
	if validated.Config.SourceIndexRequest.Capabilities[0] != "source:files" {
		t.Fatalf("validated config retained caller-owned storage: %#v", validated.Config.SourceIndexRequest.Capabilities)
	}
}

func TestBuildInvalidationArrivingDuringBuildIsQueued(t *testing.T) {
	done := make(chan struct{})
	session := &LiveSession{building: true, buildDone: done}
	joined := session.beginRebuild(InvalidationPlan{Mode: "selective", AffectedPaths: []string{"src/new.go"}, Reason: "watch_event"}, nil)
	if joined != done || session.pendingPlan == nil {
		t.Fatalf("invalidation was not queued: joined=%v pending=%#v", joined == done, session.pendingPlan)
	}
	if session.pendingPlan.Mode != "selective" || len(session.pendingPlan.AffectedPaths) != 1 {
		t.Fatalf("queued invalidation = %#v", session.pendingPlan)
	}
}

func TestStartLiveSessionReportsInitializingThenReady(t *testing.T) {
	root := t.TempDir()
	released := make(chan struct{})
	scanner := &blockingScanner{result: ScanResult{Run: orchestration.AnalysisRun{Status: analysis.StatusComplete}, Model: testModel()}, release: released}
	session, err := StartLiveSession(context.Background(), testLiveConfig("session:lifecycle"), root, SessionOptions{Scanner: scanner, Fingerprinter: StaticFingerprinter{Value: testInput("one")}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	status := session.Status()
	if status.State != SessionInitializing || status.Freshness.Status != FreshnessInitializing {
		t.Fatalf("initial status = %#v", status)
	}
	statusEnvelope, err := session.GetSnapshotStatus(context.Background(), QueryRequest{Consistency: ConsistencyLatestReady})
	if err != nil {
		t.Fatalf("initial snapshot status error = %v", err)
	}
	if statusEnvelope.ReturnedConsistency != "unavailable" || len(statusEnvelope.Diagnostics) != 1 || statusEnvelope.Diagnostics[0].Code != ErrorNoReadySnapshot {
		t.Fatalf("initial snapshot status envelope = %#v", statusEnvelope)
	}
	close(released)
	if err := session.Wait(context.Background()); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	status = session.Status()
	if status.State != SessionReady || status.Snapshot == nil || status.Snapshot.Revision != 1 {
		t.Fatalf("ready status = %#v", status)
	}
}

func TestStartLiveSessionNoReadyFailureAndCancellation(t *testing.T) {
	root := t.TempDir()
	failure, err := StartLiveSession(context.Background(), testLiveConfig("session:failure"), root, SessionOptions{Scanner: StaticScanner{Err: errors.New("scanner failed")}, Fingerprinter: StaticFingerprinter{Value: testInput("one")}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = failure.Close() })
	if err := failure.Wait(context.Background()); err == nil {
		t.Fatal("failed initial scan returned nil error")
	}
	if _, err := failure.CurrentRecord(ConsistencyLatestReady, 0); err == nil {
		t.Fatal("failed session exposed a ready revision")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = StartLiveSession(ctx, testLiveConfig("session:cancelled"), root, SessionOptions{Scanner: StaticScanner{Result: ScanResult{Model: testModel()}}, Fingerprinter: StaticFingerprinter{Value: testInput("one")}})
	if err == nil {
		t.Fatal("cancelled session configuration was accepted")
	}
}

func TestRequireCurrentReconcilesAndJoinsOneBuild(t *testing.T) {
	root := t.TempDir()
	fingerprinter := &mutableFingerprinter{value: testInput("one")}
	scanner := &countingScanner{result: ScanResult{Model: testModel()}}
	config := testLiveConfig("session:freshness")
	config.FreshnessPolicy.DefaultConsistency = ConsistencyLatestReady
	config.FreshnessPolicy.MaxWaitMS = 2_000
	session, err := StartLiveSession(context.Background(), config, root, SessionOptions{Scanner: scanner, Fingerprinter: fingerprinter})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	initialCount := scanner.Count()
	fingerprinter.Set(testInput("two"))
	var waitGroup sync.WaitGroup
	results := make(chan *RevisionRecord, 2)
	errors := make(chan error, 2)
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			record, currentErr := session.EnsureCurrentSnapshot(context.Background())
			results <- record
			errors <- currentErr
		}()
	}
	waitGroup.Wait()
	close(results)
	close(errors)
	for currentErr := range errors {
		if currentErr != nil {
			t.Fatalf("EnsureCurrentSnapshot() error = %v", currentErr)
		}
	}
	var revision int
	for record := range results {
		if record == nil || record.Snapshot.Revision != 2 {
			t.Fatalf("current record = %#v", record)
		}
		revision = record.Snapshot.Revision
	}
	if revision != 2 || scanner.Count() != initialCount+1 {
		t.Fatalf("strict requests did not join one build: revision=%d scans=%d initial=%d", revision, scanner.Count(), initialCount)
	}
	_, freshness, _, err := session.prepareConsistency(context.Background(), ConsistencySpecific, 1)
	if err != nil {
		t.Fatal(err)
	}
	if freshness.Status != FreshnessStale || freshness.LastReadyRevision != 2 {
		t.Fatalf("older specific revision freshness = %#v", freshness)
	}
}

func TestWatcherStartupReconcilesTheInitialSnapshotGap(t *testing.T) {
	root := t.TempDir()
	fingerprinter := &mutableFingerprinter{value: testInput("initial")}
	calls := make(chan int, 4)
	scanner := &countingScanner{result: ScanResult{Model: testModel()}, calls: calls}
	watcher := &mutatingWatchBackend{fingerprinter: fingerprinter, value: testInput("changed-before-start")}
	config := testLiveConfig("session:watcher-start-gap")
	config.FreshnessPolicy.MaxWaitMS = 2_000
	session, err := StartLiveSession(context.Background(), config, root, SessionOptions{Scanner: scanner, Fingerprinter: fingerprinter, Watcher: watcher, StartWatcher: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	for expected := 1; expected <= 2; expected++ {
		select {
		case call := <-calls:
			if call != expected {
				t.Fatalf("scan call = %d, want %d", call, expected)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for scan %d", expected)
		}
	}
	current, err := session.EnsureCurrentSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if current.Snapshot.Revision != 2 || scanner.Count() != 2 {
		t.Fatalf("watcher-start reconciliation = revision %d, scans %d", current.Snapshot.Revision, scanner.Count())
	}
}

func TestRevisionBoundQueriesSearchAndSourceContext(t *testing.T) {
	root := t.TempDir()
	content := "package main\n\n// Main returns a value.\nfunc Main() int {\n\treturn 42\n}\n"
	otherContent := "package main\n\nvar Other = 1\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "other.go"), []byte(otherContent), 0o644); err != nil {
		t.Fatal(err)
	}
	index, _, err := sourceindex.BuildSourceIndex(context.Background(), sourceindex.BuildInput{
		Scope:    analysis.ScopeContext{ScopeID: "scope:test", ProjectRoot: ".", Mode: analysis.SourceIndexScopeMode},
		Producer: analysis.ProducerContext{AnalyzerID: "analyzer:test", AnalyzerVersion: "1.0.0"},
		Files: []sourceindex.SourceFileInput{
			{Path: "main.go", Content: []byte(content), Language: analysis.LanguageRef{ID: "language:go"}, Roles: []string{"role:source"}, ModuleID: "example.com/main"},
			{Path: "other.go", Content: []byte(otherContent), Language: analysis.LanguageRef{ID: "language:go"}, Roles: []string{"role:source"}, ModuleID: "example.com/main"},
		},
		RequestedCapabilities: []string{sourceindex.CapabilityDeclarations, sourceindex.CapabilityDocumentation, sourceindex.CapabilityFiles, sourceindex.CapabilitySize, sourceindex.CapabilityVisibility},
		SyntaxProvider:        gosyntax.NewProvider(),
		Extractors:            sourcefacts.NewRegistry(),
	})
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	modelValue, err := canonical.Normalize(analysis.AnalysisResult{Status: analysis.StatusComplete, Analyzer: analysis.AnalyzerInfo{ID: "analyzer:test", Version: "1.0.0", Language: "go", APIVersion: analysis.AnalyzerAPIVersion}, Project: analysis.ProjectInfo{RootLabel: "test", Boundary: "repository"}, Modules: []analysis.ModuleObservation{}, References: []analysis.Reference{}, SourceReferences: []analysis.SourceReference{}, Relationships: []analysis.RelationshipObservation{}, Diagnostics: []analysis.Diagnostic{}, SourceIndex: &index})
	if err != nil {
		t.Fatalf("normalize source model: %v", err)
	}
	input := testInput("source-query")
	queryConfig := testLiveConfig("session:queries")
	queryConfig.SourceIndexRequest = SourceIndexRequest{Enabled: true, Capabilities: []string{sourceindex.CapabilityDeclarations, sourceindex.CapabilityDocumentation, sourceindex.CapabilityFiles, sourceindex.CapabilitySize, sourceindex.CapabilityVisibility}}
	session, err := StartLiveSession(context.Background(), queryConfig, root, SessionOptions{Scanner: StaticScanner{Result: ScanResult{Model: modelValue}}, Fingerprinter: StaticFingerprinter{Value: input}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}

	fileRequest := QueryRequest{Consistency: ConsistencySpecific, Revision: 1, MaxItems: 1, MaxBytes: 4096}
	files, err := session.FindFiles(context.Background(), fileRequest)
	if err != nil {
		t.Fatalf("FindFiles() error = %v", err)
	}
	filePage, ok := files.Result.(ItemPage[analysis.FileRecord])
	if !ok || len(filePage.Items) != 1 || filePage.Items[0].Path != "main.go" {
		t.Fatalf("file result = %#v", files.Result)
	}
	if files.ReturnedConsistency != string(ConsistencySpecific) || files.Revision != 1 {
		t.Fatalf("file envelope = %#v", files)
	}
	if files.NextCursor == "" || filePage.Total != 2 {
		t.Fatalf("first file page did not expose a continuation: %#v", files)
	}
	fileRequest.Cursor = files.NextCursor
	secondFiles, err := session.FindFiles(context.Background(), fileRequest)
	if err != nil {
		t.Fatalf("second FindFiles() page error = %v", err)
	}
	secondFilePage := secondFiles.Result.(ItemPage[analysis.FileRecord])
	if len(secondFilePage.Items) != 1 || secondFilePage.Items[0].Path != "other.go" || secondFiles.NextCursor != "" {
		t.Fatalf("second file page = %#v", secondFiles.Result)
	}
	record, err := session.QuerySpecificRevision(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	crossOperationRequest := QueryRequest{Consistency: ConsistencySpecific, Revision: 1, MaxItems: 1, MaxBytes: 4096}
	crossOperationRequest.Cursor = encodeQueryCursor(queryContext(record, crossOperationRequest, "find_files", 4096, 1), 1)
	_, err = session.FindSymbols(context.Background(), crossOperationRequest)
	assertLiveErrorCode(t, err, ErrorQueryCursor)

	symbols, err := session.FindSymbols(context.Background(), QueryRequest{Consistency: ConsistencySpecific, Revision: 1, Query: StructuralQuery{Name: "Main", ScopeIDs: []string{"scope:test"}}, MaxItems: 10, MaxBytes: 4096})
	if err != nil {
		t.Fatalf("FindSymbols() error = %v", err)
	}
	symbolPage, ok := symbols.Result.(ItemPage[analysis.SymbolRecord])
	if !ok || len(symbolPage.Items) == 0 || symbolPage.Items[0].Name != "Main" {
		t.Fatalf("symbol result = %#v", symbols.Result)
	}

	textMatches, err := session.SearchExactText(context.Background(), TextSearchQuery{Consistency: ConsistencySpecific, Revision: 1, Pattern: "return 42", Mode: "literal", ScopeIDs: []string{"scope:test"}, MaxItems: 10, MaxBytes: 4096})
	if err != nil {
		t.Fatalf("SearchExactText() error = %v", err)
	}
	textPage, ok := textMatches.Result.(ItemPage[TextMatch])
	if !ok || len(textPage.Items) != 1 || textPage.Items[0].Line != 5 || textPage.Items[0].Path != "main.go" {
		t.Fatalf("text result = %#v", textMatches.Result)
	}
	_, err = session.SearchExactText(context.Background(), TextSearchQuery{Consistency: ConsistencySpecific, Revision: 1, Pattern: "(", Mode: "regex", MaxItems: 10, MaxBytes: 4096})
	assertLiveErrorCode(t, err, "QueryInvalid")
	_, err = session.SearchExactText(context.Background(), TextSearchQuery{Consistency: ConsistencySpecific, Revision: 1, Pattern: "return", Mode: "literal", PathGlob: "../*", MaxItems: 10, MaxBytes: 4096})
	assertLiveErrorCode(t, err, "QueryInvalid")
	_, err = session.FindFiles(context.Background(), QueryRequest{Consistency: ConsistencySpecific, Revision: 1, Query: StructuralQuery{ScopeIDs: []string{"scope:missing"}}, MaxItems: 10, MaxBytes: 4096})
	assertLiveErrorCode(t, err, ErrorRevisionUnavailable)

	location := symbolPage.Items[0].Locations[0].Span
	contextEnvelope, err := session.GetBoundedSourceContext(context.Background(), SourceContextRequest{Consistency: ConsistencySpecific, Revision: 1, ScopeID: "scope:test", EntityID: symbolPage.Items[0].ID, Span: &location, MaxLines: 3, MaxBytes: 256})
	if err != nil {
		t.Fatalf("GetBoundedSourceContext() error = %v", err)
	}
	contextResult, ok := contextEnvelope.Result.(SourceContextResult)
	if !ok || contextResult.Context.Path != "main.go" || contextResult.Context.StartLine < 1 || contextResult.Context.EndLine-contextResult.Context.StartLine+1 > 3 {
		t.Fatalf("source context = %#v", contextEnvelope.Result)
	}
	truncatedEnvelope, err := session.GetBoundedSourceContext(context.Background(), SourceContextRequest{Consistency: ConsistencySpecific, Revision: 1, ScopeID: "scope:test", EntityID: symbolPage.Items[0].ID, Span: &location, MaxLines: 3, MaxBytes: 8})
	if err != nil {
		t.Fatalf("byte-bounded source context error = %v", err)
	}
	truncatedContext := truncatedEnvelope.Result.(SourceContextResult).Context
	if !truncatedContext.Truncated || len([]byte(truncatedContext.Content)) > 8 {
		t.Fatalf("byte-bounded source context = %#v", truncatedContext)
	}

	withoutSpanHash := location
	withoutSpanHash.ContentHash = analysis.ContentDigest{}
	contextEnvelope, err = session.GetBoundedSourceContext(context.Background(), SourceContextRequest{Consistency: ConsistencySpecific, Revision: 1, ScopeID: "scope:test", Span: &withoutSpanHash, MaxLines: 3, MaxBytes: 256})
	if err != nil {
		t.Fatalf("source context did not fall back to the indexed file hash: %v", err)
	}
	unknownSpan := location
	unknownSpan.Start.Line = 1
	unknownSpan.End.Line = 1
	_, err = session.GetBoundedSourceContext(context.Background(), SourceContextRequest{Consistency: ConsistencySpecific, Revision: 1, ScopeID: "scope:test", Span: &unknownSpan, MaxLines: 3, MaxBytes: 256})
	assertLiveErrorCode(t, err, ErrorSourceContextOutOfScope)

	moduleFacts, err := session.GetModuleFacts(context.Background(), QueryRequest{Consistency: ConsistencySpecific, Revision: 1, Query: StructuralQuery{ScopeIDs: []string{"scope:test"}}, MaxItems: 1, MaxBytes: 64 * 1024}, "example.com/main")
	if err != nil {
		t.Fatalf("GetModuleFacts() error = %v", err)
	}
	modulePage, ok := moduleFacts.Result.(ModuleFactsPage)
	if !ok || len(modulePage.Items) != 1 || modulePage.Items[0].ModuleID != "example.com/main" {
		t.Fatalf("module facts result = %#v", moduleFacts.Result)
	}
	_, err = session.GetModuleFacts(context.Background(), QueryRequest{Consistency: ConsistencySpecific, Revision: 1, MaxItems: 1, MaxBytes: 1}, "example.com/main")
	assertLiveErrorCode(t, err, ErrorQueryBudget)

	if _, err := session.FindFiles(context.Background(), QueryRequest{Consistency: ConsistencySpecific, Revision: 1, Query: StructuralQuery{ModuleIDs: []string{"module:not-present"}}, MaxItems: 10, MaxBytes: 4096}); err != nil {
		t.Fatalf("explicit containment query should return an empty result, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(strings.Replace(content, "return 42", "return 7", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = session.SearchExactText(context.Background(), TextSearchQuery{Consistency: ConsistencySpecific, Revision: 1, Pattern: "return", Mode: "literal", ScopeIDs: []string{"scope:test"}, MaxItems: 10, MaxBytes: 4096})
	assertLiveErrorCode(t, err, "SourceContextContentChanged")
}

func TestMixedCapabilityCoverageIsPartial(t *testing.T) {
	if got := mergeCoverageStatus("observed", "unsupported"); got != "partial" {
		t.Fatalf("mixed coverage = %q, want partial", got)
	}
}

func TestQualityGatewayCatalogTemporaryEvaluationAndComparison(t *testing.T) {
	root := t.TempDir()
	catalog := quality.NewDefaultCatalog()
	profile := quality.QualityProfile{SchemaVersion: quality.SchemaVersion, ProfileID: "profile:test", ProfileVersion: "1.0.0", EnabledRules: []quality.RuleBinding{}, SeverityPolicy: quality.TypedConfigBlock{}, Constraints: []quality.ArchitectureConstraint{}, Extensions: []quality.ExtensionBlock{}}
	profiles := NewMemoryQualityProfileResolver(profile)
	session, err := StartLiveSession(context.Background(), testLiveConfig("session:quality"), root, SessionOptions{Scanner: StaticScanner{Result: ScanResult{Model: testModel()}}, Fingerprinter: StaticFingerprinter{Value: testInput("quality")}, QualityCatalog: catalog, Profiles: profiles})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	gateway := session.QualityGateway()
	catalogEnvelope, err := gateway.ReadQualityCatalog(context.Background(), QualityCatalogRequest{QueryRequest: QueryRequest{Consistency: ConsistencySpecific, Revision: 1, MaxItems: 100, MaxBytes: 64 * 1024}, ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion})
	if err != nil {
		t.Fatalf("ReadQualityCatalog() error = %v", err)
	}
	catalogResult, ok := catalogEnvelope.Result.(QualityCatalogResult)
	if !ok || len(catalogResult.Rules) == 0 || len(catalogResult.Profiles) != 1 {
		t.Fatalf("catalog result = %#v", catalogEnvelope.Result)
	}
	if catalogResult.Profiles[0].Status != "valid" {
		t.Fatalf("profile validation status = %#v", catalogResult.Profiles[0])
	}

	evaluation, err := gateway.EvaluateQuality(context.Background(), QualityEvaluationRequest{SessionID: "session:quality", Consistency: ConsistencySpecific, Revision: 1, ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion})
	if err != nil {
		t.Fatalf("EvaluateQuality() error = %v", err)
	}
	evaluationResult, ok := evaluation.Result.(QualityEvaluationResult)
	if !ok || evaluationResult.Report == nil || !evaluationResult.Temporary {
		t.Fatalf("evaluation result = %#v", evaluation.Result)
	}
	findings, err := gateway.GetQualityFindings(context.Background(), QualityFindingsRequest{QueryRequest: QueryRequest{Consistency: ConsistencySpecific, Revision: 1, MaxItems: 10, MaxBytes: 64 * 1024}, ReportID: evaluationResult.Report.EvaluationID})
	if err != nil {
		t.Fatalf("GetQualityFindings() error = %v", err)
	}
	findingsResult, ok := findings.Result.(QualityFindingsResult)
	if !ok || findingsResult.Status != "observed" {
		t.Fatalf("findings result = %#v", findings.Result)
	}

	secondEvaluation, err := gateway.EvaluateQuality(context.Background(), QualityEvaluationRequest{Consistency: ConsistencySpecific, Revision: 1, ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion})
	if err != nil {
		t.Fatalf("second EvaluateQuality() error = %v", err)
	}
	secondResult := secondEvaluation.Result.(QualityEvaluationResult)
	comparison, err := gateway.CompareQualityReports(context.Background(), QualityCompareRequest{PreviousRevision: 1, CurrentRevision: 1, PreviousReportID: evaluationResult.Report.EvaluationID, CurrentReportID: secondResult.Report.EvaluationID})
	if err != nil {
		t.Fatalf("CompareQualityReports() error = %v", err)
	}
	comparisonResult, ok := comparison.Result.(QualityComparisonResult)
	if !ok || comparisonResult.Comparison == nil {
		t.Fatalf("comparison result = %#v", comparison.Result)
	}
	if _, err := gateway.EvaluateQuality(context.Background(), QualityEvaluationRequest{Consistency: ConsistencySpecific, Revision: 1, ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion, Persist: true}); err == nil {
		t.Fatal("temporary evaluation accepted persist=true")
	}
	_, err = gateway.CompareQualityReports(context.Background(), QualityCompareRequest{PreviousRevision: 1, CurrentRevision: 999})
	assertLiveErrorCode(t, err, ErrorRevisionUnavailable)
	current, ok := session.store.ReadLatestReady("session:quality")
	if !ok {
		t.Fatal("quality session has no ready record")
	}
	next := cloneRevisionRecord(current)
	next.Snapshot.Revision = 0
	next.Input = testInput("quality-next")
	next.Snapshot.SourceInputFingerprint = next.Input.ContentFingerprint
	next.Snapshot.InputVerification.ManifestFingerprint = next.Input.ManifestFingerprint
	next.Snapshot.InputVerification.ContentFingerprint = &next.Input.ContentFingerprint
	next.Snapshot.SemanticDigest = revisionSemanticDigest(next.Model, next.Run.Status, next.Snapshot.ScopeIDs, next.Model.QualityReport, next.Input.ContentFingerprint)
	next.Snapshot.SnapshotID = digestID("live-snapshot", next.SessionID, next.Input.ContentFingerprint.Value, next.Snapshot.SemanticDigest.Value)
	if _, err := session.publish(next); err != nil {
		t.Fatalf("publish next quality revision: %v", err)
	}
	firstRevision, ok := session.store.ReadRevision("session:quality", 1)
	if !ok {
		t.Fatal("first quality revision was discarded")
	}
	if _, err := gateway.reportFor(firstRevision, evaluationResult.Report.EvaluationID); err != nil {
		t.Fatalf("temporary report was discarded by a later publication: %v", err)
	}
}

type blockingScanner struct {
	result  ScanResult
	release <-chan struct{}
}

func (scanner *blockingScanner) Scan(ctx context.Context, _ ScanRequest) (ScanResult, error) {
	select {
	case <-scanner.release:
		return scanner.result, nil
	case <-ctx.Done():
		return ScanResult{}, ctx.Err()
	}
}

type countingScanner struct {
	result ScanResult
	count  atomic.Int32
	calls  chan int
}

func (scanner *countingScanner) Scan(context.Context, ScanRequest) (ScanResult, error) {
	count := int(scanner.count.Add(1))
	if scanner.calls != nil {
		scanner.calls <- count
	}
	return scanner.result, nil
}

func (scanner *countingScanner) Count() int { return int(scanner.count.Load()) }

type mutableFingerprinter struct {
	mu    sync.RWMutex
	value InputFingerprint
}

func (fingerprinter *mutableFingerprinter) Fingerprint(context.Context, string, []WatchRoot) (InputFingerprint, error) {
	fingerprinter.mu.RLock()
	defer fingerprinter.mu.RUnlock()
	return cloneInputFingerprint(fingerprinter.value), nil
}

func (fingerprinter *mutableFingerprinter) Set(value InputFingerprint) {
	fingerprinter.mu.Lock()
	fingerprinter.value = cloneInputFingerprint(value)
	fingerprinter.mu.Unlock()
}

type mutatingWatchBackend struct {
	fingerprinter *mutableFingerprinter
	value         InputFingerprint
	events        chan WatchEvent
}

func (backend *mutatingWatchBackend) Start(context.Context, string, []WatchRoot, WatchPolicy) (<-chan WatchEvent, error) {
	backend.fingerprinter.Set(backend.value)
	backend.events = make(chan WatchEvent)
	return backend.events, nil
}

func (backend *mutatingWatchBackend) Stop() error { return nil }

func testLiveConfig(sessionID string) LiveSessionConfig {
	return LiveSessionConfig{SchemaVersion: LiveSchemaVersion, SessionID: sessionID, RepositoryRoot: ".", WatchRoots: []WatchRoot{{Path: ".", Recursive: true}}, SourceIndexRequest: SourceIndexRequest{Enabled: false}}
}

func writeLiveFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func testInput(value string) InputFingerprint {
	digest := digestJSON(value)
	return InputFingerprint{ManifestFingerprint: digestJSON("manifest:" + value), ContentFingerprint: digest, Files: []FileFingerprint{{Path: "main.go", Size: int64(len(value)), Hash: digest}}}
}

func testModel() model.Model {
	value, err := canonical.Normalize(analysis.AnalysisResult{Status: analysis.StatusComplete, Analyzer: analysis.AnalyzerInfo{ID: "analyzer:test", Version: "1.0.0", Language: "go", APIVersion: analysis.AnalyzerAPIVersion}, Project: analysis.ProjectInfo{RootLabel: "test", Boundary: "repository"}, Modules: []analysis.ModuleObservation{}, References: []analysis.Reference{}, SourceReferences: []analysis.SourceReference{}, Relationships: []analysis.RelationshipObservation{}, Diagnostics: []analysis.Diagnostic{}})
	if err != nil {
		panic(fmt.Sprintf("build test model: %v", err))
	}
	return value
}

func assertLiveErrorCode(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %q", expected)
	}
	var liveErr *QueryError
	if !errors.As(err, &liveErr) || liveErr.Code != expected {
		t.Fatalf("error = %v, expected code %q", err, expected)
	}
}
