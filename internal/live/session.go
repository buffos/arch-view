package live

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/quality"
)

type LiveSession struct {
	validated     ValidatedLiveSession
	options       SessionOptions
	scanner       Scanner
	fingerprinter Fingerprinter
	store         SnapshotStore
	watcher       WatchBackend
	coalescer     *EventCoalescer

	mu             sync.RWMutex
	state          SessionState
	building       bool
	buildDone      chan struct{}
	initialDone    chan struct{}
	closed         bool
	stale          bool
	inputUnstable  bool
	lastInput      InputFingerprint
	lastChanged    []string
	lastGroupID    string
	lastFailure    *QueryError
	lastReconcile  ReconciliationStatus
	pendingPlan    *InvalidationPlan
	pendingFailure error
	cancel         context.CancelFunc
	rootContext    context.Context
	watcherCancel  context.CancelFunc
	// qualityReports uses a revision+evaluation composite key so temporary
	// reports remain queryable after newer live revisions are published.
	qualityReports map[string]quality.QualityEvaluation
	// qualityReportRevisions keeps temporary evaluation reports bound to the
	// verified revision they were computed from. They are never promoted to a
	// live model revision or written to disk.
	qualityReportRevisions map[string]int
}

// StartLiveSession validates a policy and starts its initial scan in the
// background. Call Wait when a caller needs the first ready/degraded result;
// Status can be read while the scan is initializing.
func StartLiveSession(ctx context.Context, config LiveSessionConfig, openedRepositoryRoot string, options SessionOptions) (*LiveSession, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	validated, err := ValidateLiveSession(ctx, config, openedRepositoryRoot, ValidationDependencies{AnalyzerRegistry: options.AnalyzerRegistry, QualityCatalog: options.QualityCatalog, Profiles: options.Profiles})
	if err != nil {
		return nil, err
	}
	if options.Fingerprinter == nil {
		options.Fingerprinter = FileSystemFingerprinter{}
	}
	if options.SnapshotStore == nil {
		options.SnapshotStore = NewMemorySnapshotStore()
	}
	if options.Scanner == nil {
		if options.AnalyzerRegistry == nil {
			return nil, newLiveError(ErrorSnapshotBuildFailed, "a scanner or analyzer registry is required", nil)
		}
		configuredScanner := NewMultiAnalyzerScanner(analysis.NewHost(options.AnalyzerRegistry), options.QualityCatalog, options.Profiles)
		configuredScanner.AnalyzerOptionsByID = cloneAnalyzerOptions(options.AnalyzerOptionsByID)
		options.Scanner = configuredScanner
	}
	if options.Watcher == nil && options.StartWatcher {
		options.Watcher = NewLocalWatchBackend(options.Fingerprinter)
	}
	rootContext, cancel := context.WithCancel(ctx)
	session := &LiveSession{
		validated: validated, options: options, scanner: options.Scanner,
		fingerprinter: options.Fingerprinter, store: options.SnapshotStore,
		watcher: options.Watcher, coalescer: NewEventCoalescer(validated.Config.WatchPolicy),
		state: SessionInitializing, initialDone: make(chan struct{}), buildDone: make(chan struct{}),
		building: true, lastReconcile: ReconciliationNotRequested, cancel: cancel,
		rootContext: rootContext, qualityReports: make(map[string]quality.QualityEvaluation), qualityReportRevisions: make(map[string]int),
	}
	go session.runInitialBuild()
	return session, nil
}

// NewLiveSession is an alias with the same asynchronous lifecycle semantics as
// StartLiveSession.
func NewLiveSession(ctx context.Context, config LiveSessionConfig, openedRepositoryRoot string, options SessionOptions) (*LiveSession, error) {
	return StartLiveSession(ctx, config, openedRepositoryRoot, options)
}

func (session *LiveSession) Config() LiveSessionConfig {
	if session == nil {
		return LiveSessionConfig{}
	}
	return cloneLiveSessionConfig(session.validated.Config)
}

func (session *LiveSession) RepositoryRoot() string {
	if session == nil {
		return ""
	}
	return session.validated.RepositoryRoot
}

