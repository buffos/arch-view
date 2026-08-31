package live

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/quality"
	qualitypolicy "github.com/buffo/arch-view/internal/quality/policy"
)

const (
	PolicyValidateProfile = "validate_profile"
	PolicySaveProfile     = "save_quality_profile"
	PolicySaveProfileAs   = "save_quality_profile_as"
	PolicyPreviewBaseline = "preview_baseline"
	PolicyCreateBaseline  = "create_baseline"
	PolicyAppendBaseline  = "append_baseline"
)

// ExecuteQualityPolicyCommand is the only live entry point for policy
// changes. It deliberately separates permission and freshness checks from the
// quality package's rule/profile semantics.
func (gateway *QualityGateway) ExecuteQualityPolicyCommand(ctx context.Context, command QualityPolicyCommand) (QueryEnvelope, error) {
	if gateway == nil || gateway.session == nil {
		return QueryEnvelope{}, newLiveError(ErrorLiveConfigInvalid, "quality gateway is not attached to a live session", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	command.Operation = strings.TrimSpace(command.Operation)
	if command.SessionID != "" && command.SessionID != gateway.session.validated.Config.SessionID {
		return QueryEnvelope{}, newLiveError("QueryInvalid", "quality policy session_id does not match the live session", nil)
	}
	if command.SessionID == "" {
		command.SessionID = gateway.session.validated.Config.SessionID
	}
	if command.Operation == "" {
		return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy operation is required", nil)
	}
	operation, write := policyOperation(command.Operation)
	if operation == "" {
		return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy operation is unsupported", map[string]any{"operation": command.Operation})
	}
	if err := gateway.session.requireOperation(operation); err != nil {
		return QueryEnvelope{}, err
	}
	if write {
		if gateway.authorizer == nil || strings.TrimSpace(command.Authorization) == "" {
			return QueryEnvelope{}, newLiveError(ErrorQualityPolicyPermissionDenied, "quality policy writes require an explicit authorization", map[string]any{"operation": command.Operation})
		}
		if err := gateway.authorizer.Authorize(ctx, PolicyAuthorization{SessionID: command.SessionID, Operation: operation, Authorization: command.Authorization}); err != nil {
			return QueryEnvelope{}, newLiveError(ErrorQualityPolicyPermissionDenied, "quality policy authorization was rejected", map[string]any{"operation": command.Operation, "error": err.Error()})
		}
	}

	switch command.Operation {
	case PolicyValidateProfile:
		return gateway.validatePolicyProfile(ctx, command)
	case PolicySaveProfile, PolicySaveProfileAs:
		return gateway.savePolicyProfile(ctx, command)
	case PolicyPreviewBaseline, PolicyCreateBaseline, PolicyAppendBaseline:
		return gateway.baselinePolicyCommand(ctx, command)
	default:
		return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy operation is unsupported", map[string]any{"operation": command.Operation})
	}
}

func policyOperation(name string) (Operation, bool) {
	switch name {
	case PolicyValidateProfile:
		return OperationQualityProfileRead, false
	case PolicySaveProfile, PolicySaveProfileAs:
		return OperationQualityProfileWrite, true
	case PolicyPreviewBaseline:
		return OperationBaselineRead, false
	case PolicyCreateBaseline, PolicyAppendBaseline:
		return OperationBaselineWrite, true
	default:
		return "", false
	}
}

func (gateway *QualityGateway) validatePolicyProfile(ctx context.Context, command QualityPolicyCommand) (QueryEnvelope, error) {
	profile, err := gateway.profileForCommand(ctx, command, false)
	if err != nil {
		return QueryEnvelope{}, err
	}
	validated, err := gateway.validateProfile(profile)
	if err != nil {
		return QueryEnvelope{}, policyQualityError(err)
	}
	result := QualityPolicyResult{Status: "validated", Operation: command.Operation, Profile: &validated, PolicyIdentity: policyIdentityForProfile(validated)}
	return gateway.policyEnvelope(ctx, result, nil)
}

func (gateway *QualityGateway) savePolicyProfile(ctx context.Context, command QualityPolicyCommand) (QueryEnvelope, error) {
	if gateway.policy == nil {
		return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy persistence is unavailable for this live session", nil)
	}
	sourceID := strings.TrimSpace(command.SourceProfileID)
	sourceVersion := strings.TrimSpace(command.SourceProfileVersion)
	if sourceID == "" {
		sourceID, sourceVersion = strings.TrimSpace(command.ProfileID), strings.TrimSpace(command.ProfileVersion)
	}
	var profile quality.QualityProfile
	sourceInfo := QualityPolicyProfileInfo{}
	if command.Profile != nil {
		profile = cloneQualityProfile(*command.Profile)
		if sourceID == "" {
			sourceID = strings.TrimSpace(profile.ProfileID)
			sourceVersion = strings.TrimSpace(profile.ProfileVersion)
		}
	} else {
		if sourceID == "" || sourceVersion == "" {
			return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "profile persistence requires a source profile identity", nil)
		}
		var resolveErr error
		profile, sourceInfo, resolveErr = gateway.policy.ProfileDocument(ctx, sourceID, sourceVersion)
		if resolveErr != nil {
			return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "source quality profile could not be resolved", map[string]any{"error": resolveErr.Error()})
		}
	}
	if command.RuleBindings != nil {
		profile.EnabledRules = cloneRuleBindings(command.RuleBindings)
	}
	if command.Operation == PolicySaveProfileAs {
		if strings.TrimSpace(command.ProfileID) == "" || strings.TrimSpace(command.ProfileVersion) == "" || strings.TrimSpace(command.FileName) == "" {
			return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "saving a profile as a new document requires profile_id, profile_version, and file_name", nil)
		}
		profile.ProfileID = strings.TrimSpace(command.ProfileID)
		profile.ProfileVersion = strings.TrimSpace(command.ProfileVersion)
		profile.Baseline = nil
	} else {
		if sourceID == "" || sourceVersion == "" {
			return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "saving a profile requires a source profile identity", nil)
		}
		profile.ProfileID = sourceID
		profile.ProfileVersion = sourceVersion
	}
	validated, err := gateway.validateProfile(profile)
	if err != nil {
		return QueryEnvelope{}, policyQualityError(err)
	}
	fileName := strings.TrimSpace(command.FileName)
	if fileName == "" {
		fileName = sourceInfo.FileName
	}
	if fileName == "" {
		return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "profile persistence requires a destination file", nil)
	}
	overwrite := command.Operation == PolicySaveProfile && command.Overwrite
	if command.Operation == PolicySaveProfileAs && command.Overwrite {
		return QueryEnvelope{}, newLiveError(ErrorQualityProfileConflict, "save-as cannot overwrite an existing profile", nil)
	}
	writeResult, err := gateway.policy.SaveProfile(ctx, validated, fileName, overwrite)
	if err != nil {
		return QueryEnvelope{}, policyWriteError(err)
	}
	audit := gateway.recordPolicyAudit(command, writeResult.RelativePath, "quality profile persisted")
	result := QualityPolicyResult{Status: "saved", Operation: command.Operation, Profile: &validated, PolicyIdentity: policyIdentityForProfile(validated), Write: &writeResult, Audit: &audit, Message: "Quality profile saved."}
	return gateway.policyEnvelope(ctx, result, &audit)
}

