package live

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis/orchestration"
)

// FileSystemFingerprinter computes an authoritative fingerprint from file
// contents. It deliberately does not rely on watcher delivery or timestamps
// to decide whether a strict query is current.
type FileSystemFingerprinter struct{}

func (FileSystemFingerprinter) Fingerprint(ctx context.Context, repositoryRoot string, roots []WatchRoot) (InputFingerprint, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	root, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return InputFingerprint{}, newLiveError(ErrorSourceReconciliation, "repository root could not be normalized for fingerprinting", map[string]any{"error": err.Error()})
	}
	root = filepath.Clean(root)
	if len(roots) == 0 {
		roots = []WatchRoot{{Path: ".", Recursive: true}}
	}
	filesByPath := make(map[string]FileFingerprint)
	for _, watchRoot := range roots {
		if err := ctx.Err(); err != nil {
			return InputFingerprint{}, err
		}
		relativeRoot, err := normalizeRelativePath(watchRoot.Path, true)
		if err != nil {
			return InputFingerprint{}, newLiveError(ErrorWatchRootInvalid, "watch root is invalid during fingerprinting", map[string]any{"path": watchRoot.Path})
		}
		absoluteRoot := filepath.Clean(filepath.Join(root, filepath.FromSlash(relativeRoot)))
		if !pathWithin(root, absoluteRoot) {
			return InputFingerprint{}, newLiveError(ErrorWatchRootInvalid, "watch root escapes repository root during fingerprinting", map[string]any{"path": relativeRoot})
		}
		info, err := os.Stat(absoluteRoot)
		if err != nil {
			return InputFingerprint{}, newLiveError(ErrorSourceReconciliation, "watch root could not be read during fingerprinting", map[string]any{"path": relativeRoot, "error": err.Error()})
		}
		if !info.IsDir() {
			return InputFingerprint{}, newLiveError(ErrorWatchRootInvalid, "watch root is not a directory", map[string]any{"path": relativeRoot})
		}
		walkErr := filepath.WalkDir(absoluteRoot, func(filePath string, entry fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				if filePath != absoluteRoot && orchestration.IsDefaultExcludedDirectory(entry.Name()) {
					return filepath.SkipDir
				}
				if !watchRoot.Recursive && filePath != absoluteRoot {
					return filepath.SkipDir
				}
				return nil
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			fileHash, size, err := hashFile(ctx, filePath)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(root, filePath)
			if err != nil {
				return err
			}
			relative = normalizeFingerprintPath(relative)
			if relative == "" || relative == "." || !pathWithin(root, filePath) {
				return newLiveError(ErrorWatchRootInvalid, "fingerprinted file escaped repository root", map[string]any{"path": relative})
			}
			filesByPath[relative] = FileFingerprint{Path: relative, Size: size, Hash: fileHash}
			return nil
		})
		if walkErr != nil {
			if err := ctx.Err(); err != nil {
				return InputFingerprint{}, err
			}
			return InputFingerprint{}, newLiveError(ErrorSourceReconciliation, "authoritative source fingerprint failed", map[string]any{"root": relativeRoot, "error": walkErr.Error()})
		}
	}
	files := make([]FileFingerprint, 0, len(filesByPath))
	for _, file := range filesByPath {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	manifestParts := make([]string, 0, len(files))
	contentParts := make([]string, 0, len(files))
	for _, file := range files {
		manifestParts = append(manifestParts, fmt.Sprintf("%s\x00%d", file.Path, file.Size))
		contentParts = append(contentParts, file.Path+"\x00"+file.Hash.Value)
	}
	return InputFingerprint{
		ManifestFingerprint: digestStrings(manifestParts),
		ContentFingerprint:  digestStrings(contentParts),
		Files:               files,
	}, nil
}

func hashFile(ctx context.Context, path string) (ContentDigest, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return ContentDigest{}, 0, err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	buffer := make([]byte, 64*1024)
	var size int64
	for {
		if err := ctx.Err(); err != nil {
			return ContentDigest{}, 0, err
		}
		count, readErr := file.Read(buffer)
		if count > 0 {
			written, writeErr := hash.Write(buffer[:count])
			if writeErr != nil {
				return ContentDigest{}, 0, writeErr
			}
			size += int64(written)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return ContentDigest{}, 0, readErr
		}
	}
	return ContentDigest{Algorithm: "hash:sha-256", Value: hex.EncodeToString(hash.Sum(nil))}, size, nil
}

func normalizeFingerprintPath(value string) string {
	value = filepath.ToSlash(filepath.Clean(value))
	return strings.TrimPrefix(value, "./")
}

func digestStrings(values []string) ContentDigest {
	hash := sha256.New()
	for _, value := range values {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	return ContentDigest{Algorithm: "hash:sha-256", Value: hex.EncodeToString(hash.Sum(nil))}
}

func inputEqual(left, right InputFingerprint) bool {
	if left.ContentFingerprint.Value == "" && right.ContentFingerprint.Value == "" {
		return left.ManifestFingerprint.Value == right.ManifestFingerprint.Value
	}
	return left.ContentFingerprint.Value != "" && left.ContentFingerprint == right.ContentFingerprint
}

func changedFingerprintPaths(previous, current InputFingerprint) []string {
	left := make(map[string]FileFingerprint, len(previous.Files))
	right := make(map[string]FileFingerprint, len(current.Files))
	for _, file := range previous.Files {
		left[file.Path] = file
	}
	for _, file := range current.Files {
		right[file.Path] = file
	}
	seen := make(map[string]struct{}, len(left)+len(right))
	for path := range left {
		seen[path] = struct{}{}
	}
	for path := range right {
		seen[path] = struct{}{}
	}
	result := make([]string, 0)
	for path := range seen {
		if left[path].Hash != right[path].Hash || left[path].Size != right[path].Size {
			result = append(result, path)
		}
	}
	sort.Strings(result)
	return result
}

// StaticFingerprinter is useful for deterministic service tests and adapters
// that already possess an authoritative manifest from another subsystem.
type StaticFingerprinter struct {
	Value InputFingerprint
}

func (fingerprinter StaticFingerprinter) Fingerprint(context.Context, string, []WatchRoot) (InputFingerprint, error) {
	return cloneInputFingerprint(fingerprinter.Value), nil
}

func cloneInputFingerprint(value InputFingerprint) InputFingerprint {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var clone InputFingerprint
	if err := json.Unmarshal(data, &clone); err != nil {
		return value
	}
	return clone
}
