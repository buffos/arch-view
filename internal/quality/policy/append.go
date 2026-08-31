package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/buffo/arch-view/internal/quality"
)

// BaselineAppendRequest describes a managed baseline update. Both filenames
// are direct JSON document names within the policy directories. The profile
// is updated to point at the resulting baseline revision.
type BaselineAppendRequest struct {
	Profile          quality.QualityProfile
	ProfileFileName  string
	BaselineFileName string
	BaselineID       string
	Entries          []quality.BaselineEntry
	Revision         string
	ExpectedRevision string
}

type BaselineAppendResult struct {
	Profile       quality.QualityProfile
	Baseline      quality.Baseline
	Added         []quality.BaselineEntry
	Existing      []quality.BaselineEntry
	BaselineWrite WriteResult
	ProfileWrite  WriteResult
	Changed       bool
}

var appendLocks sync.Mutex

// AppendBaseline merges entries into one project-local baseline and updates
// the selected profile reference. The two documents are written one at a time
// with rollback if the second write fails. Each document publication itself is
// atomic and the in-process lock prevents concurrent managed updates from
// losing entries.
func (service *Service) AppendBaseline(ctx context.Context, request BaselineAppendRequest) (BaselineAppendResult, error) {
	if err := contextError(ctx); err != nil {
		return BaselineAppendResult{}, err
	}
	if service == nil || service.Store == nil {
		return BaselineAppendResult{}, fmt.Errorf("quality policy store is unavailable")
	}
	profile, err := service.ValidateProfile(request.Profile)
	if err != nil {
		return BaselineAppendResult{}, err
	}
	profileFileName := strings.TrimSpace(request.ProfileFileName)
	baselineFileName := strings.TrimSpace(request.BaselineFileName)
	if profileFileName == "" || baselineFileName == "" {
		return BaselineAppendResult{}, fmt.Errorf("managed baseline update requires profile and baseline file names")
	}
	if len(request.Entries) == 0 {
		return BaselineAppendResult{}, quality.WrapQualityError(quality.ErrorBaselineInvalid, "managed baseline update requires one or more entries", nil, nil)
	}
	if _, err := safeDocumentPath(filepath.Join(service.Store.Root(), ProfileDirectoryName), profileFileName); err != nil {
		return BaselineAppendResult{}, err
	}
	if _, err := safeDocumentPath(filepath.Join(service.Store.Root(), BaselineDirectoryName), baselineFileName); err != nil {
		return BaselineAppendResult{}, err
	}
	appendLocks.Lock()
	defer appendLocks.Unlock()

	currentProfile, profileInfo, err := service.Store.ProfileDocument(ctx, profile.ProfileID, profile.ProfileVersion)
	if err != nil {
		return BaselineAppendResult{}, fmt.Errorf("managed baseline profile could not be resolved: %w", err)
	}
	validatedCurrentProfile, validateCurrentErr := service.ValidateProfile(currentProfile)
	if validateCurrentErr != nil {
		return BaselineAppendResult{}, fmt.Errorf("managed baseline profile is invalid: %w", validateCurrentErr)
	}
	if profileInfo.FileName != profileFileName || quality.ProfileDigest(validatedCurrentProfile) != quality.ProfileDigest(profile) {
		return BaselineAppendResult{}, fmt.Errorf("%w: quality profile changed since it was read", ErrBaselineConflict)
	}
	profile = validatedCurrentProfile

	baseline, baselineExists, err := service.readManagedBaseline(ctx, baselineFileName)
	if err != nil {
		return BaselineAppendResult{}, err
	}
	baselineID := strings.TrimSpace(request.BaselineID)
	if profile.Baseline != nil {
		if baselineID == "" {
			baselineID = profile.Baseline.BaselineID
		}
		if baselineID != profile.Baseline.BaselineID {
			return BaselineAppendResult{}, fmt.Errorf("%w: baseline ID does not match the profile reference", ErrBaselineConflict)
		}
	}
	if baselineExists {
		if baselineID == "" {
			baselineID = baseline.BaselineID
		}
		if baseline.BaselineID != baselineID {
			return BaselineAppendResult{}, fmt.Errorf("%w: baseline file contains %s, requested %s", ErrBaselineConflict, baseline.BaselineID, baselineID)
		}
		if profile.Baseline != nil && profile.Baseline.Revision != "" && profile.Baseline.Revision != baseline.Revision {
			return BaselineAppendResult{}, fmt.Errorf("%w: profile references baseline revision %s but the file contains %s", ErrBaselineConflict, profile.Baseline.Revision, baseline.Revision)
		}
	} else {
		if baselineID == "" {
			baselineID = DefaultBaselineID(baselineFileName)
		}
		baseline = quality.Baseline{SchemaVersion: quality.BaselineSchemaVersion, BaselineID: baselineID, Entries: []quality.BaselineEntry{}, Extensions: []quality.ExtensionBlock{}}
	}
	if baselineID == "" {
		return BaselineAppendResult{}, fmt.Errorf("baseline ID is required")
	}
	if request.ExpectedRevision != "" {
		if !baselineExists {
			return BaselineAppendResult{}, fmt.Errorf("%w: expected baseline revision %s, but the baseline does not exist", ErrBaselineConflict, request.ExpectedRevision)
		}
		if strings.TrimSpace(request.ExpectedRevision) != baseline.Revision {
			return BaselineAppendResult{}, fmt.Errorf("%w: expected baseline revision %s, found %s", ErrBaselineConflict, request.ExpectedRevision, baseline.Revision)
		}
	}
	for _, entry := range request.Entries {
		if entry.ProfileID != profile.ProfileID || entry.ProfileVersion != profile.ProfileVersion {
			return BaselineAppendResult{}, fmt.Errorf("%w: baseline entry %s belongs to %s@%s, not %s@%s", ErrBaselineConflict, entry.FindingKey, entry.ProfileID, entry.ProfileVersion, profile.ProfileID, profile.ProfileVersion)
		}
	}
	merged, err := quality.MergeBaselineEntries(baseline, request.Entries)
	if err != nil {
		return BaselineAppendResult{}, err
	}
	result := BaselineAppendResult{Profile: profile, Baseline: merged.Baseline, Added: merged.Added, Existing: merged.Existing}
	if len(merged.Added) == 0 {
		if profile.Baseline == nil || profile.Baseline.BaselineID != baseline.BaselineID || profile.Baseline.Revision != baseline.Revision {
			profile.Baseline = &quality.BaselineRef{BaselineID: baseline.BaselineID, Revision: baseline.Revision}
			validated, validateErr := service.ValidateProfile(profile)
			if validateErr != nil {
				return BaselineAppendResult{}, validateErr
			}
			profileWrite, writeErr := service.Store.SaveProfile(ctx, validated, profileFileName, true)
			if writeErr != nil {
				return BaselineAppendResult{}, writeErr
			}
			result.Profile = validated
			result.ProfileWrite = profileWrite
			result.Changed = true
		}
		return result, nil
	}

	result.Baseline.Revision = strings.TrimSpace(request.Revision)
	if result.Baseline.Revision == "" {
		result.Baseline.Revision, err = quality.NextBaselineRevision(baseline.Revision)
		if err != nil {
			return BaselineAppendResult{}, err
		}
	} else if baselineExists && result.Baseline.Revision == baseline.Revision {
		return BaselineAppendResult{}, fmt.Errorf("%w: new baseline revision must differ from the current revision", ErrBaselineConflict)
	}
	if profile.Baseline == nil || profile.Baseline.BaselineID != result.Baseline.BaselineID || profile.Baseline.Revision != result.Baseline.Revision {
		profile.Baseline = &quality.BaselineRef{BaselineID: result.Baseline.BaselineID, Revision: result.Baseline.Revision}
	}
	validated, err := service.ValidateProfile(profile)
	if err != nil {
		return BaselineAppendResult{}, err
	}
	result.Profile = validated

	// Re-read both documents after the merge calculation. This catches an
	// external writer that changed the policy while this request was running.
	latestProfile, latestInfo, readErr := service.Store.ProfileDocument(ctx, profile.ProfileID, profile.ProfileVersion)
	latestValidatedProfile, latestValidateErr := service.ValidateProfile(latestProfile)
	if readErr != nil || latestValidateErr != nil || latestInfo.FileName != profileFileName || quality.ProfileDigest(latestValidatedProfile) != quality.ProfileDigest(validatedCurrentProfile) {
		return BaselineAppendResult{}, fmt.Errorf("%w: quality profile changed before baseline publication", ErrBaselineConflict)
	}
	latestBaseline, latestExists, readErr := service.readManagedBaseline(ctx, baselineFileName)
	if readErr != nil {
		return BaselineAppendResult{}, readErr
	}
	if latestExists != baselineExists || latestExists && baselineDigest(latestBaseline) != baselineDigest(baseline) {
		return BaselineAppendResult{}, fmt.Errorf("%w: quality baseline changed before publication", ErrBaselineConflict)
	}

	baselineWrite, profileWrite, writeErr := service.writeManagedPair(ctx, result.Baseline, result.Profile, baselineFileName, profileFileName, baselineExists, baseline)
	if writeErr != nil {
		return BaselineAppendResult{}, writeErr
	}
	result.BaselineWrite = baselineWrite
	result.ProfileWrite = profileWrite
	result.Changed = true
	return result, nil
}

