package live

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/quality"
)

// MemorySnapshotStore is the default session-local immutable revision store.
// Its interface is intentionally small so a persisted store can replace it
// without changing the coordinator or query services.
type MemorySnapshotStore struct {
	mu        sync.RWMutex
	revisions map[string]map[int]*RevisionRecord
	latest    map[string]*RevisionRecord
	changed   map[string]chan struct{}
}

func NewMemorySnapshotStore() *MemorySnapshotStore {
	return &MemorySnapshotStore{
		revisions: make(map[string]map[int]*RevisionRecord),
		latest:    make(map[string]*RevisionRecord),
		changed:   make(map[string]chan struct{}),
	}
}

func (store *MemorySnapshotStore) ReadLatestReady(sessionID string) (*RevisionRecord, bool) {
	if store == nil {
		return nil, false
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	record, ok := store.latest[sessionID]
	if !ok {
		return nil, false
	}
	return cloneRevisionRecord(record), true
}

func (store *MemorySnapshotStore) ReadRevision(sessionID string, revision int) (*RevisionRecord, bool) {
	if store == nil || revision < 1 {
		return nil, false
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	revisions := store.revisions[sessionID]
	record, ok := revisions[revision]
	if !ok {
		return nil, false
	}
	return cloneRevisionRecord(record), true
}

func (store *MemorySnapshotStore) PublishAtomically(candidate *RevisionRecord) (*RevisionRecord, error) {
	if store == nil || candidate == nil {
		return nil, newLiveError(ErrorSnapshotValidation, "snapshot store cannot publish a nil candidate", nil)
	}
	if candidate.SessionID == "" || candidate.Snapshot.SnapshotID == "" || candidate.Snapshot.SemanticDigest.Value == "" {
		return nil, newLiveError(ErrorSnapshotValidation, "candidate snapshot is missing immutable identity", nil)
	}
	if _, err := json.Marshal(candidate); err != nil {
		return nil, newLiveError(ErrorSnapshotValidation, "candidate snapshot is not serializable", map[string]any{"error": err.Error()})
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.revisions[candidate.SessionID] == nil {
		store.revisions[candidate.SessionID] = make(map[int]*RevisionRecord)
	}
	current := store.latest[candidate.SessionID]
	if current != nil && current.Snapshot.SemanticDigest == candidate.Snapshot.SemanticDigest {
		return cloneRevisionRecord(current), nil
	}
	nextRevision := 1
	if current != nil {
		nextRevision = current.Snapshot.Revision + 1
	}
	if candidate.Snapshot.Revision != 0 && candidate.Snapshot.Revision <= nextRevision-1 {
		return nil, newLiveError("SnapshotPublishConflict", "candidate revision is not monotonic", map[string]any{"current": nextRevision - 1, "candidate": candidate.Snapshot.Revision})
	}
	copy := cloneRevisionRecord(candidate)
	copy.Snapshot.Revision = nextRevision
	copy.Snapshot.InputVerification.Status = "verified"
	copy.Snapshot.Freshness.LastReadyRevision = nextRevision
	store.revisions[candidate.SessionID][nextRevision] = copy
	store.latest[candidate.SessionID] = copy
	if store.changed[candidate.SessionID] == nil {
		store.changed[candidate.SessionID] = make(chan struct{})
	} else {
		close(store.changed[candidate.SessionID])
		store.changed[candidate.SessionID] = make(chan struct{})
	}
	return cloneRevisionRecord(copy), nil
}

func (store *MemorySnapshotStore) WaitForRevision(ctx context.Context, sessionID string, minimumRevision int) (*RevisionRecord, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		store.mu.Lock()
		if current := store.latest[sessionID]; current != nil && current.Snapshot.Revision >= minimumRevision {
			value := cloneRevisionRecord(current)
			store.mu.Unlock()
			return value, nil
		}
		wait := store.changed[sessionID]
		if wait == nil {
			wait = make(chan struct{})
			store.changed[sessionID] = wait
		}
		store.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-wait:
		}
	}
}

func cloneRevisionRecord(value *RevisionRecord) *RevisionRecord {
	if value == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone RevisionRecord
	if err := json.Unmarshal(data, &clone); err != nil {
		return value
	}
	clone.SessionID = value.SessionID
	clone.Run = cloneAnalysisRun(value.Run)
	clone.ScopeModels = make(map[string]model.Model, len(value.ScopeModels))
	for key, item := range value.ScopeModels {
		clone.ScopeModels[key] = cloneModel(item)
	}
	clone.ScopeResults = make(map[string]analysis.AnalysisResult, len(value.ScopeResults))
	for key, item := range value.ScopeResults {
		clone.ScopeResults[key] = cloneAnalysisResult(item)
	}
	clone.QualityReports = make(map[string]quality.QualityEvaluation, len(value.QualityReports))
	for key, item := range value.QualityReports {
		clone.QualityReports[key] = cloneQualityReport(item)
	}
	return &clone
}

func cloneAnalysisRun(value orchestration.AnalysisRun) orchestration.AnalysisRun {
	return orchestration.CloneAnalysisRun(value)
}

func cloneModel(value model.Model) model.Model {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone model.Model
	if err := json.Unmarshal(data, &clone); err != nil {
		return value
	}
	return clone
}

func cloneAnalysisResult(value analysis.AnalysisResult) analysis.AnalysisResult {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone analysis.AnalysisResult
	if err := json.Unmarshal(data, &clone); err != nil {
		return value
	}
	return clone
}