func (session *LiveSession) Wait(ctx context.Context) error {
	if session == nil {
		return newLiveError(ErrorLiveConfigInvalid, "live session is nil", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-session.initialDone:
	}
	session.mu.RLock()
	defer session.mu.RUnlock()
	if session.state == SessionFailed && session.lastFailure != nil {
		return session.lastFailure
	}
	return nil
}

type SessionStatus struct {
	SessionID   string           `json:"session_id"`
	State       SessionState     `json:"state"`
	Snapshot    *LiveSnapshot    `json:"snapshot,omitempty"`
	Freshness   Freshness        `json:"freshness"`
	Diagnostics []LiveDiagnostic `json:"diagnostics"`
}

func (session *LiveSession) Status() SessionStatus {
	if session == nil {
		return SessionStatus{State: SessionFailed, Freshness: Freshness{Status: FreshnessFailed, Reconciliation: ReconciliationFailed}}
	}
	session.mu.RLock()
	state := session.state
	building := session.building
	stale := session.stale
	unstable := session.inputUnstable
	lastFailure := session.lastFailure
	changed := append([]string(nil), session.lastChanged...)
	groupID := session.lastGroupID
	lastReconcile := session.lastReconcile
	session.mu.RUnlock()
	record, _ := session.store.ReadLatestReady(session.validated.Config.SessionID)
	freshness := session.freshnessFor(ConsistencyLatestReady, building, stale, unstable, changed, groupID, lastReconcile, record)
	if record == nil && state == SessionFailed {
		freshness.Status = FreshnessFailed
	}
	diagnostics := []LiveDiagnostic{}
	if lastFailure != nil {
		diagnostics = append(diagnostics, LiveDiagnostic{Code: lastFailure.Code, Message: lastFailure.Message, Severity: "error", Recoverable: record != nil, Details: lastFailure.Details})
	}
	var snapshot *LiveSnapshot
	if record != nil {
		value := record.Snapshot
		snapshot = &value
	}
	return SessionStatus{SessionID: session.validated.Config.SessionID, State: state, Snapshot: snapshot, Freshness: freshness, Diagnostics: diagnostics}
}

func (session *LiveSession) Close() error {
	if session == nil {
		return nil
	}
	session.mu.Lock()
	if session.closed {
		session.mu.Unlock()
		return nil
	}
	session.closed = true
	cancel := session.cancel
	watcherCancel := session.watcherCancel
	session.mu.Unlock()
	if watcherCancel != nil {
		watcherCancel()
	}
	if session.watcher != nil {
		_ = session.watcher.Stop()
	}
	if cancel != nil {
		cancel()
	}
	return nil
}

func (session *LiveSession) runInitialBuild() {
	defer close(session.initialDone)
	plan := InvalidationPlan{Mode: "full_rescan", Reason: "initial_snapshot"}
	record, err := session.buildStable(session.rootContext, plan)
	if err == nil {
		_, err = session.publish(record)
	}
	if err != nil {
		session.mu.Lock()
		session.building = false
		session.state = SessionFailed
		session.lastFailure = asQueryError(err, ErrorSnapshotBuildFailed, "initial live snapshot could not be built")
		session.lastReconcile = ReconciliationFailed
		close(session.buildDone)
		session.mu.Unlock()
		return
	}
	session.mu.Lock()
	session.building = false
	close(session.buildDone)
	session.mu.Unlock()
	session.startBackgroundInputs()
}

func (session *LiveSession) startBackgroundInputs() {
	if session == nil {
		return
	}
	if session.options.StartWatcher && session.watcher != nil {
		session.startWatcher()
	}
	if interval := time.Duration(session.validated.Config.FreshnessPolicy.ReconcileIntervalMS) * time.Millisecond; interval > 0 {
		session.startPeriodicReconciliation(interval)
	}
}

func (session *LiveSession) startWatcher() {
	watchCtx, cancel := context.WithCancel(session.rootContext)
	session.mu.Lock()
	session.watcherCancel = cancel
	session.mu.Unlock()
	events, err := session.watcher.Start(watchCtx, session.validated.RepositoryRoot, session.validated.Config.WatchRoots, session.validated.Config.WatchPolicy)
	if err != nil {
		session.triggerRebuild(InvalidationPlan{Mode: "full_rescan", Reason: "watcher_start_failed"}, err)
		return
	}
	go session.consumeWatcherEvents(watchCtx, events)
	// Starting a backend establishes its own baseline. Reconcile immediately
	// afterward so a change between the initial snapshot's final fingerprint
	// and watcher startup cannot disappear into that baseline.
	current, fingerprintErr := session.fingerprinter.Fingerprint(watchCtx, session.validated.RepositoryRoot, session.validated.Config.WatchRoots)
	if fingerprintErr != nil {
		session.triggerRebuild(InvalidationPlan{Mode: "full_rescan", Reason: "watcher_start_reconciliation_failed"}, fingerprintErr)
		return
	}
	previous := session.lastInputValue()
	if !inputEqual(previous, current) {
		session.triggerRebuild(InvalidationPlan{Mode: "full_rescan", AffectedPaths: changedFingerprintPaths(previous, current), Reason: "watcher_start_reconciliation_changed"}, nil)
	}
}

func (session *LiveSession) consumeWatcherEvents(ctx context.Context, events <-chan WatchEvent) {
	if events == nil {
		return
	}
	var timer *time.Timer
	var timerChannel <-chan time.Time
	stopTimer := func() {
		if timer != nil {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		}
		timerChannel = nil
	}
	defer stopTimer()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			normalized, err := NormalizeWatchEvent(event, session.validated.RepositoryRoot, session.validated.Config.WatchRoots)
			if err != nil {
				session.coalescer.Add(NormalizedEvent{BackendID: event.BackendID, Kind: "error", Sequence: event.Sequence, EventGroupID: eventIdentity(NormalizedEvent{BackendID: event.BackendID, Kind: "error", Sequence: event.Sequence})}, time.Now())
			} else {
				session.coalescer.Add(normalized, time.Now())
			}
			if session.validated.Config.WatchPolicy.DebounceMS <= 0 {
				if group := session.coalescer.Flush(); group != nil {
					session.handleEventGroup(*group)
				}
				continue
			}
			stopTimer()
			timer = time.NewTimer(time.Duration(session.validated.Config.WatchPolicy.DebounceMS) * time.Millisecond)
			timerChannel = timer.C
		case <-timerChannel:
			if group := session.coalescer.Flush(); group != nil {
				session.handleEventGroup(*group)
			}
			timerChannel = nil
		}
	}
}

