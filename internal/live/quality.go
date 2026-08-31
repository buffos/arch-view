package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/quality"
	qualityadapter "github.com/buffo/arch-view/internal/quality/adapter"
)

// QualityGateway is the live boundary around the deterministic-quality
// services. It owns revision selection and response envelopes; it does not
// implement rule matching, metric calculation, suppression, or comparison.
type QualityGateway struct {
	session  *LiveSession
	catalog  *quality.Catalog
	profiles QualityProfileResolver
}

func NewQualityGateway(session *LiveSession, catalog *quality.Catalog, profiles QualityProfileResolver) *QualityGateway {
	if session != nil {
		if catalog == nil {
			catalog = session.options.QualityCatalog
		}
		if profiles == nil {
			profiles = session.options.Profiles
		}
	}
	return &QualityGateway{session: session, catalog: catalog, profiles: profiles}
}

// QualityGateway returns the shared gateway configured for this session.
func (session *LiveSession) QualityGateway() *QualityGateway {
	if session == nil {
		return nil
	}
	return NewQualityGateway(session, session.options.QualityCatalog, session.options.Profiles)
}

// MemoryQualityProfileResolver is a deterministic profile port for tests,
// embedders, and transports that already loaded profile JSON. File-backed
// loading remains a host concern and can implement QualityProfileResolver
// without changing the live boundary.
type MemoryQualityProfileResolver struct {
	Profiles map[string]quality.QualityProfile
}

func NewMemoryQualityProfileResolver(profiles ...quality.QualityProfile) *MemoryQualityProfileResolver {
	result := &MemoryQualityProfileResolver{Profiles: make(map[string]quality.QualityProfile, len(profiles))}
	for _, profile := range profiles {
		result.Profiles[profileKey(profile.ProfileID, profile.ProfileVersion)] = cloneQualityProfile(profile)
	}
	return result
}

func (resolver *MemoryQualityProfileResolver) ResolveProfile(_ context.Context, id, version string) (quality.QualityProfile, error) {
	if resolver == nil {
		return quality.QualityProfile{}, fmt.Errorf("quality profile resolver is unavailable")
	}
	if value, ok := resolver.Profiles[profileKey(id, version)]; ok {
		return cloneQualityProfile(value), nil
	}
	return quality.QualityProfile{}, fmt.Errorf("quality profile %s@%s was not found", id, version)
}

