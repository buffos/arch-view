package viewer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestLayoutCatalogMatchesPinnedELKSurface(t *testing.T) {
	catalog := layoutCatalog()
	if len(catalog.Algorithms) != 11 {
		t.Fatalf("algorithms = %d, want 11", len(catalog.Algorithms))
	}
	if len(catalog.Categories) != 8 {
		t.Fatalf("categories = %d, want 8", len(catalog.Categories))
	}
	if len(catalog.Options) != 235 {
		t.Fatalf("options = %d, want 235", len(catalog.Options))
	}
	for _, option := range catalog.Options {
		if option.ID == "" || option.Name == "" || option.Type == "" || len(option.Targets) == 0 || option.Description == "" {
			t.Fatalf("incomplete catalog option = %#v", option)
		}
	}
	direction, ok := layoutOptionByID("org.eclipse.elk.direction")
	if !ok || !direction.Editable || direction.RendererSupport != "supported" || len(direction.AllowedValues) != 4 {
		t.Fatalf("direction metadata = %#v", direction)
	}
	unsupported, ok := layoutOptionByID("org.eclipse.elk.padding")
	if !ok || unsupported.Editable || unsupported.RendererSupport != "unsupported" {
		t.Fatalf("padding metadata = %#v", unsupported)
	}
}

func TestValidateLayoutProfileRejectsUnknownAndUnsafeOptions(t *testing.T) {
	valid := LayoutProfile{Algorithm: "layered", Options: map[string]any{
		"org.eclipse.elk.direction":                             "RIGHT",
		"org.eclipse.elk.edgeRouting":                           "ORTHOGONAL",
		"org.eclipse.elk.spacing.nodeNode":                      40.0,
		"org.eclipse.elk.layered.spacing.nodeNodeBetweenLayers": 90.0,
		"org.eclipse.elk.layered.thoroughness":                  8.0,
		"org.eclipse.elk.separateConnectedComponents":           true,
	}}
	if _, err := validateLayoutProfile(valid); err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}
	cases := []struct {
		name    string
		profile LayoutProfile
		code    analysis.ErrorCode
	}{
		{name: "unknown option", profile: LayoutProfile{Algorithm: "layered", Options: map[string]any{"elk.notReal": true}}, code: analysis.ErrUnsupportedOption},
		{name: "wrong type", profile: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.direction": 1.0}}, code: analysis.ErrInvalidOptions},
		{name: "bad enum", profile: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.edgeRouting": "curved"}}, code: analysis.ErrInvalidOptions},
		{name: "unsupported object", profile: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.padding": "10"}}, code: analysis.ErrUnsupportedOption},
		{name: "unknown algorithm", profile: LayoutProfile{Algorithm: "not-real", Options: map[string]any{}}, code: analysis.ErrUnsupportedOption},
		{name: "missing algorithm", profile: LayoutProfile{Options: map[string]any{}}, code: analysis.ErrInvalidOptions},
		{name: "blank algorithm", profile: LayoutProfile{Algorithm: "  ", Options: map[string]any{}}, code: analysis.ErrInvalidOptions},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := validateLayoutProfile(testCase.profile)
			if analysis.ErrorCodeOf(err) != testCase.code {
				t.Fatalf("error code = %q, want %q (%v)", analysis.ErrorCodeOf(err), testCase.code, err)
			}
		})
	}
}

func TestDecodeLayoutConfigRejectsMissingLayoutAlgorithm(t *testing.T) {
	for _, data := range []string{
		`{"schema_version":"arch-view.config/v1"}`,
		`{"schema_version":"arch-view.config/v1","layout":{"options":{}}}`,
	} {
		if _, err := decodeLayoutConfig([]byte(data)); analysis.ErrorCodeOf(err) != analysis.ErrInvalidOptions {
			t.Fatalf("decode error = %q, want %q for %s", analysis.ErrorCodeOf(err), analysis.ErrInvalidOptions, data)
		}
	}
}

func TestDiscoverLayoutSessionUsesNearestCompleteConfiguration(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "project", "nested")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	ancestorProfile := LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.direction": "LEFT"}}
	ancestorData, err := encodeLayoutConfig(ancestorProfile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, layoutConfigFileName), ancestorData, 0o644); err != nil {
		t.Fatal(err)
	}
	session := discoverLayoutSession(root)
	if session.origin != "ancestor" || !samePath(session.activePath, filepath.Join(parent, layoutConfigFileName)) || session.profile.Options["org.eclipse.elk.direction"] != "LEFT" {
		t.Fatalf("ancestor session = %#v", session)
	}
	projectProfile := LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.direction": "DOWN"}}
	projectData, err := encodeLayoutConfig(projectProfile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, layoutConfigFileName), projectData, 0o644); err != nil {
		t.Fatal(err)
	}
	session = discoverLayoutSession(root)
	if session.origin != "project" || !samePath(session.activePath, filepath.Join(root, layoutConfigFileName)) || session.profile.Options["org.eclipse.elk.direction"] != "DOWN" {
		t.Fatalf("project session = %#v", session)
	}
}

func TestDiscoverLayoutSessionDoesNotFallBackPastInvalidNearestFile(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "project")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	validData, err := encodeLayoutConfig(defaultLayoutProfile())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, layoutConfigFileName), validData, 0o644); err != nil {
		t.Fatal(err)
	}
	invalidPath := filepath.Join(root, layoutConfigFileName)
	if err := os.WriteFile(invalidPath, []byte(`{"schema_version":"arch-view.config/v1","layout":{"algorithm":"layered","options":{"elk.notReal":true}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	session := discoverLayoutSession(root)
	if session.status != "invalid" || session.origin != "project" || !samePath(session.activePath, invalidPath) || len(session.diagnostics) != 1 {
		t.Fatalf("invalid nearest session = %#v", session)
	}
	if session.profile.Algorithm != "layered" || strings.Contains(session.diagnostics[0].Message, "ancestor") {
		t.Fatalf("invalid fallback leaked into session = %#v", session)
	}
}

func TestLayoutConfigEncodingIsStableAndScoped(t *testing.T) {
	profile := LayoutProfile{Algorithm: "layered", Options: map[string]any{
		"org.eclipse.elk.spacing.nodeNode": 40.0,
		"org.eclipse.elk.direction":        "RIGHT",
	}}
	first, err := encodeLayoutConfig(profile)
	if err != nil {
		t.Fatal(err)
	}
	second, err := encodeLayoutConfig(profile)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("configuration encoding is not deterministic\nfirst=%s\nsecond=%s", first, second)
	}
	var decoded map[string]any
	if err := json.Unmarshal(first, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 2 {
		t.Fatalf("configuration has fields outside schema: %#v", decoded)
	}
	if _, err := encodeLayoutConfig(LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.padding": "bad"}}); err == nil {
		t.Fatal("unsafe option encoded successfully")
	}
}
