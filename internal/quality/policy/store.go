// Package policy owns the file-backed quality policy boundary.
//
// The package deliberately knows only about quality profiles and baselines.
// It does not know about live sessions, analyzers, viewers, or transports.
// Callers must perform permission and report-compatibility checks before
// invoking a write.
package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/quality"
)

const (
	ProfileDirectoryName  = "quality-profiles"
	BaselineDirectoryName = "quality-baselines"
	MaxProfileBytes       = 1 << 20
	MaxBaselineBytes      = 8 << 20
)

var (
	ErrDestinationConflict = errors.New("quality policy destination already exists")
	ErrInvalidDestination  = errors.New("quality policy destination is invalid")
	ErrBaselineNotFound    = errors.New("quality baseline was not found")
	ErrBaselineConflict    = errors.New("quality baseline resolution is ambiguous")
	ErrBaselineInvalid     = errors.New("quality baseline is invalid")
)

// ProfileInfo is the safe, project-relative identity returned by discovery.
type ProfileInfo struct {
	ProfileID      string `json:"profile_id"`
	ProfileVersion string `json:"profile_version"`
	FileName       string `json:"file_name"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
}

// WriteResult identifies a successful policy write without exposing an
// absolute filesystem path to a transport.
type WriteResult struct {
	FileName     string                `json:"file_name"`
	RelativePath string                `json:"relative_path"`
	Digest       quality.ContentDigest `json:"digest"`
	Overwritten  bool                  `json:"overwritten"`
}

// BaselineInfo is the bounded, project-relative metadata returned by
// baseline discovery. Invalid documents remain visible to read-only callers
// without becoming candidates for suppression.
type BaselineInfo struct {
	BaselineID string   `json:"baseline_id,omitempty"`
	Revision   string   `json:"revision,omitempty"`
	FileName   string   `json:"file_name"`
	Status     string   `json:"status"`
	EntryCount int      `json:"entry_count"`
	Profiles   []string `json:"profiles"`
	Reason     string   `json:"reason,omitempty"`
}

// FileStore restricts policy documents to the two configured project
// directories. The root is canonicalized once, so later symlink changes do
// not turn a relative policy destination into an arbitrary write target.
type FileStore struct {
	root string
}

func NewFileStore(root string) (*FileStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("policy root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("policy root could not be normalized: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("policy root must be an existing directory")
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("policy root symlinks could not be resolved: %w", err)
	}
	return &FileStore{root: filepath.Clean(resolved)}, nil
}

func (store *FileStore) Root() string {
	if store == nil {
		return ""
	}
	return store.root
}

func (store *FileStore) ListProfiles(ctx context.Context) ([]ProfileInfo, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	directory, err := store.directory(ProfileDirectoryName, false)
	if err != nil {
		return nil, err
	}
	if directory == "" {
		return []ProfileInfo{}, nil
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("quality profile directory could not be read: %w", err)
	}
	result := make([]ProfileInfo, 0, len(entries))
	seen := make(map[string]int)
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		info := ProfileInfo{FileName: entry.Name(), Status: "invalid"}
		profile, readErr := readJSONProfile(filepath.Join(directory, entry.Name()))
		if readErr != nil {
			info.Reason = readErr.Error()
			result = append(result, info)
			continue
		}
		info.ProfileID = profile.ProfileID
		info.ProfileVersion = profile.ProfileVersion
		if _, ok := seen[profileKey(profile.ProfileID, profile.ProfileVersion)]; ok {
			info.Reason = "another profile uses the same profile id and version"
			result = append(result, info)
			continue
		}
		seen[profileKey(profile.ProfileID, profile.ProfileVersion)] = len(result)
		info.Status = "available"
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ProfileID != result[j].ProfileID {
			return result[i].ProfileID < result[j].ProfileID
		}
		if result[i].ProfileVersion != result[j].ProfileVersion {
			return result[i].ProfileVersion < result[j].ProfileVersion
		}
		return result[i].FileName < result[j].FileName
	})
	return result, nil
}

func (store *FileStore) ResolveProfile(ctx context.Context, id, version string) (quality.QualityProfile, error) {
	profile, _, err := store.resolveProfileDocument(ctx, id, version)
	return profile, err
}

func (store *FileStore) ProfileDocument(ctx context.Context, id, version string) (quality.QualityProfile, ProfileInfo, error) {
	return store.resolveProfileDocument(ctx, id, version)
}

func (store *FileStore) resolveProfileDocument(ctx context.Context, id, version string) (quality.QualityProfile, ProfileInfo, error) {
	if err := contextError(ctx); err != nil {
		return quality.QualityProfile{}, ProfileInfo{}, err
	}
	directory, err := store.directory(ProfileDirectoryName, false)
	if err != nil {
		return quality.QualityProfile{}, ProfileInfo{}, err
	}
	if directory == "" {
		return quality.QualityProfile{}, ProfileInfo{}, fmt.Errorf("quality profile %s@%s was not found", id, version)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return quality.QualityProfile{}, ProfileInfo{}, fmt.Errorf("quality profile directory could not be read: %w", err)
	}
	var selected *quality.QualityProfile
	var selectedInfo ProfileInfo
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		profile, readErr := readJSONProfile(filepath.Join(directory, entry.Name()))
		if readErr != nil || profile.ProfileID != id || profile.ProfileVersion != version {
			continue
		}
		if selected != nil {
			return quality.QualityProfile{}, ProfileInfo{}, fmt.Errorf("quality profile %s@%s has conflicting documents", id, version)
		}
		copy := profile
		selected = &copy
		selectedInfo = ProfileInfo{ProfileID: id, ProfileVersion: version, FileName: entry.Name(), Status: "available"}
	}
	if selected == nil {
		return quality.QualityProfile{}, ProfileInfo{}, fmt.Errorf("quality profile %s@%s was not found", id, version)
	}
	return *selected, selectedInfo, nil
}

func (store *FileStore) SaveProfile(ctx context.Context, profile quality.QualityProfile, fileName string, overwrite bool) (WriteResult, error) {
	if err := contextError(ctx); err != nil {
		return WriteResult{}, err
	}
	return store.writeJSON(ProfileDirectoryName, fileName, profile, overwrite, MaxProfileBytes)
}

func (store *FileStore) SaveBaseline(ctx context.Context, baseline quality.Baseline, fileName string, overwrite bool) (WriteResult, error) {
	if err := contextError(ctx); err != nil {
		return WriteResult{}, err
	}
	return store.writeJSON(BaselineDirectoryName, fileName, baseline, overwrite, MaxBaselineBytes)
}

func (store *FileStore) ReadProfile(ctx context.Context, fileName string) (quality.QualityProfile, error) {
	if err := contextError(ctx); err != nil {
		return quality.QualityProfile{}, err
	}
	directory, err := store.directory(ProfileDirectoryName, false)
	if err != nil {
		return quality.QualityProfile{}, err
	}
	if directory == "" {
		return quality.QualityProfile{}, fmt.Errorf("quality profile was not found: %s", fileName)
	}
	path, err := safeDocumentPath(directory, fileName)
	if err != nil {
		return quality.QualityProfile{}, err
	}
	return readJSONProfile(path)
}

func (store *FileStore) ReadBaseline(ctx context.Context, fileName string) (quality.Baseline, error) {
	if err := contextError(ctx); err != nil {
		return quality.Baseline{}, err
	}
	directory, err := store.directory(BaselineDirectoryName, false)
	if err != nil {
		return quality.Baseline{}, err
	}
	if directory == "" {
		return quality.Baseline{}, fmt.Errorf("%w: %s", ErrBaselineNotFound, fileName)
	}
	path, err := safeDocumentPath(directory, fileName)
	if err != nil {
		return quality.Baseline{}, err
	}
	return readJSONBaseline(path)
}

// ListBaselines discovers project-local baseline documents without making any
// of them active. Invalid documents are returned as metadata so an agent can
// explain why a baseline was not usable.
func (store *FileStore) ListBaselines(ctx context.Context) ([]BaselineInfo, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	directory, err := store.directory(BaselineDirectoryName, false)
	if err != nil {
		return nil, err
	}
	if directory == "" {
		return []BaselineInfo{}, nil
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("quality baseline directory could not be read: %w", err)
	}
	result := make([]BaselineInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		info := BaselineInfo{FileName: entry.Name(), Status: "invalid", Profiles: []string{}}
		baseline, readErr := store.ReadBaseline(ctx, entry.Name())
		if readErr != nil {
			info.Reason = readErr.Error()
			result = append(result, info)
			continue
		}
		if validateErr := quality.ValidateBaseline(baseline); validateErr != nil {
			info.BaselineID = baseline.BaselineID
			info.Revision = baseline.Revision
			info.Reason = validateErr.Error()
			result = append(result, info)
			continue
		}
		info.BaselineID = baseline.BaselineID
		info.Revision = baseline.Revision
		info.Status = "available"
		info.EntryCount = len(baseline.Entries)
		seenProfiles := make(map[string]struct{})
		for _, entry := range baseline.Entries {
			profile := entry.ProfileID + "@" + entry.ProfileVersion
			if _, ok := seenProfiles[profile]; ok {
				continue
			}
			seenProfiles[profile] = struct{}{}
			info.Profiles = append(info.Profiles, profile)
		}
		sort.Strings(info.Profiles)
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].BaselineID != result[j].BaselineID {
			return result[i].BaselineID < result[j].BaselineID
		}
		if result[i].Revision != result[j].Revision {
			return result[i].Revision < result[j].Revision
		}
		return result[i].FileName < result[j].FileName
	})
	return result, nil
}

// ResolveBaseline returns the unique valid baseline matching an explicit
// profile reference. Multiple matching files are rejected rather than
// selecting one nondeterministically.
func (store *FileStore) ResolveBaseline(ctx context.Context, baselineID, revision string) (quality.Baseline, BaselineInfo, error) {
	baselineID = strings.TrimSpace(baselineID)
	revision = strings.TrimSpace(revision)
	if baselineID == "" {
		return quality.Baseline{}, BaselineInfo{}, fmt.Errorf("%w: baseline ID is required", ErrBaselineNotFound)
	}
	infos, err := store.ListBaselines(ctx)
	if err != nil {
		return quality.Baseline{}, BaselineInfo{}, err
	}
	var matches []BaselineInfo
	var invalidMatches []BaselineInfo
	for _, info := range infos {
		if info.BaselineID != baselineID || revision != "" && info.Revision != revision {
			// A syntactically broken managed document cannot expose its own
			// identity. When its filename carries the default managed identity,
			// treat it as an invalid match rather than downgrading the profile
			// reference to a misleading "missing" warning.
			if info.Status != "available" && info.BaselineID == "" && DefaultBaselineID(info.FileName) == baselineID {
				invalidMatches = append(invalidMatches, info)
			}
			continue
		}
		if info.Status != "available" {
			invalidMatches = append(invalidMatches, info)
			continue
		}
		matches = append(matches, info)
	}
	if len(matches) == 0 {
		if len(invalidMatches) > 0 {
			return quality.Baseline{}, BaselineInfo{}, fmt.Errorf("%w: %s", ErrBaselineInvalid, invalidBaselineFiles(invalidMatches))
		}
		return quality.Baseline{}, BaselineInfo{}, fmt.Errorf("%w: %s@%s", ErrBaselineNotFound, baselineID, revision)
	}
	if len(invalidMatches) > 0 {
		return quality.Baseline{}, BaselineInfo{}, fmt.Errorf("%w: valid and invalid documents match %s@%s", ErrBaselineInvalid, baselineID, revision)
	}
	if len(matches) > 1 {
		files := make([]string, 0, len(matches))
		for _, match := range matches {
			files = append(files, match.FileName)
		}
		return quality.Baseline{}, BaselineInfo{}, fmt.Errorf("%w: %s", ErrBaselineConflict, strings.Join(files, ", "))
	}
	baseline, err := store.ReadBaseline(ctx, matches[0].FileName)
	if err != nil {
		return quality.Baseline{}, BaselineInfo{}, err
	}
	if err := quality.ValidateBaseline(baseline); err != nil {
		return quality.Baseline{}, BaselineInfo{}, err
	}
	return baseline, matches[0], nil
}

func invalidBaselineFiles(values []BaselineInfo) string {
	files := make([]string, 0, len(values))
	for _, value := range values {
		files = append(files, value.FileName)
	}
	sort.Strings(files)
	return strings.Join(files, ", ")
}

func (store *FileStore) writeJSON(directoryName, fileName string, value any, overwrite bool, maxBytes int) (WriteResult, error) {
	if store == nil || store.root == "" {
		return WriteResult{}, fmt.Errorf("policy store is not initialized")
	}
	if strings.TrimSpace(fileName) == "" {
		return WriteResult{}, fmt.Errorf("%w: file name must be one direct .json file", ErrInvalidDestination)
	}
	// Validate the user-controlled name before creating a policy directory. A
	// rejected request must not leave even an empty policy directory behind.
	if _, err := safeDocumentPath(store.root, fileName); err != nil {
		return WriteResult{}, err
	}
	directory, err := store.directory(directoryName, true)
	if err != nil {
		return WriteResult{}, err
	}
	path, err := safeDocumentPath(directory, fileName)
	if err != nil {
		return WriteResult{}, err
	}
	if info, statErr := os.Lstat(path); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return WriteResult{}, fmt.Errorf("%w: destination may not be a symlink", ErrInvalidDestination)
		}
		if !overwrite {
			return WriteResult{}, fmt.Errorf("%w: %s", ErrDestinationConflict, fileName)
		}
	} else if !os.IsNotExist(statErr) {
		return WriteResult{}, fmt.Errorf("destination could not be inspected: %w", statErr)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return WriteResult{}, fmt.Errorf("policy document could not be encoded: %w", err)
	}
	data = append(data, '\n')
	if len(data) > maxBytes {
		return WriteResult{}, fmt.Errorf("policy document exceeds the %d byte limit", maxBytes)
	}
	// Write beside the destination and publish only after the complete JSON
	// document has been flushed. A direct O_TRUNC write could leave an
	// existing policy document damaged when a write, sync, or close fails.
	temp, err := os.CreateTemp(directory, ".arch-view-policy-*")
	if err != nil {
		return WriteResult{}, fmt.Errorf("policy document could not be staged: %w", err)
	}
	tempPath := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(0o644); err != nil {
		_ = temp.Close()
		return WriteResult{}, fmt.Errorf("policy document permissions could not be set: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return WriteResult{}, fmt.Errorf("policy document could not be written: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return WriteResult{}, fmt.Errorf("policy document could not be flushed: %w", err)
	}
	if err := temp.Close(); err != nil {
		return WriteResult{}, fmt.Errorf("policy document could not be closed: %w", err)
	}
	if overwrite {
		if info, statErr := os.Lstat(path); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
			return WriteResult{}, fmt.Errorf("%w: destination may not be a symlink", ErrInvalidDestination)
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return WriteResult{}, fmt.Errorf("destination could not be inspected: %w", statErr)
		}
		if err := os.Rename(tempPath, path); err != nil {
			if os.IsExist(err) {
				return WriteResult{}, fmt.Errorf("%w: %s", ErrDestinationConflict, fileName)
			}
			return WriteResult{}, fmt.Errorf("policy document could not be published: %w", err)
		}
	} else {
		// A hard link publishes the staged file without replacing a file that
		// appeared after the initial conflict check. This keeps save-as and
		// first-time baseline creation non-overwriting on all supported hosts.
		if err := os.Link(tempPath, path); err != nil {
			if os.IsExist(err) {
				return WriteResult{}, fmt.Errorf("%w: %s", ErrDestinationConflict, fileName)
			}
			return WriteResult{}, fmt.Errorf("policy document could not be published: %w", err)
		}
		_ = os.Remove(tempPath)
	}
	removeTemp = false
	digest := sha256.Sum256(data)
	return WriteResult{
		FileName:     fileName,
		RelativePath: filepath.ToSlash(filepath.Join(directoryName, fileName)),
		Digest:       quality.ContentDigest{Algorithm: "hash:sha-256", Value: hex.EncodeToString(digest[:])},
		Overwritten:  overwrite,
	}, nil
}

func (store *FileStore) directory(name string, create bool) (string, error) {
	if store == nil || store.root == "" {
		return "", fmt.Errorf("policy store is not initialized")
	}
	candidate := filepath.Join(store.root, name)
	if info, err := os.Lstat(candidate); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("policy directory %s is not a normal directory", name)
		}
	} else if os.IsNotExist(err) {
		if !create {
			return "", nil
		}
		if err := os.MkdirAll(candidate, 0o755); err != nil {
			return "", fmt.Errorf("policy directory %s could not be created: %w", name, err)
		}
	} else {
		return "", fmt.Errorf("policy directory %s could not be inspected: %w", name, err)
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("policy directory %s could not be resolved: %w", name, err)
	}
	if !pathWithin(store.root, resolved) {
		return "", fmt.Errorf("policy directory %s escapes the project root", name)
	}
	return filepath.Clean(resolved), nil
}

func safeDocumentPath(directory, fileName string) (string, error) {
	fileName = strings.TrimSpace(fileName)
	if directory == "" || fileName == "" || fileName == "." || fileName == ".." ||
		strings.ContainsAny(fileName, `/\\`) || strings.Contains(fileName, "..") ||
		!strings.EqualFold(filepath.Ext(fileName), ".json") {
		return "", fmt.Errorf("%w: file name must be one direct .json file", ErrInvalidDestination)
	}
	return filepath.Join(directory, fileName), nil
}

// ValidateManagedBaselineFileName checks the public filename boundary used by
// live queries and managed baseline writes. Baseline callers may name one
// direct JSON document, but may not escape the quality-baselines directory.
func ValidateManagedBaselineFileName(fileName string) error {
	_, err := safeDocumentPath(".", fileName)
	return err
}

func readJSONProfile(path string) (quality.QualityProfile, error) {
	data, err := readLimited(path, MaxProfileBytes)
	if err != nil {
		return quality.QualityProfile{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var profile quality.QualityProfile
	if err := decoder.Decode(&profile); err != nil {
		return quality.QualityProfile{}, fmt.Errorf("profile JSON is invalid: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return quality.QualityProfile{}, fmt.Errorf("profile JSON contains trailing data")
	}
	return profile, nil
}

func readJSONBaseline(path string) (quality.Baseline, error) {
	data, err := readLimited(path, MaxBaselineBytes)
	if err != nil {
		return quality.Baseline{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var baseline quality.Baseline
	if err := decoder.Decode(&baseline); err != nil {
		return quality.Baseline{}, fmt.Errorf("baseline JSON is invalid: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return quality.Baseline{}, fmt.Errorf("baseline JSON contains trailing data")
	}
	return baseline, nil
}

func readLimited(path string, limit int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("policy document could not be read: %w", err)
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, fmt.Errorf("policy document could not be read: %w", err)
	}
	if len(data) > limit {
		return nil, fmt.Errorf("policy document exceeds the %d byte limit", limit)
	}
	return data, nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func profileKey(id, version string) string {
	return strings.TrimSpace(id) + "\x00" + strings.TrimSpace(version)
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

// Service is the validation seam used by live policy commands. Keeping this
// small makes it possible for transports to inject a memory-backed store in
// tests while production uses FileStore.
type Service struct {
	Store   *FileStore
	Catalog *quality.Catalog
}

func NewService(store *FileStore, catalog *quality.Catalog) *Service {
	return &Service{Store: store, Catalog: catalog}
}

func (service *Service) ValidateProfile(profile quality.QualityProfile) (quality.QualityProfile, error) {
	if service == nil || service.Catalog == nil {
		return quality.QualityProfile{}, fmt.Errorf("quality policy catalog is unavailable")
	}
	return quality.ValidateQualityProfile(profile, service.Catalog)
}

func (service *Service) ListProfiles(ctx context.Context) ([]ProfileInfo, error) {
	if service.Store == nil {
		return nil, fmt.Errorf("quality policy store is unavailable")
	}
	return service.Store.ListProfiles(ctx)
}

func (service *Service) ResolveProfile(ctx context.Context, id, version string) (quality.QualityProfile, error) {
	if service == nil || service.Store == nil {
		return quality.QualityProfile{}, fmt.Errorf("quality policy store is unavailable")
	}
	return service.Store.ResolveProfile(ctx, id, version)
}

func (service *Service) SaveProfile(ctx context.Context, profile quality.QualityProfile, fileName string, overwrite bool) (WriteResult, error) {
	if service == nil {
		return WriteResult{}, fmt.Errorf("quality policy service is unavailable")
	}
	validated, err := service.ValidateProfile(profile)
	if err != nil {
		return WriteResult{}, err
	}
	if service == nil || service.Store == nil {
		return WriteResult{}, fmt.Errorf("quality policy store is unavailable")
	}
	return service.Store.SaveProfile(ctx, validated, fileName, overwrite)
}

func (service *Service) SaveBaseline(ctx context.Context, baseline quality.Baseline, fileName string, overwrite bool) (WriteResult, error) {
	if service == nil {
		return WriteResult{}, fmt.Errorf("quality policy service is unavailable")
	}
	if service == nil || service.Store == nil {
		return WriteResult{}, fmt.Errorf("quality policy store is unavailable")
	}
	if err := quality.ValidateBaseline(baseline); err != nil {
		return WriteResult{}, err
	}
	return service.Store.SaveBaseline(ctx, baseline, fileName, overwrite)
}

func (service *Service) ListBaselines(ctx context.Context) ([]BaselineInfo, error) {
	if service == nil || service.Store == nil {
		return nil, fmt.Errorf("quality policy store is unavailable")
	}
	return service.Store.ListBaselines(ctx)
}

func (service *Service) ResolveBaseline(ctx context.Context, baselineID, revision string) (quality.Baseline, BaselineInfo, error) {
	if service == nil || service.Store == nil {
		return quality.Baseline{}, BaselineInfo{}, fmt.Errorf("quality policy store is unavailable")
	}
	return service.Store.ResolveBaseline(ctx, baselineID, revision)
}
