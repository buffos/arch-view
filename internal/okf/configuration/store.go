package configuration

import (
	"bytes"
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
	"sync"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/projectdocument"
)

const (
	SectionSchemaVersion = "arch-view.okf/v1"
	configFileName       = ".archview.json"
	maxConfigurationSize = 4 << 20
)

type Store struct {
	mu          sync.Mutex
	idempotency map[operationKey]idempotentResult
}

type operationKey struct {
	projectRoot string
	operationID string
}

type idempotentResult struct {
	input []byte
	value domain.ProjectConfiguration
}

func NewStore() *Store {
	return &Store{idempotency: make(map[operationKey]idempotentResult)}
}

func (store *Store) Load(ctx context.Context, projectRoot string) (domain.ProjectConfiguration, error) {
	if err := contextErr(ctx); err != nil {
		return domain.ProjectConfiguration{}, err
	}
	pathValue, data, found, err := nearestConfiguration(projectRoot)
	if err != nil {
		return domain.ProjectConfiguration{}, domain.WrapError("okf_configuration_invalid", 500, "project configuration could not be read", err)
	}
	if !found {
		return domain.ProjectConfiguration{SchemaVersion: SectionSchemaVersion, Revision: ""}, nil
	}
	value, err := decodeConfiguration(data)
	if err != nil {
		return domain.ProjectConfiguration{}, domain.WrapError("okf_configuration_invalid", 422, "project configuration is invalid", err)
	}
	value.Revision = revision(data)
	_ = pathValue
	return value, nil
}

func (store *Store) Save(ctx context.Context, projectRoot string, value domain.ProjectConfiguration, expectedRevision, operationID string, input []byte) (domain.ProjectConfiguration, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := contextErr(ctx); err != nil {
		return domain.ProjectConfiguration{}, err
	}
	operationID = strings.TrimSpace(operationID)
	absoluteRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return domain.ProjectConfiguration{}, domain.WrapError("okf_configuration_invalid", 400, "project root could not be resolved", err)
	}
	key := operationKey{projectRoot: filepath.Clean(absoluteRoot), operationID: operationID}
	if operationID != "" {
		if previous, exists := store.idempotency[key]; exists {
			if bytes.Equal(previous.input, input) {
				return domain.CloneConfiguration(previous.value), nil
			}
			return domain.ProjectConfiguration{}, domain.NewError("okf_idempotency_conflict", 409, "operation ID was already used with different input", map[string]any{"operation_id": operationID})
		}
	}
	pathValue, existingData, found, err := nearestConfiguration(projectRoot)
	if err != nil {
		return domain.ProjectConfiguration{}, domain.WrapError("okf_configuration_invalid", 500, "project configuration could not be read", err)
	}
	if !found {
		pathValue = filepath.Join(projectRoot, configFileName)
		existingData = []byte(`{"schema_version":"arch-view.config/v1","layout":{"algorithm":"layered","options":{}}}`)
		if expectedRevision == "" {
			expectedRevision = revision(existingData)
		}
	}
	encoded, err := projectdocument.Update(pathValue, existingData, func(current []byte) ([]byte, error) {
		if err := contextErr(ctx); err != nil {
			return nil, err
		}
		currentRevision := revision(current)
		if expectedRevision != currentRevision {
			return nil, domain.NewError("okf_revision_conflict", 409, "project configuration changed since it was read", map[string]any{"expected_revision": expectedRevision, "actual_revision": currentRevision})
		}
		if _, err := decodeConfiguration(current); err != nil {
			return nil, domain.WrapError("okf_configuration_invalid", 422, "project configuration is invalid", err)
		}
		return encodeConfiguration(current, value)
	})
	if err != nil {
		var domainErr *domain.Error
		if errors.As(err, &domainErr) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return domain.ProjectConfiguration{}, err
		}
		return domain.ProjectConfiguration{}, domain.WrapError("okf_configuration_write_failed", 500, "project configuration could not be updated", err)
	}
	value.SchemaVersion = SectionSchemaVersion
	value.Revision = revision(encoded)
	if operationID != "" {
		store.idempotency[key] = idempotentResult{input: append([]byte(nil), input...), value: domain.CloneConfiguration(value)}
	}
	return value, nil
}