func (gateway *QualityGateway) baselinePolicyCommand(ctx context.Context, command QualityPolicyCommand) (QueryEnvelope, error) {
	if gateway.policy == nil && (command.Operation == PolicyCreateBaseline || command.Operation == PolicyAppendBaseline) {
		return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy persistence is unavailable for this live session", nil)
	}
	record, report, err := gateway.currentPolicyReport(ctx, command)
	if err != nil {
		return QueryEnvelope{}, err
	}
	if command.ProfileID != "" && command.ProfileID != report.ProfileID || command.ProfileVersion != "" && command.ProfileVersion != report.ProfileVersion {
		return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "quality report does not match the selected profile", map[string]any{"report_profile_id": report.ProfileID, "report_profile_version": report.ProfileVersion})
	}
	keys, entries, identity, err := gateway.baselineEntries(command, report)
	if err != nil {
		return QueryEnvelope{}, err
	}
	identity.ReportRevision = record.Snapshot.Revision
	baselineID := strings.TrimSpace(command.BaselineID)
	if command.Operation == PolicyAppendBaseline {
		profile, profileInfo, profileErr := gateway.policy.ProfileDocument(ctx, report.ProfileID, report.ProfileVersion)
		if profileErr != nil {
			return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "the quality profile used by the report could not be read from the policy store", map[string]any{"error": profileErr.Error()})
		}
		baselineID = strings.TrimSpace(command.BaselineID)
		if baselineID == "" && profile.Baseline != nil {
			baselineID = profile.Baseline.BaselineID
		}
		baselineFileName := strings.TrimSpace(command.FileName)
		if baselineFileName == "" && profile.Baseline != nil {
			_, baselineInfo, resolveErr := gateway.policy.ResolveBaseline(ctx, profile.Baseline.BaselineID, profile.Baseline.Revision)
			if resolveErr == nil {
				baselineFileName = baselineInfo.FileName
			} else if liveErrorCode(resolveErr) != ErrorBaselineNotFound {
				return QueryEnvelope{}, resolveErr
			}
		}
		if baselineFileName == "" {
			return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "append_baseline requires file_name when the profile has no resolvable canonical baseline", nil)
		}
		appendResult, appendErr := gateway.policy.AppendBaseline(ctx, QualityBaselineAppendRequest{
			Profile: profile, ProfileFileName: profileInfo.FileName, BaselineFileName: baselineFileName,
			BaselineID: baselineID, Entries: entries, Revision: strings.TrimSpace(command.BaselineRevision), ExpectedRevision: strings.TrimSpace(command.ExpectedBaselineRevision),
		})
		if appendErr != nil {
			return QueryEnvelope{}, policyWriteError(appendErr)
		}
		updatedBaseline := appendResult.Baseline
		updatedProfile := appendResult.Profile
		var baselineWrite *QualityPolicyWriteResult
		if appendResult.BaselineWrite.FileName != "" {
			write := appendResult.BaselineWrite
			baselineWrite = &write
		}
		var profileWrite *QualityPolicyWriteResult
		if appendResult.ProfileWrite.FileName != "" {
			write := appendResult.ProfileWrite
			profileWrite = &write
		}
		status := "unchanged"
		message := "No new baseline entries were added."
		if appendResult.Changed {
			status = "updated"
			message = "Quality baseline updated; run quality evaluation again."
		}
		paths := make([]string, 0, 2)
		if appendResult.BaselineWrite.RelativePath != "" {
			paths = append(paths, appendResult.BaselineWrite.RelativePath)
		}
		if appendResult.ProfileWrite.RelativePath != "" {
			paths = append(paths, appendResult.ProfileWrite.RelativePath)
		}
		auditDetails := "quality baseline append produced no changes"
		if appendResult.Changed {
			auditDetails = "quality baseline appended and profile reference updated"
		}
		recorded := gateway.recordPolicyAudit(command, strings.Join(paths, ","), auditDetails)
		audit := &recorded
		result := QualityPolicyResult{
			Status: status, Operation: command.Operation, Message: message, Profile: &updatedProfile, Baseline: &updatedBaseline,
			FindingKeys: keys, AddedFindingKeys: findingKeysForBaselineEntries(appendResult.Added), ExistingFindingKeys: findingKeysForBaselineEntries(appendResult.Existing),
			PolicyIdentity: identity, Write: baselineWrite, ProfileWrite: profileWrite, Audit: audit, ReevaluationRequired: appendResult.Changed,
		}
		return gateway.policyEnvelope(ctx, result, audit)
	}
	if baselineID == "" {
		return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "baseline_id is required", nil)
	}
	baseline := quality.Baseline{SchemaVersion: quality.BaselineSchemaVersion, BaselineID: baselineID, Revision: strings.TrimSpace(command.BaselineRevision), Entries: entries, Extensions: []quality.ExtensionBlock{}}
	if err := quality.ValidateBaseline(baseline); err != nil {
		return QueryEnvelope{}, policyQualityError(err)
	}
	preview := QualityBaselinePreview{BaselineID: baseline.BaselineID, Revision: baseline.Revision, FindingKeys: keys, Entries: append([]quality.BaselineEntry(nil), entries...), PolicyIdentity: identity}
	if command.Operation == PolicyPreviewBaseline {
		result := QualityPolicyResult{Status: "preview", Operation: command.Operation, Preview: &preview, FindingKeys: keys, PolicyIdentity: identity, Message: "Baseline preview only; no files were changed."}
		return gateway.policyEnvelope(ctx, result, nil)
	}
	if strings.TrimSpace(command.FileName) == "" || strings.TrimSpace(command.Reason) == "" {
		return QueryEnvelope{}, newLiveError(ErrorQualityPolicyIncompatible, "baseline creation requires file_name and reason", nil)
	}
	writeResult, err := gateway.policy.SaveBaseline(ctx, baseline, strings.TrimSpace(command.FileName), command.Overwrite)
	if err != nil {
		return QueryEnvelope{}, policyWriteError(err)
	}
	audit := gateway.recordPolicyAudit(command, writeResult.RelativePath, "quality baseline persisted")
	result := QualityPolicyResult{Status: "created", Operation: command.Operation, Baseline: &baseline, Preview: &preview, FindingKeys: keys, PolicyIdentity: identity, Write: &writeResult, Audit: &audit, Message: "Quality baseline created."}
	_ = record // record anchors the exact report revision used above.
	return gateway.policyEnvelope(ctx, result, &audit)
}