func (session *LiveSession) startPeriodicReconciliation(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-session.rootContext.Done():
				return
			case <-ticker.C:
				current, err := session.fingerprinter.Fingerprint(session.rootContext, session.validated.RepositoryRoot, session.validated.Config.WatchRoots)
				if err != nil {
					session.triggerRebuild(InvalidationPlan{Mode: "full_rescan", Reason: "periodic_reconciliation_failed"}, err)
					continue
				}
				record, ok := session.store.ReadLatestReady(session.validated.Config.SessionID)
				if !ok || !inputEqual(session.lastInputValue(), current) || record.Snapshot.SourceInputFingerprint.Value != current.ContentFingerprint.Value {
					session.triggerRebuild(InvalidationPlan{Mode: "full_rescan", AffectedPaths: changedFingerprintPaths(session.lastInputValue(), current), Reason: "periodic_reconciliation_changed"}, nil)
				}
			}
		}
	}()
}

func (session *LiveSession) handleEventGroup(group EventGroup) {
	if session == nil {
		return
	}
	scopes := session.scopeIdentities()
	plan := PlanInvalidation(group, scopes)
	if len(plan.AffectedPaths) == 0 {
		plan.AffectedPaths = append([]string(nil), group.ChangedPaths...)
	}
	session.triggerRebuild(plan, nil)
}

func (session *LiveSession) triggerRebuild(plan InvalidationPlan, triggerErr error) {
	_ = session.beginRebuild(plan, triggerErr)
}

