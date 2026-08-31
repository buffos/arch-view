package live

import (
	"context"
	"errors"
	"sort"

	"github.com/buffo/arch-view/internal/quality"
	qualitypolicy "github.com/buffo/arch-view/internal/quality/policy"
)

// FileQualityPolicyService adapts the file-backed quality policy service to
// the live boundary without exposing filesystem-specific types in live's
// public request/response contract.
type FileQualityPolicyService struct {
	service *qualitypolicy.Service
}

func NewFileQualityPolicyService(service *qualitypolicy.Service) *FileQualityPolicyService {
	return &FileQualityPolicyService{service: service}
}

func (adapter *FileQualityPolicyService) ValidateProfile(profile quality.QualityProfile) (quality.QualityProfile, error) {
	if adapter == nil || adapter.service == nil {
		return quality.QualityProfile{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy service is unavailable", nil)
	}
	return adapter.service.ValidateProfile(profile)
}

func (adapter *FileQualityPolicyService) ResolveProfile(ctx context.Context, id, version string) (quality.QualityProfile, error) {
	if adapter == nil || adapter.service == nil {
		return quality.QualityProfile{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy service is unavailable", nil)
	}
	return adapter.service.ResolveProfile(ctx, id, version)
}

func (adapter *FileQualityPolicyService) ProfileDocument(ctx context.Context, id, version string) (quality.QualityProfile, QualityPolicyProfileInfo, error) {
	if adapter == nil || adapter.service == nil {
		return quality.QualityProfile{}, QualityPolicyProfileInfo{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy service is unavailable", nil)
	}
	profile, info, err := adapter.service.Store.ProfileDocument(ctx, id, version)
	if err != nil {
		return quality.QualityProfile{}, QualityPolicyProfileInfo{}, err
	}
	return profile, QualityPolicyProfileInfo{ProfileID: info.ProfileID, ProfileVersion: info.ProfileVersion, FileName: info.FileName, Status: info.Status, Reason: info.Reason}, nil
}

func (adapter *FileQualityPolicyService) SaveProfile(ctx context.Context, profile quality.QualityProfile, fileName string, overwrite bool) (QualityPolicyWriteResult, error) {
	if adapter == nil || adapter.service == nil {
		return QualityPolicyWriteResult{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy service is unavailable", nil)
	}
	result, err := adapter.service.SaveProfile(ctx, profile, fileName, overwrite)
	if err != nil {
		return QualityPolicyWriteResult{}, err
	}
	return QualityPolicyWriteResult{FileName: result.FileName, RelativePath: result.RelativePath, Digest: result.Digest, Overwritten: result.Overwritten}, nil
}

func (adapter *FileQualityPolicyService) SaveBaseline(ctx context.Context, baseline quality.Baseline, fileName string, overwrite bool) (QualityPolicyWriteResult, error) {
	if adapter == nil || adapter.service == nil {
		return QualityPolicyWriteResult{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy service is unavailable", nil)
	}
	result, err := adapter.service.SaveBaseline(ctx, baseline, fileName, overwrite)
	if err != nil {
		return QualityPolicyWriteResult{}, err
	}
	return QualityPolicyWriteResult{FileName: result.FileName, RelativePath: result.RelativePath, Digest: result.Digest, Overwritten: result.Overwritten}, nil
}

func (adapter *FileQualityPolicyService) ListBaselines(ctx context.Context) ([]QualityBaselineInfo, error) {
	if adapter == nil || adapter.service == nil || adapter.service.Store == nil {
		return nil, newLiveError(ErrorQualityPolicyIncompatible, "quality policy service is unavailable", nil)
	}
	values, err := adapter.service.ListBaselines(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]QualityBaselineInfo, 0, len(values))
	for _, value := range values {
		result = append(result, QualityBaselineInfo{BaselineID: value.BaselineID, Revision: value.Revision, FileName: value.FileName, Status: value.Status, EntryCount: value.EntryCount, Profiles: append([]string(nil), value.Profiles...), Reason: value.Reason})
	}
	return result, nil
}

func (adapter *FileQualityPolicyService) ReadBaseline(ctx context.Context, fileName string) (quality.Baseline, QualityBaselineInfo, error) {
	if adapter == nil || adapter.service == nil || adapter.service.Store == nil {
		return quality.Baseline{}, QualityBaselineInfo{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy service is unavailable", nil)
	}
	baseline, err := adapter.service.Store.ReadBaseline(ctx, fileName)
	if err != nil {
		return quality.Baseline{}, QualityBaselineInfo{}, err
	}
	if err := quality.ValidateBaseline(baseline); err != nil {
		return quality.Baseline{}, QualityBaselineInfo{}, err
	}
	return baseline, QualityBaselineInfo{BaselineID: baseline.BaselineID, Revision: baseline.Revision, FileName: fileName, Status: "available", EntryCount: len(baseline.Entries), Profiles: baselineProfiles(baseline)}, nil
}

func (adapter *FileQualityPolicyService) ResolveBaseline(ctx context.Context, baselineID, revision string) (quality.Baseline, QualityBaselineInfo, error) {
	if adapter == nil || adapter.service == nil || adapter.service.Store == nil {
		return quality.Baseline{}, QualityBaselineInfo{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy service is unavailable", nil)
	}
	baseline, info, err := adapter.service.ResolveBaseline(ctx, baselineID, revision)
	if err != nil {
		if errors.Is(err, qualitypolicy.ErrBaselineNotFound) {
			return quality.Baseline{}, QualityBaselineInfo{}, newLiveError(ErrorBaselineNotFound, "quality baseline was not found", map[string]any{"baseline_id": baselineID, "revision": revision})
		}
		if errors.Is(err, qualitypolicy.ErrBaselineConflict) {
			return quality.Baseline{}, QualityBaselineInfo{}, newLiveError(ErrorQualityPolicyIncompatible, "quality baseline resolution is ambiguous", map[string]any{"baseline_id": baselineID, "revision": revision, "error": err.Error()})
		}
		if errors.Is(err, qualitypolicy.ErrBaselineInvalid) {
			return quality.Baseline{}, QualityBaselineInfo{}, newLiveError(ErrorQualityPolicyIncompatible, "quality baseline is invalid", map[string]any{"baseline_id": baselineID, "revision": revision, "error": err.Error()})
		}
		return quality.Baseline{}, QualityBaselineInfo{}, err
	}
	return baseline, QualityBaselineInfo{BaselineID: info.BaselineID, Revision: info.Revision, FileName: info.FileName, Status: info.Status, EntryCount: info.EntryCount, Profiles: append([]string(nil), info.Profiles...), Reason: info.Reason}, nil
}

func (adapter *FileQualityPolicyService) AppendBaseline(ctx context.Context, request QualityBaselineAppendRequest) (QualityBaselineAppendResult, error) {
	if adapter == nil || adapter.service == nil {
		return QualityBaselineAppendResult{}, newLiveError(ErrorQualityPolicyIncompatible, "quality policy service is unavailable", nil)
	}
	result, err := adapter.service.AppendBaseline(ctx, qualitypolicy.BaselineAppendRequest{
		Profile: request.Profile, ProfileFileName: request.ProfileFileName, BaselineFileName: request.BaselineFileName,
		BaselineID: request.BaselineID, Entries: append([]quality.BaselineEntry(nil), request.Entries...), Revision: request.Revision, ExpectedRevision: request.ExpectedRevision,
	})
	if err != nil {
		return QualityBaselineAppendResult{}, err
	}
	return QualityBaselineAppendResult{
		Profile: result.Profile, Baseline: result.Baseline, Added: append([]quality.BaselineEntry(nil), result.Added...), Existing: append([]quality.BaselineEntry(nil), result.Existing...),
		BaselineWrite: QualityPolicyWriteResult{FileName: result.BaselineWrite.FileName, RelativePath: result.BaselineWrite.RelativePath, Digest: result.BaselineWrite.Digest, Overwritten: result.BaselineWrite.Overwritten},
		ProfileWrite:  QualityPolicyWriteResult{FileName: result.ProfileWrite.FileName, RelativePath: result.ProfileWrite.RelativePath, Digest: result.ProfileWrite.Digest, Overwritten: result.ProfileWrite.Overwritten},
		Changed:       result.Changed,
	}, nil
}

func baselineProfiles(value quality.Baseline) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, entry := range value.Entries {
		key := entry.ProfileID + "@" + entry.ProfileVersion
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