// DefaultBaselineID gives a first-time managed baseline a stable identity
// without forcing a second name on the common "main.json" workflow.
func DefaultBaselineID(fileName string) string {
	base := filepath.Base(strings.TrimSpace(fileName))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return "baseline:" + base
}

func (service *Service) readManagedBaseline(ctx context.Context, fileName string) (quality.Baseline, bool, error) {
	baseline, err := service.Store.ReadBaseline(ctx, fileName)
	if err == nil {
		if validateErr := quality.ValidateBaseline(baseline); validateErr != nil {
			return quality.Baseline{}, true, validateErr
		}
		return baseline, true, nil
	}
	if errors.Is(err, ErrBaselineNotFound) || os.IsNotExist(unwrapPolicyPathError(err)) {
		return quality.Baseline{}, false, nil
	}
	return quality.Baseline{}, false, err
}

func unwrapPolicyPathError(err error) error {
	for err != nil {
		if os.IsNotExist(err) {
			return err
		}
		unwrapped, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		err = unwrapped.Unwrap()
	}
	return err
}

func baselineDigest(value quality.Baseline) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func (service *Service) writeManagedPair(ctx context.Context, baseline quality.Baseline, profile quality.QualityProfile, baselineFileName, profileFileName string, hadBaseline bool, previousBaseline quality.Baseline) (WriteResult, WriteResult, error) {
	baselineWrite, err := service.Store.SaveBaseline(ctx, baseline, baselineFileName, true)
	if err != nil {
		return WriteResult{}, WriteResult{}, err
	}
	profileWrite, err := service.Store.SaveProfile(ctx, profile, profileFileName, true)
	if err == nil {
		return baselineWrite, profileWrite, nil
	}
	// Restore the baseline document so a failed profile publication never
	// leaves a new baseline revision without its profile reference.
	if hadBaseline {
		_, _ = service.Store.SaveBaseline(ctx, previousBaseline, baselineFileName, true)
	} else {
		if directory, directoryErr := service.Store.directory(BaselineDirectoryName, false); directoryErr == nil && directory != "" {
			if path, pathErr := safeDocumentPath(directory, baselineFileName); pathErr == nil {
				_ = os.Remove(path)
			}
		}
	}
	return WriteResult{}, WriteResult{}, fmt.Errorf("profile publication failed after baseline publication: %w", err)
}
