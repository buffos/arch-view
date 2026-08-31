package live

import "context"

// LocalQueryAdapter is the in-process read adapter used by future CLI,
// viewer, and MCP transports. It contains no transport or analyzer logic; it
// simply makes the shared session/gateway ports easy to inject.
type LocalQueryAdapter struct {
	Session *LiveSession
	Quality *QualityGateway
}

func NewLocalQueryAdapter(session *LiveSession) *LocalQueryAdapter {
	if session == nil {
		return &LocalQueryAdapter{}
	}
	return &LocalQueryAdapter{Session: session, Quality: session.QualityGateway()}
}

func (adapter *LocalQueryAdapter) GetSnapshotStatus(ctx context.Context, request QueryRequest) (QueryEnvelope, error) {
	return adapter.Session.GetSnapshotStatus(ctx, request)
}

func (adapter *LocalQueryAdapter) ListScopes(ctx context.Context, request QueryRequest) (QueryEnvelope, error) {
	return adapter.Session.ListScopes(ctx, request)
}

func (adapter *LocalQueryAdapter) FindFiles(ctx context.Context, request QueryRequest) (QueryEnvelope, error) {
	return adapter.Session.FindFiles(ctx, request)
}

func (adapter *LocalQueryAdapter) FindSymbols(ctx context.Context, request QueryRequest) (QueryEnvelope, error) {
	return adapter.Session.FindSymbols(ctx, request)
}

func (adapter *LocalQueryAdapter) FindText(ctx context.Context, request TextSearchQuery) (QueryEnvelope, error) {
	return adapter.Session.FindText(ctx, request)
}

func (adapter *LocalQueryAdapter) GetDocumentation(ctx context.Context, request QueryRequest) (QueryEnvelope, error) {
	return adapter.Session.GetDocumentation(ctx, request)
}

func (adapter *LocalQueryAdapter) GetModuleFacts(ctx context.Context, request QueryRequest, moduleID string) (QueryEnvelope, error) {
	return adapter.Session.GetModuleFacts(ctx, request, moduleID)
}

func (adapter *LocalQueryAdapter) GetCallersCallees(ctx context.Context, request QueryRequest, entityID string) (QueryEnvelope, error) {
	return adapter.Session.GetCallersCallees(ctx, request, entityID)
}

func (adapter *LocalQueryAdapter) GetSourceContext(ctx context.Context, request SourceContextRequest) (QueryEnvelope, error) {
	return adapter.Session.GetSourceContext(ctx, request)
}

func (adapter *LocalQueryAdapter) GetQualityCatalog(ctx context.Context, request QualityCatalogRequest) (QueryEnvelope, error) {
	return adapter.Quality.ReadQualityCatalog(ctx, request)
}

func (adapter *LocalQueryAdapter) EvaluateQuality(ctx context.Context, request QualityEvaluationRequest) (QueryEnvelope, error) {
	return adapter.Quality.EvaluateQuality(ctx, request)
}

func (adapter *LocalQueryAdapter) GetQualityFindings(ctx context.Context, request QualityFindingsRequest) (QueryEnvelope, error) {
	return adapter.Quality.GetQualityFindings(ctx, request)
}

func (adapter *LocalQueryAdapter) GetFindingEvidence(ctx context.Context, request QualityEvidenceRequest) (QueryEnvelope, error) {
	return adapter.Quality.GetFindingEvidence(ctx, request)
}

func (adapter *LocalQueryAdapter) CompareQualityReports(ctx context.Context, request QualityCompareRequest) (QueryEnvelope, error) {
	return adapter.Quality.CompareQualityReports(ctx, request)
}