func (gateway *QualityGateway) profileForCommand(ctx context.Context, command QualityPolicyCommand, requireExisting bool) (quality.QualityProfile, error) {
	if command.Profile != nil {
		return cloneQualityProfile(*command.Profile), nil
	}
	if strings.TrimSpace(command.ProfileID) == "" || strings.TrimSpace(command.ProfileVersion) == "" {
		return quality.QualityProfile{}, newLiveError(ErrorQualityPolicyIncompatible, "profile_id and profile_version are required", nil)
	}
	if gateway.policy != nil {
		profile, err := gateway.policy.ResolveProfile(ctx, command.ProfileID, command.ProfileVersion)
		if err == nil {
			return profile, nil
		}
		if requireExisting {
			return quality.QualityProfile{}, newLiveError(ErrorQualityPolicyIncompatible, "quality profile could not be resolved", map[string]any{"error": err.Error()})
		}
	}
	if gateway.profiles == nil {
		return quality.QualityProfile{}, newLiveError(ErrorQualityPolicyIncompatible, "quality profile resolver is unavailable", nil)
	}
	profile, err := gateway.profiles.ResolveProfile(ctx, command.ProfileID, command.ProfileVersion)
	if err != nil {
		return quality.QualityProfile{}, newLiveError(ErrorQualityPolicyIncompatible, "quality profile could not be resolved", map[string]any{"error": err.Error()})
	}
	return profile, nil
}

