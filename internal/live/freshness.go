package live

import (
	"context"
	"time"
)

// EnsureCurrentSnapshot reconciles the configured roots before returning a
// revision. Watcher state is never treated as proof that the source is clean.
func (session *LiveSession) EnsureCurrentSnapshot(ctx context.Context) (*RevisionRecord, error) {
	if session == nil {
		return nil, newLiveError(ErrorLiveConfigInvalid, "live session is nil", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	maxWait := time.Duration(session.validated.Config.FreshnessPolicy.MaxWaitMS) * time.Millisecond
	if maxWait > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, maxWait)
		defer cancel()
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, newLiveError(ErrorAnalysisWaitTimeout, "waiting for a current live revision timed out", map[string]any{"error": err.Error()})
		}
		record, hasRecord := session.store.ReadLatestReady(session.validated.Config.SessionID)
		current, err := session.fingerprinter.Fingerprint(ctx, session.validated.RepositoryRoot, session.validated.Config.WatchRoots)
		if err != nil {
			session.markReconciliation(ReconciliationFailed, nil, err)
			if hasRecord {
				return nil, asQueryError(err, ErrorSourceReconciliation, "authoritative source reconciliation failed")
			}
			return nil, asQueryError(err, ErrorSourceReconciliation, "authoritative source reconciliation failed")
		}
		if hasRecord && inputEqual(record.Input, current) {
			session.markReconciliation(ReconciliationPassed, nil, nil)
			return record, nil
		}
		changed := []string{}
		if hasRecord {
			changed = changedFingerprintPaths(record.Input, current)
		}
		plan := InvalidationPlan{Mode: "full_rescan", AffectedPaths: changed, Reason: "authoritative_reconciliation_changed"}
		if done := session.beginRebuild(plan, nil); done != nil {
			select {
			case <-ctx.Done():
				return nil, newLiveError(ErrorAnalysisWaitTimeout, "waiting for the current live revision timed out", map[string]any{"error": ctx.Err().Error()})
			case <-done:
			}
		}
		after, afterOK := session.store.ReadLatestReady(session.validated.Config.SessionID)
		if afterOK {
			verified, verifyErr := session.fingerprinter.Fingerprint(ctx, session.validated.RepositoryRoot, session.validated.Config.WatchRoots)
			if verifyErr == nil && inputEqual(after.Input, verified) {
				session.markReconciliation(ReconciliationPassed, nil, nil)
				return after, nil
			}
			if verifyErr != nil {
				session.markReconciliation(ReconciliationFailed, nil, verifyErr)
			} else {
				session.markReconciliation(ReconciliationUnstable, changedFingerprintPaths(after.Input, verified), nil)
			}
		}
		session.mu.RLock()
		unstable := session.inputUnstable
		failure := session.lastFailure
		session.mu.RUnlock()
		if unstable {
			return nil, newLiveError(ErrorInputUnstable, "source input did not settle within the live stability policy", nil)
		}
		if failure != nil {
			return nil, failure
		}
		if !hasRecord {
			return nil, newLiveError(ErrorNoReadySnapshot, "no ready live snapshot is available", nil)
		}
		return nil, newLiveError(ErrorAnalysisWaitTimeout, "a current live revision was not published within the wait budget", nil)
	}
}

func (session *LiveSession) beginRebuild(plan InvalidationPlan, triggerErr error) chan struct{} {
	if session == nil {
		return nil
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return nil
	}
	session.stale = true
	session.lastChanged = append([]string(nil), plan.AffectedPaths...)
	session.lastGroupID = plan.EventGroupID
	if session.building {
		return session.buildDone
	}
	session.building = true
	session.inputUnstable = false
	session.lastFailure = nil
	session.lastReconcile = ReconciliationChanged
	done := make(chan struct{})
	session.buildDone = done
	go session.runRebuild(session.rootContext, plan, triggerErr, done)
	return done
}

func (session *LiveSession) markReconciliation(status ReconciliationStatus, changed []string, err error) {
	if session == nil {
		return
	}
	_, hasReady := session.store.ReadLatestReady(session.validated.Config.SessionID)
	session.mu.Lock()
	session.lastReconcile = status
	if changed != nil {
		session.lastChanged = append([]string(nil), changed...)
	}
	if err != nil {
		session.lastFailure = asQueryError(err, ErrorSourceReconciliation, "source reconciliation failed")
	}
	switch status {
	case ReconciliationPassed:
		session.stale = false
		session.inputUnstable = false
	case ReconciliationChanged, ReconciliationFailed, ReconciliationUnstable:
		// A failed or incomplete authoritative check must never leave a prior
		// revision looking current. Readers may still use it with
		// latest_ready, but its freshness is explicitly degraded.
		session.stale = true
	}
	if status == ReconciliationUnstable {
		session.inputUnstable = true
	}
	if hasReady && status != ReconciliationPassed && session.state == SessionReady {
		session.state = SessionDegraded
	}
	session.mu.Unlock()
}

func (session *LiveSession) CurrentRecord(consistency Consistency, revision int) (*RevisionRecord, error) {
	if session == nil {
		return nil, newLiveError(ErrorLiveConfigInvalid, "live session is nil", nil)
	}
	if consistency == "" {
		consistency = session.validated.Config.FreshnessPolicy.DefaultConsistency
	}
	switch consistency {
	case ConsistencyRequireCurrent:
		return session.EnsureCurrentSnapshot(context.Background())
	case ConsistencySpecific:
		if revision < 1 {
			return nil, newLiveError(ErrorRevisionUnavailable, "specific_revision requires a positive revision", nil)
		}
		record, ok := session.store.ReadRevision(session.validated.Config.SessionID, revision)
		if !ok {
			return nil, newLiveError(ErrorRevisionUnavailable, "requested live revision is unavailable", map[string]any{"revision": revision})
		}
		return record, nil
	case ConsistencyLatestReady:
		record, ok := session.store.ReadLatestReady(session.validated.Config.SessionID)
		if !ok {
			return nil, newLiveError(ErrorNoReadySnapshot, "no ready live snapshot is available", nil)
		}
		return record, nil
	default:
		return nil, newLiveError(ErrorLiveConfigInvalid, "consistency selector is unsupported", map[string]any{"consistency": consistency})
	}
}