func (resolver *MemoryQualityProfileResolver) ListProfiles(_ context.Context) ([]QualityProfileInfo, error) {
	if resolver == nil {
		return nil, fmt.Errorf("quality profile resolver is unavailable")
	}
	result := make([]QualityProfileInfo, 0, len(resolver.Profiles))
	for _, profile := range resolver.Profiles {
		result = append(result, QualityProfileInfo{ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ProfileID == result[j].ProfileID {
			return result[i].ProfileVersion < result[j].ProfileVersion
		}
		return result[i].ProfileID < result[j].ProfileID
	})
	return result, nil
}

func (gateway *QualityGateway) ReadQualityCatalog(ctx context.Context, request QualityCatalogRequest) (QueryEnvelope, error) {
	if err := gateway.require(OperationQualityProfileRead); err != nil {
		return QueryEnvelope{}, err
	}
	if gateway.catalog == nil {
		return QueryEnvelope{}, newLiveError(ErrorQualityEvaluation, "quality rule catalog is unavailable", nil)
	}
	record, freshness, returned, query, err := gateway.session.prepareQuery(ctx, request.QueryRequest)
	if err != nil {
		return QueryEnvelope{}, err
	}
	profiles := []QualityProfileInfo{}
	if gateway.profiles != nil {
		profiles, err = gateway.profiles.ListProfiles(ctx)
		if err != nil {
			return QueryEnvelope{}, newLiveError(ErrorQualityEvaluation, "quality profiles could not be listed", map[string]any{"error": err.Error()})
		}
		for index := range profiles {
			profile, resolveErr := gateway.profiles.ResolveProfile(ctx, profiles[index].ProfileID, profiles[index].ProfileVersion)
			if resolveErr != nil {
				profiles[index].Status = "invalid"
				profiles[index].Reason = resolveErr.Error()
				continue
			}
			if _, validateErr := quality.ValidateQualityProfile(profile, gateway.catalog); validateErr != nil {
				profiles[index].Status = "invalid"
				profiles[index].Reason = validateErr.Error()
				continue
			}
			profiles[index].Status = "valid"
			profiles[index].Reason = ""
		}
	}
	var selected *quality.QualityProfile
	if request.ProfileID != "" || request.ProfileVersion != "" {
		if gateway.profiles == nil || request.ProfileID == "" || request.ProfileVersion == "" {
			return QueryEnvelope{}, newLiveError(ErrorQualityEvaluation, "a catalog profile selector requires profile_id and profile_version", nil)
		}
		profile, resolveErr := gateway.profiles.ResolveProfile(ctx, request.ProfileID, request.ProfileVersion)
		if resolveErr != nil {
			return QueryEnvelope{}, newLiveError(ErrorQualityEvaluation, "selected quality profile could not be resolved", map[string]any{"error": resolveErr.Error()})
		}
		validated, validateErr := quality.ValidateQualityProfile(profile, gateway.catalog)
		if validateErr != nil {
			return QueryEnvelope{}, newLiveError(ErrorQualityEvaluation, "selected quality profile is invalid", map[string]any{"error": validateErr.Error()})
		}
		selected = &validated
	}
	rules := make([]QualityRuleCatalogEntry, 0)
	bindings := map[string]quality.RuleBinding{}
	if selected != nil {
		for _, binding := range selected.EnabledRules {
			bindings[ruleKey(binding.RuleID, binding.RuleVersion)] = cloneRuleBinding(binding)
		}
	}
	for _, descriptor := range gateway.catalog.ListQualityRules() {
		if len(query.Query.RuleIDs) > 0 && !containsString(query.Query.RuleIDs, descriptor.ID) {
			continue
		}
		binding, ok := bindings[ruleKey(descriptor.ID, descriptor.Version)]
		if !ok {
			binding, ok = gateway.catalog.DefaultRuleBinding(descriptor.ID, descriptor.Version)
		}
		status := "catalog_available"
		reason := ""
		if selected == nil {
			status = "catalog_only"
			reason = "no profile was selected; the entry shows catalog defaults"
		}
		entry := QualityRuleCatalogEntry{RuleID: descriptor.ID, RuleVersion: descriptor.Version, AssessmentKind: descriptor.AssessmentKind, RequiredCapabilities: append([]string(nil), descriptor.RequiredCapabilities...), ParameterSchema: descriptor.ParameterSchema, DefaultSeverity: descriptor.DefaultSeverity, Description: descriptor.Description, Limitations: append([]string(nil), descriptor.Limitations...), Status: status, Reason: reason}
		if ok {
			entry.Enabled = binding.Enabled
			entry.Parameters = cloneTypedConfigBlock(binding.Parameters)
			entry.Severity = binding.Severity
		}
		rules = append(rules, entry)
	}
	request.QueryRequest = query
	return gateway.catalogEnvelope(record, freshness, returned, request, profiles, rules)
}

// GetQualityRules and GetQualityProfiles are transport-friendly aliases. The
// combined catalog keeps one revision and one pagination boundary for both.
func (gateway *QualityGateway) GetQualityRules(ctx context.Context, request QualityCatalogRequest) (QueryEnvelope, error) {
	return gateway.ReadQualityCatalog(ctx, request)
}

func (gateway *QualityGateway) GetQualityProfiles(ctx context.Context, request QualityCatalogRequest) (QueryEnvelope, error) {
	return gateway.ReadQualityCatalog(ctx, request)
}

func (gateway *QualityGateway) EvaluateQuality(ctx context.Context, request QualityEvaluationRequest) (QueryEnvelope, error) {
	if err := gateway.require(OperationQualityEvaluate); err != nil {
		return QueryEnvelope{}, err
	}
	if request.SessionID != "" && request.SessionID != gateway.session.validated.Config.SessionID {
		return QueryEnvelope{}, newLiveError("QueryInvalid", "quality evaluation session_id does not match the live session", nil)
	}
	if request.Persist {
		return QueryEnvelope{}, newLiveError("quality_policy_permission_denied", "temporary quality evaluation cannot persist profile or rule changes", nil)
	}
	if gateway.catalog == nil || gateway.profiles == nil {
		return QueryEnvelope{}, newLiveError(ErrorQualityEvaluation, "quality evaluation requires a catalog and profile resolver", nil)
	}
	consistency := request.Consistency
	if consistency == "" {
		// Evaluation is a command over source facts. It defaults to strict
		// freshness even when ordinary navigation defaults to latest_ready.
		consistency = ConsistencyRequireCurrent
	}
	record, freshness, returned, err := gateway.session.prepareConsistency(ctx, consistency, request.Revision)
	if err != nil {
		return QueryEnvelope{}, err
	}
	if record.Snapshot.InputVerification.Status != "verified" {
		return QueryEnvelope{}, newLiveError(ErrorQualityEvaluation, "quality evaluation requires a verified source revision", map[string]any{"revision": record.Snapshot.Revision})
	}
	if request.ProfileID == "" || request.ProfileVersion == "" {
		return QueryEnvelope{}, newLiveError(ErrorQualityEvaluation, "quality evaluation requires profile_id and profile_version", nil)
	}
	profile, err := gateway.profiles.ResolveProfile(ctx, request.ProfileID, request.ProfileVersion)
	if err != nil {
		return QueryEnvelope{}, newLiveError(ErrorQualityEvaluation, "quality profile could not be resolved", map[string]any{"error": err.Error()})
	}
	profile = cloneQualityProfile(profile)
	if request.RuleBindings != nil {
		profile.EnabledRules = cloneRuleBindings(request.RuleBindings)
	}
	validated, err := quality.ValidateQualityProfile(profile, gateway.catalog)
	if err != nil {
		return QueryEnvelope{}, newLiveError(ErrorQualityEvaluation, "quality profile or temporary rule bindings are invalid", map[string]any{"error": err.Error()})
	}
	value, err := gateway.modelForScopes(record, request.ScopeIDs)
	if err != nil {
		return QueryEnvelope{}, err
	}
	report, err := qualityadapter.EvaluateModel(validated, value, gateway.catalog)
	if err != nil {
		return QueryEnvelope{}, newLiveError(ErrorQualityEvaluation, "quality evaluation failed", map[string]any{"error": err.Error()})
	}
	reportCopy := cloneQualityReport(report)
	result := QualityEvaluationResult{Status: "evaluated", Temporary: true, Report: &reportCopy}
	maxBytes, maxItems, err := gateway.session.normalizeBudget(request.MaxBytes, request.MaxItems)
	if err != nil {
		return QueryEnvelope{}, err
	}
	emittedBytes := jsonSize(result)
	if emittedBytes > maxBytes {
		return QueryEnvelope{}, newLiveError(ErrorQueryBudget, "quality evaluation exceeds the query byte budget", map[string]any{"max_bytes": maxBytes, "emitted_bytes": emittedBytes})
	}
	gateway.session.rememberTemporaryReport(record.Snapshot.Revision, report)
	return gateway.qualityEnvelope(record, freshness, returned, consistency, result, qualityCoverage(report), []string{}, BudgetUsage{MaxBytes: maxBytes, MaxItems: maxItems, EmittedBytes: emittedBytes, EmittedItems: 1}), nil
}

func (gateway *QualityGateway) GetQualityFindings(ctx context.Context, request QualityFindingsRequest) (QueryEnvelope, error) {
	if err := gateway.require(OperationSearch); err != nil {
		return QueryEnvelope{}, err
	}
	record, freshness, returned, query, err := gateway.session.prepareQuery(ctx, request.QueryRequest)
	if err != nil {
		return QueryEnvelope{}, err
	}
	report, err := gateway.reportFor(record, request.ReportID)
	if err != nil {
		if request.ReportID == "" && isMissingQualityReport(err) {
			return gateway.unavailableQualityEnvelope(record, freshness, returned, query, "no quality report is attached to this revision", "quality:report"), nil
		}
		return QueryEnvelope{}, err
	}
	service := quality.NewQualityQueryService(report)
	options := quality.QualityFindingQueryOptions{ReportID: report.EvaluationID, SubjectID: query.Query.SubjectID, ScopeID: singletonString(query.Query.ScopeIDs), FileID: singletonString(query.Query.FileIDs), RuleID: singletonString(query.Query.RuleIDs), AssessmentKind: singletonString(query.Query.AssessmentKinds), Severity: singletonString(query.Query.Severities), Status: singletonString(query.Query.Statuses)}
	findings, coverage, err := collectQualityFindings(service, report.EvaluationID, options)
	if err != nil {
		return QueryEnvelope{}, err
	}
	findings = filterQualityFindings(findings, query.Query)
	coverage = filterQualityCoverage(coverage, query.Query)
	maxBytes, maxItems, err := gateway.session.normalizeBudget(query.MaxBytes, query.MaxItems)
	if err != nil {
		return QueryEnvelope{}, err
	}
	contextKey := queryContext(record, request, "quality_findings", maxBytes, maxItems)
	offset, err := decodeQueryCursor(query.Cursor, contextKey)
	if err != nil {
		return QueryEnvelope{}, err
	}
	page, total, next, budget, err := pageSlice(findings, offset, maxItems, maxBytes)
	if err != nil {
		return QueryEnvelope{}, err
	}
	if next != "" {
		nextOffset, parseErr := parseOffsetCursor(next)
		if parseErr != nil {
			return QueryEnvelope{}, newLiveError(ErrorQueryCursor, "quality pagination produced an invalid continuation", nil)
		}
		next = encodeQueryCursor(contextKey, nextOffset)
	}
	result := QualityFindingsResult{Status: "observed", ReportID: report.EvaluationID, EvaluationID: report.EvaluationID, SnapshotIDs: append([]string(nil), report.SourceSnapshotIDs...), Items: page, Total: total, NextCursor: next, Coverage: coverage}
	envelope := gateway.qualityEnvelope(record, freshness, returned, query.Consistency, result, qualityCoverageToLive(coverage), []string{"source_context"}, budget)
	envelope.NextCursor = next
	return envelope, nil
}

func (gateway *QualityGateway) GetFindingEvidence(ctx context.Context, request QualityEvidenceRequest) (QueryEnvelope, error) {
	if err := gateway.require(OperationEvidence); err != nil {
		return QueryEnvelope{}, err
	}
	record, freshness, returned, query, err := gateway.session.prepareQuery(ctx, request.QueryRequest)
	if err != nil {
		return QueryEnvelope{}, err
	}
	report, err := gateway.reportFor(record, request.ReportID)
	if err != nil {
		if request.ReportID == "" && isMissingQualityReport(err) {
			return gateway.unavailableQualityEnvelope(record, freshness, returned, query, "no quality report is attached to this revision", "quality:evidence"), nil
		}
		return QueryEnvelope{}, err
	}
	options := quality.QualityEvidenceQueryOptions{IncludeSourceContext: request.IncludeSourceContext}
	if request.IncludeSourceContext {
		options.MaxLines = request.MaxLines
		if options.MaxLines == 0 {
			options.MaxLines = gateway.session.validated.Config.QueryPolicy.DefaultContextLines
		}
		options.MaxBytes = request.MaxContextBytes
		if options.MaxBytes == 0 {
			options.MaxBytes = gateway.session.validated.Config.QueryPolicy.DefaultMaxBytes
		}
		if options.MaxLines < 1 || options.MaxLines > gateway.session.validated.Config.QueryPolicy.HardContextLines || options.MaxBytes < 1 || options.MaxBytes > gateway.session.validated.Config.QueryPolicy.HardMaxBytes {
			return QueryEnvelope{}, newLiveError(ErrorQueryBudget, "quality evidence source-context budget exceeds the configured hard bound", nil)
		}
	}
	evidence, err := quality.NewQualityQueryService(report).GetFindingEvidence(report.EvaluationID, request.FindingID, options)
	if err != nil {
		return QueryEnvelope{}, qualityGatewayError(err)
	}
	result := QualityEvidenceResult{Status: "observed", Evidence: &evidence, Contexts: []SourceContext{}}
	coverage := []CapabilityCoverage{{Capability: "quality:evidence", Status: "observed"}}
	if request.IncludeSourceContext {
		for _, span := range evidence.Finding.Evidence.SourceSpans {
			contextValue, contextErr := gateway.readSourceContext(ctx, record, span, evidence.Finding.SubjectRef.ScopeID, options.MaxLines, options.MaxBytes)
			if contextErr != nil {
				result.Status = "partial"
				continue
			}
			result.Contexts = append(result.Contexts, contextValue)
		}
		if len(result.Contexts) == 0 && len(evidence.Finding.Evidence.SourceSpans) > 0 {
			coverage = append(coverage, CapabilityCoverage{Capability: "source:context", Status: "partial", Reason: "evidence spans were reported but no bounded source context could be read"})
		} else {
			coverage = append(coverage, CapabilityCoverage{Capability: "source:context", Status: "observed"})
		}
	}
	maxBytes, maxItems, err := gateway.session.normalizeBudget(query.MaxBytes, query.MaxItems)
	if err != nil {
		return QueryEnvelope{}, err
	}
	emittedBytes := jsonSize(result)
	if emittedBytes > maxBytes {
		return QueryEnvelope{}, newLiveError(ErrorQueryBudget, "quality evidence exceeds the query byte budget", map[string]any{"max_bytes": maxBytes, "emitted_bytes": emittedBytes})
	}
	return gateway.qualityEnvelope(record, freshness, returned, query.Consistency, result, coverage, []string{}, BudgetUsage{MaxBytes: maxBytes, MaxItems: maxItems, EmittedBytes: emittedBytes, EmittedItems: 1}), nil
}

func (gateway *QualityGateway) CompareQualityReports(ctx context.Context, request QualityCompareRequest) (QueryEnvelope, error) {
	if err := gateway.require(OperationSearch); err != nil {
		return QueryEnvelope{}, err
	}
	if gateway.session == nil {
		return QueryEnvelope{}, newLiveError(ErrorLiveConfigInvalid, "live session is nil", nil)
	}
	if request.SessionID != "" && request.SessionID != gateway.session.validated.Config.SessionID {
		return QueryEnvelope{}, newLiveError("QueryInvalid", "quality comparison session_id does not match the live session", nil)
	}
	if request.PreviousRevision < 1 {
		return QueryEnvelope{}, newLiveError(ErrorRevisionUnavailable, "previous_revision is required for quality comparison", nil)
	}
	previous, ok := gateway.session.store.ReadRevision(gateway.session.validated.Config.SessionID, request.PreviousRevision)
	if !ok {
		return QueryEnvelope{}, newLiveError(ErrorRevisionUnavailable, "previous quality revision is unavailable", map[string]any{"revision": request.PreviousRevision})
	}
	currentConsistency := request.Consistency
	if currentConsistency == "" {
		currentConsistency = ConsistencyLatestReady
	}
	var current *RevisionRecord
	var freshness Freshness
	var returned string
	var err error
	if request.CurrentRevision > 0 {
		current, freshness, returned, err = gateway.session.prepareConsistency(ctx, ConsistencySpecific, request.CurrentRevision)
		if err != nil {
			return QueryEnvelope{}, err
		}
		currentConsistency = ConsistencySpecific
	} else {
		current, freshness, returned, err = gateway.session.prepareConsistency(ctx, currentConsistency, 0)
	}
	if err != nil {
		return QueryEnvelope{}, err
	}
	previousReport, err := gateway.reportFor(previous, request.PreviousReportID)
	if err != nil {
		return QueryEnvelope{}, err
	}
	currentReport, err := gateway.reportFor(current, request.CurrentReportID)
	if err != nil {
		return QueryEnvelope{}, err
	}
	comparison, err := quality.CompareQualityReports(previousReport, currentReport)
	if err != nil {
		return QueryEnvelope{}, qualityGatewayError(err)
	}
	result := QualityComparisonResult{Status: "observed", Comparison: &comparison}
	maxBytes, maxItems, err := gateway.session.normalizeBudget(request.MaxBytes, request.MaxItems)
	if err != nil {
		return QueryEnvelope{}, err
	}
	emittedBytes := jsonSize(result)
	if emittedBytes > maxBytes {
		return QueryEnvelope{}, newLiveError(ErrorQueryBudget, "quality comparison exceeds the query byte budget", map[string]any{"max_bytes": maxBytes, "emitted_bytes": emittedBytes})
	}
	return gateway.qualityEnvelope(current, freshness, returned, currentConsistency, result, []CapabilityCoverage{{Capability: "quality:comparison", Status: "observed"}}, []string{}, BudgetUsage{MaxBytes: maxBytes, MaxItems: maxItems, EmittedBytes: emittedBytes, EmittedItems: 1}), nil
}

func (gateway *QualityGateway) CompareQualityRevisions(ctx context.Context, request QualityCompareRequest) (QueryEnvelope, error) {
	return gateway.CompareQualityReports(ctx, request)
}

func (gateway *QualityGateway) require(operation Operation) error {
	if gateway == nil || gateway.session == nil {
		return newLiveError(ErrorLiveConfigInvalid, "quality gateway is not attached to a live session", nil)
	}
	return gateway.session.requireOperation(operation)
}

func (gateway *QualityGateway) readSourceContext(ctx context.Context, record *RevisionRecord, span quality.SourceSpan, scopeID string, maxLines, maxBytes int) (SourceContext, error) {
	converted := analysis.SourceSpan{FileID: span.FileID, Start: analysis.SpanPosition{ByteOffset: span.Start.ByteOffset, Line: span.Start.Line, Column: span.Start.Column}, End: analysis.SpanPosition{ByteOffset: span.End.ByteOffset, Line: span.End.Line, Column: span.End.Column}, CoordinateSystem: span.CoordinateSystem, ContentHash: analysis.ContentDigest{Algorithm: span.ContentHash.Algorithm, Value: span.ContentHash.Value}}
	envelope, err := gateway.session.GetSourceContext(ctx, SourceContextRequest{SessionID: gateway.session.validated.Config.SessionID, Consistency: ConsistencySpecific, Revision: record.Snapshot.Revision, ScopeID: scopeID, Span: &converted, MaxLines: maxLines, MaxBytes: maxBytes})
	if err != nil {
		return SourceContext{}, err
	}
	result, ok := envelope.Result.(SourceContextResult)
	if !ok {
		return SourceContext{}, newLiveError(ErrorSourceContextOutOfScope, "source context result has an unexpected shape", nil)
	}
	return result.Context, nil
}

func (gateway *QualityGateway) catalogEnvelope(record *RevisionRecord, freshness Freshness, returned string, request QualityCatalogRequest, profiles []QualityProfileInfo, rules []QualityRuleCatalogEntry) (QueryEnvelope, error) {
	maxBytes, maxItems, err := gateway.session.normalizeBudget(request.MaxBytes, request.MaxItems)
	if err != nil {
		return QueryEnvelope{}, err
	}
	contextKey := queryContext(record, request, "quality_catalog", maxBytes, maxItems)
	offset, err := decodeQueryCursor(request.Cursor, contextKey)
	if err != nil {
		return QueryEnvelope{}, err
	}
	page, total, next, budget, err := pageSlice(rules, offset, maxItems, maxBytes)
	if err != nil {
		return QueryEnvelope{}, err
	}
	if next != "" {
		nextOffset, parseErr := parseOffsetCursor(next)
		if parseErr != nil {
			return QueryEnvelope{}, newLiveError(ErrorQueryCursor, "quality catalog pagination produced an invalid continuation", nil)
		}
		next = encodeQueryCursor(contextKey, nextOffset)
	}
	result := QualityCatalogResult{Profiles: append([]QualityProfileInfo(nil), profiles...), Rules: append([]QualityRuleCatalogEntry(nil), page...), Capabilities: append([]quality.CapabilityDescriptor(nil), gateway.catalog.ListQualityCapabilities()...)}
	budget.EmittedBytes = jsonSize(result)
	budget.EmittedItems = len(profiles) + len(page) + len(result.Capabilities)
	if budget.EmittedBytes > maxBytes || budget.EmittedItems > maxItems {
		return QueryEnvelope{}, newLiveError(ErrorQueryBudget, "quality catalog metadata exceeds the query budget", map[string]any{"max_bytes": maxBytes, "max_items": maxItems, "emitted_bytes": budget.EmittedBytes, "emitted_items": budget.EmittedItems})
	}
	budget.Truncated = next != ""
	envelope := gateway.qualityEnvelope(record, freshness, returned, request.Consistency, result, []CapabilityCoverage{{Capability: "quality:catalog", Status: "observed"}}, []string{}, budget)
	envelope.ResultCount = total
	envelope.NextCursor = next
	return envelope, nil
}

func (gateway *QualityGateway) qualityEnvelope(record *RevisionRecord, freshness Freshness, returned string, requested Consistency, result any, coverage []CapabilityCoverage, omitted []string, budget BudgetUsage) QueryEnvelope {
	if budget.MaxBytes == 0 {
		budget.MaxBytes = DefaultQueryMaxBytes
	}
	if budget.MaxItems == 0 {
		budget.MaxItems = 1
	}
	resultCount := budget.EmittedItems
	return QueryEnvelope{SchemaVersion: QuerySchemaVersion, SessionID: gateway.session.validated.Config.SessionID, SnapshotID: record.Snapshot.SnapshotID, Revision: record.Snapshot.Revision, RequestedConsistency: requested, ReturnedConsistency: returned, Freshness: freshness, ScopeIDs: append([]string(nil), record.Snapshot.ScopeIDs...), Result: result, ResultCount: resultCount, OmittedFields: uniqueStrings(omitted), Capabilities: mergeCapabilityCoverage(nil, coverage), Budget: budget, Diagnostics: []QueryDiagnostic{}}
}

func (gateway *QualityGateway) unavailableQualityEnvelope(record *RevisionRecord, freshness Freshness, returned string, request QueryRequest, reason, capability string) QueryEnvelope {
	result := QualityFindingsResult{Status: "unavailable", Items: []quality.QualityFinding{}, Total: 0, Coverage: []quality.QualityCoverage{}}
	return gateway.qualityEnvelope(record, freshness, returned, request.Consistency, result, []CapabilityCoverage{{Capability: capability, Status: "unavailable", Reason: reason}}, []string{"source_context"}, BudgetUsage{MaxBytes: maxOrDefault(request.MaxBytes, DefaultQueryMaxBytes), MaxItems: maxOrDefault(request.MaxItems, DefaultQueryMaxItems), EmittedBytes: jsonSize(result), EmittedItems: 0})
}

func (gateway *QualityGateway) modelForScopes(record *RevisionRecord, scopeIDs []string) (model.Model, error) {
	if record == nil {
		return model.Model{}, newLiveError(ErrorNoReadySnapshot, "no live revision is available for quality evaluation", nil)
	}
	if len(scopeIDs) == 0 {
		return cloneModel(record.Model), nil
	}
	if len(scopeIDs) > 1 {
		for _, scopeID := range scopeIDs {
			if _, ok := record.ScopeModels[scopeID]; !ok {
				return model.Model{}, newLiveError(ErrorRevisionUnavailable, "requested quality scope is unavailable", map[string]any{"scope_id": scopeID})
			}
		}
		value, err := record.Run.CombinedCanonicalModelForScopes(scopeIDs)
		if err != nil {
			return model.Model{}, newLiveError(ErrorQualityEvaluation, "requested quality scopes could not be combined", map[string]any{"error": err.Error()})
		}
		return value, nil
	}
	value, ok := record.ScopeModels[scopeIDs[0]]
	if !ok {
		return model.Model{}, newLiveError(ErrorRevisionUnavailable, "requested quality scope is unavailable", map[string]any{"scope_id": scopeIDs[0]})
	}
	return cloneModel(value), nil
}

func (session *LiveSession) rememberTemporaryReport(revision int, report quality.QualityEvaluation) {
	if session == nil {
		return
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.qualityReports == nil {
		session.qualityReports = make(map[string]quality.QualityEvaluation)
	}
	if session.qualityReportRevisions == nil {
		session.qualityReportRevisions = make(map[string]int)
	}
	key := revisionReportKey(revision, report.EvaluationID)
	session.qualityReports[key] = cloneQualityReport(report)
	session.qualityReportRevisions[key] = revision
}

func (gateway *QualityGateway) reportFor(record *RevisionRecord, reportID string) (quality.QualityEvaluation, error) {
	if record == nil {
		return quality.QualityEvaluation{}, newLiveError(ErrorNoReadySnapshot, "no live revision is available for quality query", nil)
	}
	if reportID == "" {
		if record.Model.QualityReport != nil {
			return cloneQualityReport(*record.Model.QualityReport), nil
		}
		ids := make([]string, 0, len(record.QualityReports))
		for id := range record.QualityReports {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			return cloneQualityReport(record.QualityReports[id]), nil
		}
		return quality.QualityEvaluation{}, newLiveError("QualityReportUnavailable", "no quality report is attached to this revision", map[string]any{"revision": record.Snapshot.Revision})
	}
	for id, report := range record.QualityReports {
		if id == reportID || report.EvaluationID == reportID {
			return cloneQualityReport(report), nil
		}
	}
	gateway.session.mu.RLock()
	key := revisionReportKey(record.Snapshot.Revision, reportID)
	report, ok := gateway.session.qualityReports[key]
	gateway.session.mu.RUnlock()
	if ok {
		return cloneQualityReport(report), nil
	}
	gateway.session.mu.RLock()
	for storedKey, stored := range gateway.session.qualityReports {
		if stored.EvaluationID == reportID {
			revision := gateway.session.qualityReportRevisions[storedKey]
			gateway.session.mu.RUnlock()
			return quality.QualityEvaluation{}, newLiveError(ErrorQualityEvaluation, "quality report belongs to a different source revision", map[string]any{"report_id": reportID, "report_revision": revision, "requested_revision": record.Snapshot.Revision})
		}
	}
	gateway.session.mu.RUnlock()
	return quality.QualityEvaluation{}, newLiveError("QualityReportNotFound", "quality report was not found for the requested revision", map[string]any{"report_id": reportID})
}

func revisionReportKey(revision int, reportID string) string {
	return fmt.Sprintf("%d\x00%s", revision, reportID)
}

func isMissingQualityReport(err error) bool {
	return strings.Contains(errString(err), "QualityReportUnavailable")
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func collectQualityFindings(service *quality.QualityQueryService, reportID string, options quality.QualityFindingQueryOptions) ([]quality.QualityFinding, []quality.QualityCoverage, error) {
	findings := []quality.QualityFinding{}
	cursor := ""
	for {
		options.Cursor = cursor
		options.Limit = quality.MaxQualityQueryLimit
		page, err := service.ListFindings(reportID, options)
		if err != nil {
			return nil, nil, qualityGatewayError(err)
		}
		findings = append(findings, page.Items...)
		if page.NextCursor == "" || len(findings) >= page.Total {
			return findings, page.Coverage, nil
		}
		if page.NextCursor == cursor {
			return nil, nil, newLiveError(ErrorQueryCursor, "quality findings pagination did not advance", nil)
		}
		cursor = page.NextCursor
	}
}

func filterQualityFindings(values []quality.QualityFinding, query StructuralQuery) []quality.QualityFinding {
	result := make([]quality.QualityFinding, 0, len(values))
	for _, finding := range values {
		if query.SubjectID != "" && finding.SubjectRef.ID != query.SubjectID || len(query.ScopeIDs) > 0 && !containsString(query.ScopeIDs, finding.SubjectRef.ScopeID) || len(query.FileIDs) > 0 && (finding.SubjectRef.Kind != "file" || !containsString(query.FileIDs, finding.SubjectRef.ID)) || len(query.RuleIDs) > 0 && !containsString(query.RuleIDs, finding.RuleID) || len(query.AssessmentKinds) > 0 && !containsString(query.AssessmentKinds, finding.AssessmentKind) || len(query.Severities) > 0 && !containsString(query.Severities, finding.Severity) || len(query.Statuses) > 0 && !containsString(query.Statuses, finding.Status) {
			continue
		}
		result = append(result, finding)
	}
	return result
}

func filterQualityCoverage(values []quality.QualityCoverage, query StructuralQuery) []quality.QualityCoverage {
	result := make([]quality.QualityCoverage, 0, len(values))
	for _, value := range values {
		if len(query.RuleIDs) > 0 && !containsString(query.RuleIDs, value.RuleID) {
			continue
		}
		result = append(result, value)
	}
	return result
}

func qualityCoverage(report quality.QualityEvaluation) []CapabilityCoverage {
	return qualityCoverageToLive(report.Coverage)
}

func qualityCoverageToLive(values []quality.QualityCoverage) []CapabilityCoverage {
	result := make([]CapabilityCoverage, 0, len(values))
	for _, value := range values {
		result = append(result, CapabilityCoverage{Capability: "quality:rule:" + value.RuleID, Status: value.Status, Reason: value.Reason})
	}
	return mergeCapabilityCoverage(nil, result)
}

func qualityGatewayError(err error) error {
	if err == nil {
		return nil
	}
	var qualityErr *quality.QualityError
	if errors.As(err, &qualityErr) {
		return newLiveError(string(qualityErr.Code), qualityErr.Message, qualityErr.Details)
	}
	return newLiveError(ErrorQualityEvaluation, err.Error(), nil)
}

func cloneQualityProfile(value quality.QualityProfile) quality.QualityProfile {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone quality.QualityProfile
	if err := json.Unmarshal(data, &clone); err != nil {
		return value
	}
	return clone
}

func cloneRuleBindings(values []quality.RuleBinding) []quality.RuleBinding {
	result := make([]quality.RuleBinding, 0, len(values))
	for _, value := range values {
		result = append(result, cloneRuleBinding(value))
	}
	return result
}

func cloneRuleBinding(value quality.RuleBinding) quality.RuleBinding {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone quality.RuleBinding
	if err := json.Unmarshal(data, &clone); err != nil {
		return value
	}
	return clone
}

func cloneTypedConfigBlock(value quality.TypedConfigBlock) quality.TypedConfigBlock {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone quality.TypedConfigBlock
	if err := json.Unmarshal(data, &clone); err != nil {
		return value
	}
	return clone
}

func profileKey(id, version string) string {
	return strings.TrimSpace(id) + "\x00" + strings.TrimSpace(version)
}

func ruleKey(id, version string) string {
	return strings.TrimSpace(id) + "\x00" + strings.TrimSpace(version)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func singletonString(values []string) string {
	if len(values) != 1 {
		return ""
	}
	return values[0]
}

func maxOrDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func parseOffsetCursor(value string) (int, error) {
	return strconv.Atoi(strings.TrimPrefix(value, "offset:"))
}