func (gateway *QualityGateway) validateProfile(profile quality.QualityProfile) (quality.QualityProfile, error) {
	if gateway.policy != nil {
		return gateway.policy.ValidateProfile(profile)
	}
	if gateway.catalog == nil {
		return quality.QualityProfile{}, newLiveError(ErrorQualityPolicyIncompatible, "quality rule catalog is unavailable", nil)
	}
	return quality.ValidateQualityProfile(profile, gateway.catalog)
}

func (gateway *QualityGateway) currentPolicyReport(ctx context.Context, command QualityPolicyCommand) (*RevisionRecord, quality.QualityEvaluation, error) {
	record, _, _, err := gateway.session.prepareConsistency(ctx, ConsistencyRequireCurrent, 0)
	if err != nil {
		return nil, quality.QualityEvaluation{}, err
	}
	if command.ReportRevision > 0 && command.ReportRevision != record.Snapshot.Revision {
		return nil, quality.QualityEvaluation{}, newLiveError(ErrorBaselineRevisionStale, "the requested quality report revision is no longer current", map[string]any{"requested_revision": command.ReportRevision, "current_revision": record.Snapshot.Revision})
	}
	report, err := gateway.reportFor(record, strings.TrimSpace(command.ReportID))
	if err != nil {
		return nil, quality.QualityEvaluation{}, err
	}
	if err := quality.ValidateQualityEvaluation(report); err != nil {
		return nil, quality.QualityEvaluation{}, policyQualityError(err)
	}
	if report.ModelRevision != "" {
		modelWithoutReport, modelErr := canonical.WithQualityReport(record.Model, nil)
		if modelErr != nil || report.ModelRevision != modelWithoutReport.ModelID {
			return nil, quality.QualityEvaluation{}, newLiveError(ErrorQualityPolicyIncompatible, "quality report model identity does not match the current live revision", nil)
		}
	}
	// IDs and versions alone are not enough: a profile document can change
	// while retaining its version. Resolve the current document and compare
	// its canonical digest with the digest captured by the report before any
	// finding is allowed into a baseline.
	profile, profileErr := gateway.profileForReport(ctx, report.ProfileID, report.ProfileVersion)
	if profileErr != nil {
		return nil, quality.QualityEvaluation{}, newLiveError(ErrorQualityPolicyIncompatible, "the quality profile used by the report could not be resolved", map[string]any{"error": profileErr.Error()})
	}
	validatedProfile, validateErr := gateway.validateProfile(profile)
	if validateErr != nil {
		return nil, quality.QualityEvaluation{}, policyQualityError(validateErr)
	}
	// A baseline is evaluation context, not part of the rule configuration
	// being reviewed. Rebuild the current profile with the report's baseline
	// reference before comparing digests so reports from baseline_mode=none or
	// baseline_mode=selected can still be appended after review.
	reportProfile := cloneQualityProfile(validatedProfile)
	reportProfile.Baseline = report.Baseline
	expectedProfileDigest := profileDigest(reportProfile)
	if expectedProfileDigest == nil || report.ProfileDigest == nil || *expectedProfileDigest != *report.ProfileDigest {
		details := map[string]any{"profile_id": report.ProfileID, "profile_version": report.ProfileVersion}
		if expectedProfileDigest != nil {
			details["current_profile_digest"] = expectedProfileDigest.Value
		}
		if report.ProfileDigest != nil {
			details["report_profile_digest"] = report.ProfileDigest.Value
		}
		return nil, quality.QualityEvaluation{}, newLiveError(ErrorQualityPolicyIncompatible, "the quality report was produced by a different profile document", details)
	}
	if !reportSnapshotsMatch(record, report) {
		return nil, quality.QualityEvaluation{}, newLiveError(ErrorQualityPolicyIncompatible, "quality report source snapshots do not match the current live revision", nil)
	}
	return record, report, nil
}