func (session *LiveSession) runRebuild(ctx context.Context, plan InvalidationPlan, triggerErr error, done chan struct{}) {
	var err error
	if triggerErr != nil {
		err = triggerErr
	} else {
		var record *RevisionRecord
		record, err = session.buildStable(ctx, plan)
		if err == nil {
			_, err = session.publish(record)
		}
	}
	session.mu.Lock()
	session.building = false
	if err != nil {
		session.lastFailure = asQueryError(err, ErrorSnapshotBuildFailed, "live snapshot rebuild failed")
		session.lastReconcile = ReconciliationFailed
		if _, ready := session.store.ReadLatestReady(session.validated.Config.SessionID); ready {
			session.state = SessionDegraded
		} else {
			session.state = SessionFailed
		}
		if strings.Contains(session.lastFailure.Code, ErrorInputUnstable) {
			session.inputUnstable = true
			session.lastReconcile = ReconciliationUnstable
		}
	} else {
		session.stale = false
		session.inputUnstable = false
		session.lastFailure = nil
		session.lastReconcile = ReconciliationPassed
	}
	close(done)
	pendingPlan := session.pendingPlan
	pendingFailure := session.pendingFailure
	session.pendingPlan = nil
	session.pendingFailure = nil
	if pendingPlan != nil && !session.closed {
		nextDone := make(chan struct{})
		session.building = true
		session.buildDone = nextDone
		session.stale = true
		session.mu.Unlock()
		go session.runRebuild(session.rootContext, *pendingPlan, pendingFailure, nextDone)
		return
	}
	session.mu.Unlock()
}

func (session *LiveSession) queueRebuildLocked(plan InvalidationPlan, triggerErr error) {
	if session.pendingPlan == nil {
		copy := plan
		copy.AffectedPaths = append([]string(nil), plan.AffectedPaths...)
		copy.AffectedScopeIDs = append([]string(nil), plan.AffectedScopeIDs...)
		session.pendingPlan = &copy
	} else {
		session.pendingPlan.Mode = "full_rescan"
		session.pendingPlan.Reason = "changes_arrived_during_build"
		session.pendingPlan.AffectedPaths = uniqueStrings(append(session.pendingPlan.AffectedPaths, plan.AffectedPaths...))
		session.pendingPlan.AffectedScopeIDs = nil
	}
	if triggerErr != nil {
		session.pendingFailure = triggerErr
	}
}

func (session *LiveSession) publish(record *RevisionRecord) (*RevisionRecord, error) {
	if record == nil {
		return nil, newLiveError(ErrorSnapshotValidation, "cannot publish a nil revision", nil)
	}
	if err := validateRevisionCandidate(record); err != nil {
		return nil, err
	}
	published, err := session.store.PublishAtomically(record)
	if err != nil {
		return nil, err
	}
	session.mu.Lock()
	session.lastInput = cloneInputFingerprint(recordInput(record))
	if session.qualityReports == nil {
		session.qualityReports = make(map[string]quality.QualityEvaluation)
	}
	if session.qualityReportRevisions == nil {
		session.qualityReportRevisions = make(map[string]int)
	}
	for reportID, report := range record.QualityReports {
		key := revisionReportKey(published.Snapshot.Revision, reportID)
		session.qualityReports[key] = cloneQualityReport(report)
		session.qualityReportRevisions[key] = published.Snapshot.Revision
	}
	if published.Snapshot.State == SessionDegraded {
		session.state = SessionDegraded
	} else {
		session.state = SessionReady
	}
	session.mu.Unlock()
	return published, nil
}

func (session *LiveSession) buildStable(ctx context.Context, plan InvalidationPlan) (*RevisionRecord, error) {
	policy := session.validated.Config.FreshnessPolicy
	tries := policy.MaxStabilityRetries
	if tries < 1 {
		tries = 1
	}
	for attempt := 1; attempt <= tries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		before, err := session.fingerprinter.Fingerprint(ctx, session.validated.RepositoryRoot, session.validated.Config.WatchRoots)
		if err != nil {
			return nil, err
		}
		request := ScanRequest{SessionID: session.validated.Config.SessionID, RepositoryRoot: session.validated.RepositoryRoot, Config: session.validated.Config, Invalidation: plan, InputFingerprint: before}
		scan, scanErr := session.scanner.Scan(ctx, request)
		if scanErr != nil {
			return nil, scanErr
		}
		after, err := session.fingerprinter.Fingerprint(ctx, session.validated.RepositoryRoot, session.validated.Config.WatchRoots)
		if err != nil {
			return nil, err
		}
		if !inputEqual(before, after) {
			if attempt == tries {
				return nil, newLiveError(ErrorInputUnstable, "source input changed while the live snapshot was being built", map[string]any{"attempts": attempt, "changed_paths": changedFingerprintPaths(before, after)})
			}
			if err := waitForSettle(ctx, time.Duration(policy.SettleMS)*time.Millisecond); err != nil {
				return nil, err
			}
			continue
		}
		record := buildRevisionRecord(session.validated.Config.SessionID, scan, after, attempt, session.validated.Config.SourceIndexRequest.Enabled)
		if err := validateRevisionCandidate(record); err != nil {
			return nil, err
		}
		return record, nil
	}
	return nil, newLiveError(ErrorInputUnstable, "source input could not be stabilized", nil)
}