func encodeConfiguration(existingData []byte, value domain.ProjectConfiguration) ([]byte, error) {
	raw, err := projectdocument.Object(existingData)
	if err != nil {
		return nil, domain.WrapError("okf_configuration_invalid", 422, "project configuration is not valid JSON", err)
	}
	sectionValues := sectionValue(value)
	for _, profile := range value.Profiles {
		if len(profile.Layout.Features) > 0 {
			projectdocument.RequireV2(raw)
			break
		}
	}
	section, err := json.Marshal(sectionValues)
	if err != nil {
		return nil, domain.WrapError("okf_configuration_invalid", 400, "OKF configuration could not be encoded", err)
	}
	var encodedSection map[string]json.RawMessage
	if err := decodeRaw(section, &encodedSection); err != nil {
		return nil, domain.WrapError("okf_configuration_invalid", 400, "OKF configuration could not be encoded", err)
	}
	if existingSection, exists := raw["okf"]; exists {
		var previous map[string]json.RawMessage
		if err := decodeRaw(existingSection, &previous); err == nil {
			for key, item := range previous {
				if !isKnownSectionKey(key) {
					encodedSection[key] = item
				}
			}
		}
	}
	section, err = json.Marshal(encodedSection)
	if err != nil {
		return nil, domain.WrapError("okf_configuration_invalid", 400, "OKF configuration could not be encoded", err)
	}
	raw["okf"] = section
	encoded, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return nil, domain.WrapError("okf_configuration_write_failed", 500, "project configuration could not be encoded", err)
	}
	return append(encoded, '\n'), nil
}

func isKnownSectionKey(key string) bool {
	switch key {
	case "schema_version", "default_graph", "bindings", "profiles":
		return true
	default:
		return false
	}
}

func sectionValue(value domain.ProjectConfiguration) map[string]any {
	profiles := append([]domain.Profile(nil), value.Profiles...)
	sort.SliceStable(profiles, func(left, right int) bool { return profiles[left].ProfileID < profiles[right].ProfileID })
	bindings := append([]domain.ProfileBinding(nil), value.Bindings...)
	sort.SliceStable(bindings, func(left, right int) bool { return bindings[left].BundleID < bindings[right].BundleID })
	return map[string]any{"schema_version": SectionSchemaVersion, "default_graph": value.DefaultGraph, "bindings": bindings, "profiles": profiles}
}

func decodeConfiguration(data []byte) (domain.ProjectConfiguration, error) {
	raw, err := projectdocument.Object(data)
	if err != nil {
		return domain.ProjectConfiguration{}, err
	}
	value := domain.ProjectConfiguration{SchemaVersion: SectionSchemaVersion}
	sectionData, exists := raw["okf"]
	if !exists || bytes.Equal(bytes.TrimSpace(sectionData), []byte("null")) {
		return value, nil
	}
	var section struct {
		SchemaVersion string                  `json:"schema_version"`
		DefaultGraph  string                  `json:"default_graph"`
		Bindings      []domain.ProfileBinding `json:"bindings"`
		Profiles      []domain.Profile        `json:"profiles"`
	}
	if err := decodeRaw(sectionData, &section); err != nil {
		return value, err
	}
	if section.SchemaVersion != "" && section.SchemaVersion != SectionSchemaVersion {
		return value, fmt.Errorf("unsupported OKF configuration schema %q", section.SchemaVersion)
	}
	value.DefaultGraph = filepath.ToSlash(strings.TrimSpace(section.DefaultGraph))
	value.Bindings = append([]domain.ProfileBinding(nil), section.Bindings...)
	value.Profiles = append([]domain.Profile(nil), section.Profiles...)
	for index := range value.Profiles {
		value.Profiles[index].Origin = "project_local"
		value.Profiles[index].Immutable = false
		if value.Profiles[index].ProfileID == "" {
			return value, fmt.Errorf("profile %d has no profile_id", index)
		}
	}
	return value, nil
}

func nearestConfiguration(projectRoot string) (string, []byte, bool, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return "", nil, false, fmt.Errorf("project root is required")
	}
	absolute, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", nil, false, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", nil, false, err
	}
	if !info.IsDir() {
		return "", nil, false, fmt.Errorf("project root is not a directory")
	}
	directory := filepath.Clean(absolute)
	for {
		pathValue := filepath.Join(directory, configFileName)
		data, readErr := projectdocument.Read(pathValue)
		if readErr == nil {
			if len(data) > maxConfigurationSize {
				return "", nil, false, fmt.Errorf("configuration exceeds %d bytes", maxConfigurationSize)
			}
			return pathValue, data, true, nil
		}
		if !os.IsNotExist(readErr) {
			return pathValue, nil, false, readErr
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return "", nil, false, nil
}

func decodeRaw(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("more than one JSON value")
		}
		return err
	}
	return nil
}

func revision(data []byte) string {
	hash := sha256.Sum256(bytes.TrimSpace(data))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