// profileForReport resolves the persisted profile first so a policy append is
// visible to the next temporary evaluation. Hosts that use an external or
// in-memory profile source fall back to that resolver when no persisted
// document matches.
func (gateway *QualityGateway) profileForReport(ctx context.Context, profileID, profileVersion string) (quality.QualityProfile, error) {
	if gateway.policy != nil {
		if profile, err := gateway.policy.ResolveProfile(ctx, profileID, profileVersion); err == nil {
			return profile, nil
		}
	}
	if gateway.profiles != nil {
		return gateway.profiles.ResolveProfile(ctx, profileID, profileVersion)
	}
	if gateway.policy != nil {
		return gateway.policy.ResolveProfile(ctx, profileID, profileVersion)
	}
	return quality.QualityProfile{}, errors.New("quality profile resolver is unavailable")
}

func (gateway *QualityGateway) baselineEntries(command QualityPolicyCommand, report quality.QualityEvaluation) ([]string, []quality.BaselineEntry, QualityPolicyIdentity, error) {
	if strings.TrimSpace(command.Reason) == "" {
		return nil, nil, QualityPolicyIdentity{}, newLiveError(ErrorQualityPolicyIncompatible, "baseline reason is required", nil)
	}
	if len(command.FindingKeys) == 0 {
		return nil, nil, QualityPolicyIdentity{}, newLiveError(ErrorQualityPolicyIncompatible, "baseline requires one or more exact finding keys", nil)
	}
	seen := make(map[string]struct{}, len(command.FindingKeys))
	keys := make([]string, 0, len(command.FindingKeys))
	entries := make([]quality.BaselineEntry, 0, len(command.FindingKeys))
	for _, raw := range command.FindingKeys {
		key := strings.TrimSpace(raw)
		if key == "" {
			return nil, nil, QualityPolicyIdentity{}, newLiveError(ErrorQualityPolicyIncompatible, "finding keys may not be empty", nil)
		}
		if _, ok := seen[key]; ok {
			return nil, nil, QualityPolicyIdentity{}, newLiveError(ErrorQualityPolicyIncompatible, "finding keys must be unique", map[string]any{"finding_key": key})
		}
		seen[key] = struct{}{}
		var finding *quality.QualityFinding
		for index := range report.Findings {
			if report.Findings[index].FindingKey == key {
				finding = &report.Findings[index]
				break
			}
		}
		if finding == nil || finding.Status != quality.StatusActive {
			return nil, nil, QualityPolicyIdentity{}, newLiveError(ErrorQualityPolicyIncompatible, "baseline keys must identify active findings in the current report", map[string]any{"finding_key": key})
		}
		if !coverageObserved(report, finding.RuleID, finding.RuleVersion) {
			return nil, nil, QualityPolicyIdentity{}, newLiveError(ErrorQualityPolicyIncompatible, "baseline cannot be created from incomplete rule coverage", map[string]any{"rule_id": finding.RuleID, "rule_version": finding.RuleVersion})
		}
		entry, err := quality.CreateBaselineEntry(report, key, command.Reason, command.Owner)
		if err != nil {
			return nil, nil, QualityPolicyIdentity{}, policyQualityError(err)
		}
		keys = append(keys, key)
		entries = append(entries, entry)
	}
	sort.Strings(keys)
	sort.Slice(entries, func(i, j int) bool { return entries[i].FindingKey < entries[j].FindingKey })
	return keys, entries, policyIdentityForReport(report), nil
}