func validateRevisionCandidate(record *RevisionRecord) error {
	if record == nil {
		return newLiveError(ErrorSnapshotValidation, "revision candidate is nil", nil)
	}
	if err := canonical.Validate(record.Model); err != nil {
		return newLiveError(ErrorSnapshotValidation, "canonical model candidate is invalid", map[string]any{"error": err.Error()})
	}
	if record.Model.SourceIndex != nil {
		if err := analysis.ValidateSourceIndex(*record.Model.SourceIndex); err != nil {
			return newLiveError(ErrorSnapshotValidation, "source-index candidate is invalid", map[string]any{"error": err.Error()})
		}
	}
	if record.Model.QualityReport != nil {
		if err := quality.ValidateQualityEvaluation(*record.Model.QualityReport); err != nil {
			return newLiveError(ErrorSnapshotValidation, "quality report candidate is invalid", map[string]any{"error": err.Error()})
		}
	}
	if record.Snapshot.SourceInputFingerprint.Value == "" || record.Snapshot.InputVerification.Status != "verified" {
		return newLiveError(ErrorSnapshotValidation, "revision candidate does not contain a verified source fingerprint", nil)
	}
	verifiedContent := record.Snapshot.InputVerification.ContentFingerprint
	if verifiedContent == nil || *verifiedContent != record.Snapshot.SourceInputFingerprint || record.Input.ContentFingerprint != record.Snapshot.SourceInputFingerprint || record.Input.ManifestFingerprint != record.Snapshot.InputVerification.ManifestFingerprint {
		return newLiveError(ErrorSnapshotValidation, "revision candidate input fingerprints do not identify one source state", nil)
	}
	if record.Snapshot.ModelRef == nil || record.Snapshot.ModelRef.Kind != "model" || record.Snapshot.ModelRef.ID != record.Model.ModelID {
		return newLiveError(ErrorSnapshotValidation, "revision candidate model reference is incoherent", nil)
	}
	expectedSourceRef := sourceIndexRef(record.Model.SourceIndex)
	if !equalOpaqueRef(record.Snapshot.SourceIndexRef, expectedSourceRef) {
		return newLiveError(ErrorSnapshotValidation, "revision candidate source-index reference is incoherent", nil)
	}
	expectedQualityRef := qualityReportRef(record.Model.QualityReport)
	if !equalOpaqueRef(record.Snapshot.QualityReportRef, expectedQualityRef) {
		return newLiveError(ErrorSnapshotValidation, "revision candidate quality-report reference is incoherent", nil)
	}
	if !equalOpaqueRef(record.Snapshot.QualityPolicyRef, qualityPolicyRef(record.Model.QualityReport)) {
		return newLiveError(ErrorSnapshotValidation, "revision candidate quality-policy reference is incoherent", nil)
	}
	if record.Model.QualityReport != nil {
		stored, ok := record.QualityReports[record.Model.QualityReport.EvaluationID]
		if !ok || stored.EvaluationID != record.Model.QualityReport.EvaluationID {
			return newLiveError(ErrorSnapshotValidation, "revision candidate quality report is not attached to the same record", nil)
		}
	}
	expectedScopes := revisionScopeIDs(record.Run, record.Model)
	actualScopes := uniqueStrings(record.Snapshot.ScopeIDs)
	if len(actualScopes) != len(record.Snapshot.ScopeIDs) || strings.Join(actualScopes, "\x00") != strings.Join(expectedScopes, "\x00") {
		return newLiveError(ErrorSnapshotValidation, "revision candidate scope references are incoherent", nil)
	}
	expectedSemanticDigest := revisionSemanticDigest(record.Model, record.Run.Status, expectedScopes, record.Model.QualityReport, record.Input.ContentFingerprint)
	if record.Snapshot.SemanticDigest != expectedSemanticDigest || record.Snapshot.SnapshotID != digestID("live-snapshot", record.SessionID, record.Input.ContentFingerprint.Value, expectedSemanticDigest.Value) {
		return newLiveError(ErrorSnapshotValidation, "revision candidate semantic identity is incoherent", nil)
	}
	return nil
}

