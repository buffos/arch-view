package live

import (
	"context"

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