func coverageObserved(report quality.QualityEvaluation, ruleID, version string) bool {
	found := false
	for _, coverage := range report.Coverage {
		if coverage.RuleID != ruleID || coverage.RuleVersion != version {
			continue
		}
		found = true
		if coverage.Status != quality.CoverageObserved {
			return false
		}
	}
	return found
}

func findingKeysForBaselineEntries(entries []quality.BaselineEntry) []string {
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.FindingKey)
	}
	sort.Strings(result)
	return result
}

func reportSnapshotsMatch(record *RevisionRecord, report quality.QualityEvaluation) bool {
	if record == nil || record.Model.SourceIndex == nil {
		return len(report.SourceSnapshotIDs) == 0
	}
	want := make([]string, 0, len(record.Model.SourceIndex.Snapshots))
	if record.Model.SourceIndex.Projection != nil {
		want = append(want, record.Model.SourceIndex.Projection.SnapshotID)
	} else {
		for _, snapshot := range record.Model.SourceIndex.Snapshots {
			want = append(want, snapshot.SnapshotID)
		}
	}
	sort.Strings(want)
	have := append([]string(nil), report.SourceSnapshotIDs...)
	sort.Strings(have)
	if len(want) != len(have) {
		return false
	}
	for index := range want {
		if want[index] != have[index] {
			return false
		}
	}
	return true
}

func policyIdentityForProfile(profile quality.QualityProfile) QualityPolicyIdentity {
	return QualityPolicyIdentity{ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion, ProfileDigest: profileDigest(profile), RuleVersions: ruleVersionsForProfile(profile), FormulaVersions: []string{}}
}