func equalOpaqueRef(left, right *OpaqueRef) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func waitForSettle(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func buildRevisionRecord(sessionID string, scan ScanResult, input InputFingerprint, attempts int, sourceEnabled bool) *RevisionRecord {
	value := scan.Model
	if value.SchemaVersion == "" {
		if canonicalModel, ok := scan.Run.CombinedCanonicalModel(); ok {
			value = canonicalModel
		}
	}
	if !sourceEnabled {
		value.SourceIndex = nil
	}
	qualityReport := scan.QualityReport
	if qualityReport == nil {
		qualityReport = value.QualityReport
	}
	if qualityReport != nil {
		value.QualityReport = qualityReport
	}
	if value.ModelID == "" {
		value.ModelID = scan.Run.RunID
	}
	scopeModels := make(map[string]model.Model)
	scopeResults := make(map[string]analysis.AnalysisResult)
	for _, summary := range scan.Run.Scopes {
		selected, err := scan.Run.SelectAnalysisScope(summary.ScopeID)
		if err == nil && selected.Model.ModelID != "" {
			item := selected.Model
			if !sourceEnabled {
				item.SourceIndex = nil
			}
			scopeModels[summary.ScopeID] = item
		}
		if result, err := scan.Run.ScopeResult(summary.ScopeID); err == nil {
			if !sourceEnabled {
				result.SourceIndex = nil
			}
			scopeResults[summary.ScopeID] = result
		}
	}
	qualityReports := make(map[string]quality.QualityEvaluation)
	if qualityReport != nil {
		qualityReports[qualityReport.EvaluationID] = cloneQualityReport(*qualityReport)
	}
	scopeIDs := revisionScopeIDs(scan.Run, value)
	semanticDigest := revisionSemanticDigest(value, scan.Run.Status, scopeIDs, qualityReport, input.ContentFingerprint)
	snapshotID := digestID("live-snapshot", sessionID, input.ContentFingerprint.Value, semanticDigest.Value)
	state := SessionReady
	if scan.Run.Status != analysis.StatusComplete || value.Status == model.StatusPartial {
		state = SessionDegraded
	}
	content := input.ContentFingerprint
	manifest := input.ManifestFingerprint
	return &RevisionRecord{
		SessionID: sessionID,
		Snapshot: LiveSnapshot{
			SchemaVersion: LiveSchemaVersion, SnapshotID: snapshotID, State: state,
			SourceInputFingerprint: content,
			InputVerification:      InputVerification{Status: "verified", ManifestFingerprint: manifest, ContentFingerprint: &content, Attempts: attempts},
			SourceIndexRef:         sourceIndexRef(value.SourceIndex),
			ModelRef:               &OpaqueRef{Kind: "model", ID: value.ModelID},
			QualityReportRef:       qualityReportRef(qualityReport),
			QualityPolicyRef:       qualityPolicyRef(qualityReport),
			ScopeIDs:               scopeIDs, Diagnostics: append([]LiveDiagnostic(nil), scan.Diagnostics...),
			Freshness: Freshness{Status: FreshnessCurrent, Reconciliation: ReconciliationPassed}, SemanticDigest: semanticDigest, Extensions: []ExtensionBlock{},
		},
		Input: input, Run: scan.Run, Model: value, ScopeModels: scopeModels, ScopeResults: scopeResults, QualityReports: qualityReports,
	}
}

func revisionSemanticDigest(value model.Model, status analysis.AnalysisStatus, scopeIDs []string, qualityReport *quality.QualityEvaluation, input ContentDigest) ContentDigest {
	return digestJSON(struct {
		Model     model.Model
		RunStatus string
		Scopes    []string
		Quality   *quality.QualityEvaluation
		Input     ContentDigest
	}{Model: value, RunStatus: string(status), Scopes: append([]string(nil), scopeIDs...), Quality: qualityReport, Input: input})
}

func sourceIndexRef(value *analysis.SourceIndex) *OpaqueRef {
	if value == nil {
		return nil
	}
	digest := digestJSON(value)
	return &OpaqueRef{Kind: "source_index", ID: digestID("source-index", digest.Value)}
}

func qualityReportRef(value *quality.QualityEvaluation) *OpaqueRef {
	if value == nil || value.EvaluationID == "" {
		return nil
	}
	return &OpaqueRef{Kind: "quality_report", ID: value.EvaluationID}
}

func qualityPolicyRef(value *quality.QualityEvaluation) *OpaqueRef {
	if value == nil {
		return nil
	}
	profileDigest := ""
	if value.ProfileDigest != nil {
		profileDigest = value.ProfileDigest.Value
	}
	optionsDigest := ""
	if value.OptionsDigest != nil {
		optionsDigest = value.OptionsDigest.Value
	}
	return &OpaqueRef{Kind: "quality_policy", ID: digestID("quality-policy", value.ProfileID, value.ProfileVersion, profileDigest, optionsDigest)}
}

func recordInput(record *RevisionRecord) InputFingerprint {
	if record == nil {
		return InputFingerprint{}
	}
	if len(record.Input.Files) > 0 {
		return cloneInputFingerprint(record.Input)
	}
	return InputFingerprint{ContentFingerprint: record.Snapshot.SourceInputFingerprint, ManifestFingerprint: record.Snapshot.InputVerification.ManifestFingerprint}
}

func sortedScopeIDs(values []orchestration.ScopeSummary) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.ScopeID)
	}
	sort.Strings(result)
	return result
}