// GetSnapshotStatus is the transport-neutral status operation. It deliberately
// returns a query envelope so status reads carry the same revision/freshness
// metadata as structural and quality reads.
func (session *LiveSession) GetSnapshotStatus(ctx context.Context, request QueryRequest) (QueryEnvelope, error) {
	if err := session.requireOperation(OperationStatus); err != nil {
		return QueryEnvelope{}, err
	}
	if request.Consistency == "" {
		request.Consistency = ConsistencyLatestReady
	}
	if request.SessionID != "" && request.SessionID != session.validated.Config.SessionID {
		return QueryEnvelope{}, newLiveError("QueryInvalid", "status session_id does not match the live session", nil)
	}
	status := session.Status()
	maxBytes, maxItems, budgetErr := session.normalizeBudget(request.MaxBytes, request.MaxItems)
	if budgetErr != nil {
		return QueryEnvelope{}, budgetErr
	}
	record, freshness, returned, err := session.prepareConsistency(ctx, request.Consistency, request.Revision)
	if err != nil {
		if liveErr, ok := err.(*QueryError); !ok || liveErr.Code != ErrorNoReadySnapshot {
			return QueryEnvelope{}, err
		}
		result := SnapshotStatusResult{State: status.State, Diagnostics: append([]LiveDiagnostic(nil), status.Diagnostics...)}
		emittedBytes := jsonSize(result)
		if emittedBytes > maxBytes {
			return QueryEnvelope{}, newLiveError(ErrorQueryBudget, "snapshot status exceeds the query byte budget", map[string]any{"max_bytes": maxBytes, "emitted_bytes": emittedBytes})
		}
		return QueryEnvelope{SchemaVersion: QuerySchemaVersion, SessionID: session.validated.Config.SessionID, RequestedConsistency: request.Consistency, ReturnedConsistency: "unavailable", Freshness: status.Freshness, ScopeIDs: []string{}, Result: result, ResultCount: 0, OmittedFields: []string{}, Capabilities: []CapabilityCoverage{}, Budget: BudgetUsage{MaxBytes: maxBytes, MaxItems: maxItems, EmittedBytes: emittedBytes, EmittedItems: 0}, Diagnostics: []QueryDiagnostic{{Code: ErrorNoReadySnapshot, Message: "no ready live snapshot is available"}}}, nil
	}
	result := SnapshotStatusResult{State: status.State, Snapshot: &record.Snapshot, Diagnostics: append([]LiveDiagnostic(nil), status.Diagnostics...)}
	emittedBytes := jsonSize(result)
	if emittedBytes > maxBytes {
		return QueryEnvelope{}, newLiveError(ErrorQueryBudget, "snapshot status exceeds the query byte budget", map[string]any{"max_bytes": maxBytes, "emitted_bytes": emittedBytes})
	}
	return QueryEnvelope{SchemaVersion: QuerySchemaVersion, SessionID: session.validated.Config.SessionID, SnapshotID: record.Snapshot.SnapshotID, Revision: record.Snapshot.Revision, RequestedConsistency: request.Consistency, ReturnedConsistency: returned, Freshness: freshness, ScopeIDs: append([]string(nil), record.Snapshot.ScopeIDs...), Result: result, ResultCount: 1, OmittedFields: []string{}, Capabilities: []CapabilityCoverage{}, Budget: BudgetUsage{MaxBytes: maxBytes, MaxItems: maxItems, EmittedBytes: emittedBytes, EmittedItems: 1}, Diagnostics: []QueryDiagnostic{}}, nil
}

// QueryLatestReady and QuerySpecificRevision are small selection ports used
// by local adapters that need the immutable record rather than a projection.
func (session *LiveSession) QueryLatestReady(_ context.Context) (*RevisionRecord, error) {
	return session.CurrentRecord(ConsistencyLatestReady, 0)
}

func (session *LiveSession) QuerySpecificRevision(_ context.Context, revision int) (*RevisionRecord, error) {
	return session.CurrentRecord(ConsistencySpecific, revision)
}

func (session *LiveSession) SearchExactText(ctx context.Context, request TextSearchQuery) (QueryEnvelope, error) {
	return session.FindText(ctx, request)
}

func (session *LiveSession) GetBoundedSourceContext(ctx context.Context, request SourceContextRequest) (QueryEnvelope, error) {
	return session.GetSourceContext(ctx, request)
}