func policyIdentityForReport(report quality.QualityEvaluation) QualityPolicyIdentity {
	identity := QualityPolicyIdentity{ProfileID: report.ProfileID, ProfileVersion: report.ProfileVersion, ProfileDigest: report.ProfileDigest, ReportID: report.EvaluationID, ReportDigest: report.ReportDigest, RuleVersions: []string{}, FormulaVersions: []string{}}
	seenRules := map[string]struct{}{}
	for _, coverage := range report.Coverage {
		key := coverage.RuleID + "@" + coverage.RuleVersion
		if _, ok := seenRules[key]; !ok {
			seenRules[key] = struct{}{}
			identity.RuleVersions = append(identity.RuleVersions, key)
		}
	}
	seenFormulas := map[string]struct{}{}
	for _, metric := range report.Metrics {
		key := metric.MetricID + "@" + metric.FormulaVersion
		if _, ok := seenFormulas[key]; !ok {
			seenFormulas[key] = struct{}{}
			identity.FormulaVersions = append(identity.FormulaVersions, key)
		}
	}
	sort.Strings(identity.RuleVersions)
	sort.Strings(identity.FormulaVersions)
	return identity
}

func ruleVersionsForProfile(profile quality.QualityProfile) []string {
	result := make([]string, 0, len(profile.EnabledRules))
	for _, binding := range profile.EnabledRules {
		result = append(result, binding.RuleID+"@"+binding.RuleVersion)
	}
	sort.Strings(result)
	return result
}

func profileDigest(profile quality.QualityProfile) *quality.ContentDigest {
	digest := quality.ProfileDigest(profile)
	return &digest
}

func (gateway *QualityGateway) policyEnvelope(ctx context.Context, result QualityPolicyResult, audit *QualityPolicyAudit) (QueryEnvelope, error) {
	if err := ctx.Err(); err != nil {
		return QueryEnvelope{}, err
	}
	record, freshness, returned, err := gateway.session.prepareConsistency(ctx, ConsistencyLatestReady, 0)
	if err != nil {
		return QueryEnvelope{}, err
	}
	if audit != nil {
		result.Audit = audit
	}
	maxBytes, maxItems, err := gateway.session.normalizeBudget(0, 1)
	if err != nil {
		return QueryEnvelope{}, err
	}
	budget := BudgetUsage{MaxBytes: maxBytes, MaxItems: maxItems, EmittedBytes: jsonSize(result), EmittedItems: 1}
	if budget.EmittedBytes > maxBytes {
		return QueryEnvelope{}, newLiveError(ErrorQueryBudget, "quality policy result exceeds the query byte budget", nil)
	}
	return gateway.qualityEnvelope(record, freshness, returned, ConsistencyLatestReady, result, []CapabilityCoverage{{Capability: "quality:policy", Status: "observed"}}, []string{}, budget), nil
}

func (gateway *QualityGateway) recordPolicyAudit(command QualityPolicyCommand, relativePath, details string) QualityPolicyAudit {
	now := time.Now().UTC()
	audit := QualityPolicyAudit{AuditID: digestID("quality-policy-audit", command.SessionID, command.Operation, relativePath, now.Format(time.RFC3339Nano)), Operation: command.Operation, SessionID: command.SessionID, Authorized: true, At: now.Format(time.RFC3339Nano), RelativePath: relativePath, Details: details}
	return audit
}

func policyQualityError(err error) error {
	if err == nil {
		return nil
	}
	var qualityErr *quality.QualityError
	if errors.As(err, &qualityErr) {
		return newLiveError(ErrorQualityPolicyIncompatible, qualityErr.Message, qualityErr.Details)
	}
	return newLiveError(ErrorQualityPolicyIncompatible, err.Error(), nil)
}

func policyWriteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, qualitypolicy.ErrDestinationConflict) {
		return newLiveError(ErrorQualityProfileConflict, err.Error(), nil)
	}
	if errors.Is(err, qualitypolicy.ErrInvalidDestination) {
		return newLiveError(ErrorQualityPolicyIncompatible, err.Error(), nil)
	}
	if strings.Contains(err.Error(), "destination already exists") {
		return newLiveError(ErrorQualityProfileConflict, err.Error(), nil)
	}
	return newLiveError(ErrorQualityPolicyIncompatible, err.Error(), nil)
}