func revisionScopeIDs(run orchestration.AnalysisRun, value model.Model) []string {
	result := sortedScopeIDs(run.Scopes)
	if value.SourceIndex != nil {
		for _, snapshot := range value.SourceIndex.Snapshots {
			result = append(result, snapshot.ScopeContext.ScopeID)
		}
		if value.SourceIndex.Projection != nil {
			result = append(result, value.SourceIndex.Projection.ScopeContext.ScopeID)
		}
	}
	return uniqueStrings(result)
}

func digestJSON(value any) ContentDigest {
	data, err := json.Marshal(value)
	if err != nil {
		data = []byte(fmt.Sprintf("%v", value))
	}
	hash := sha256.Sum256(data)
	return ContentDigest{Algorithm: "hash:sha-256", Value: hex.EncodeToString(hash[:])}
}

func digestID(prefix string, values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	return prefix + "-" + hex.EncodeToString(hash.Sum(nil)[:12])
}

func cloneQualityReport(value quality.QualityEvaluation) quality.QualityEvaluation {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone quality.QualityEvaluation
	if err := json.Unmarshal(data, &clone); err != nil {
		return value
	}
	return clone
}

func (session *LiveSession) lastInputValue() InputFingerprint {
	if session == nil {
		return InputFingerprint{}
	}
	session.mu.RLock()
	defer session.mu.RUnlock()
	return cloneInputFingerprint(session.lastInput)
}

func (session *LiveSession) scopeIdentities() []ScopeIdentity {
	record, ok := session.store.ReadLatestReady(session.validated.Config.SessionID)
	if !ok {
		return nil
	}
	result := make([]ScopeIdentity, 0, len(record.Run.Scopes))
	for _, scope := range record.Run.Scopes {
		result = append(result, ScopeIdentity{ScopeID: scope.ScopeID, ProjectRoot: scope.ProjectRoot, AnalyzerID: scope.Analyzer.ID})
	}
	return result
}

func (session *LiveSession) freshnessFor(requested Consistency, building, stale, unstable bool, changed []string, groupID string, reconciliation ReconciliationStatus, record *RevisionRecord) Freshness {
	status := FreshnessCurrent
	if record == nil {
		status = FreshnessInitializing
	}
	if building {
		if record == nil {
			status = FreshnessInitializing
		} else {
			status = FreshnessUpdating
		}
	} else if unstable {
		status = FreshnessInputUnstable
	} else if stale {
		status = FreshnessStale
	}
	freshness := Freshness{Status: status, RequestedConsistency: requested, Reconciliation: reconciliation, ChangedPaths: append([]string(nil), changed...), PendingEventGroupID: groupID}
	if record != nil {
		freshness.LastReadyRevision = record.Snapshot.Revision
	}
	return freshness
}

func asQueryError(err error, fallbackCode, fallbackMessage string) *QueryError {
	if err == nil {
		return newLiveError(fallbackCode, fallbackMessage, nil)
	}
	if value, ok := err.(*QueryError); ok {
		return value
	}
	return newLiveError(fallbackCode, fallbackMessage+": "+err.Error(), nil)
}
