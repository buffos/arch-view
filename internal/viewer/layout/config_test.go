package layout

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func TestParentLayoutOptionTrancheHasPinnedMetadata(t *testing.T) {
	zero := 0.0
	tests := []struct {
		id               string
		typeName         string
		targets          []string
		algorithms       []string
		defaultValue     any
		allowedValues    []any
		minimum          *float64
		minimumExclusive bool
	}{
		{
			id:               "org.eclipse.elk.aspectRatio",
			typeName:         "DOUBLE",
			targets:          []string{"PARENTS"},
			algorithms:       []string{"box", "random", "layered", "mrtree", "force", "rectpacking"},
			defaultValue:     "engine default",
			allowedValues:    []any{},
			minimum:          &zero,
			minimumExclusive: true,
		},
		{
			id:            "org.eclipse.elk.layered.spacing.baseValue",
			typeName:      "DOUBLE",
			targets:       []string{"PARENTS"},
			algorithms:    []string{"layered"},
			defaultValue:  "engine default",
			allowedValues: []any{},
			minimum:       &zero,
		},
		{
			id:            "org.eclipse.elk.layered.spacing.edgeEdgeBetweenLayers",
			typeName:      "DOUBLE",
			targets:       []string{"PARENTS"},
			algorithms:    []string{"layered"},
			defaultValue:  10.0,
			allowedValues: []any{},
			minimum:       &zero,
		},
		{
			id:           "org.eclipse.elk.layered.layering.strategy",
			typeName:     "ENUM",
			targets:      []string{"PARENTS"},
			algorithms:   []string{"layered"},
			defaultValue: "NETWORK_SIMPLEX",
			allowedValues: []any{
				"NETWORK_SIMPLEX", "LONGEST_PATH", "LONGEST_PATH_SOURCE", "COFFMAN_GRAHAM", "INTERACTIVE", "STRETCH_WIDTH", "MIN_WIDTH", "BF_MODEL_ORDER", "DF_MODEL_ORDER",
			},
		},
		{
			id:           "org.eclipse.elk.layered.cycleBreaking.strategy",
			typeName:     "ENUM",
			targets:      []string{"PARENTS"},
			algorithms:   []string{"layered"},
			defaultValue: "GREEDY",
			allowedValues: []any{
				"GREEDY", "DEPTH_FIRST", "INTERACTIVE", "MODEL_ORDER", "GREEDY_MODEL_ORDER", "SCC_CONNECTIVITY", "SCC_NODE_TYPE", "DFS_NODE_ORDER", "BFS_NODE_ORDER",
			},
		},
		{
			id:           "org.eclipse.elk.layered.crossingMinimization.strategy",
			typeName:     "ENUM",
			targets:      []string{"PARENTS"},
			algorithms:   []string{"layered"},
			defaultValue: "LAYER_SWEEP",
			allowedValues: []any{
				"LAYER_SWEEP", "MEDIAN_LAYER_SWEEP", "INTERACTIVE", "NONE",
			},
		},
		{
			id:           "org.eclipse.elk.layered.nodePlacement.strategy",
			typeName:     "ENUM",
			targets:      []string{"PARENTS"},
			algorithms:   []string{"layered"},
			defaultValue: "BRANDES_KOEPF",
			allowedValues: []any{
				"SIMPLE", "INTERACTIVE", "LINEAR_SEGMENTS", "BRANDES_KOEPF", "NETWORK_SIMPLEX",
			},
		},
		{
			id:            "org.eclipse.elk.layered.compaction.connectedComponents",
			typeName:      "BOOLEAN",
			targets:       []string{"PARENTS"},
			algorithms:    []string{"layered"},
			defaultValue:  false,
			allowedValues: []any{},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.id, func(t *testing.T) {
			option, ok := layoutOptionByID(testCase.id)
			if !ok {
				t.Fatalf("option is not in the pinned catalog")
			}
			if !option.Editable || option.RendererSupport != "supported" || !layoutOptionTargetsParent(option) {
				t.Fatalf("option is not an editable parent option: %#v", option)
			}
			if option.Type != testCase.typeName || !reflect.DeepEqual(option.Targets, testCase.targets) || option.Description == "" {
				t.Fatalf("type/targets/description = %q/%#v/%q, want %q/%#v/non-empty", option.Type, option.Targets, option.Description, testCase.typeName, testCase.targets)
			}
			if !reflect.DeepEqual(option.Algorithms, testCase.algorithms) {
				t.Fatalf("algorithms = %#v, want %#v", option.Algorithms, testCase.algorithms)
			}
			if !reflect.DeepEqual(option.DefaultValue, testCase.defaultValue) {
				t.Fatalf("default = %#v, want %#v", option.DefaultValue, testCase.defaultValue)
			}
			if !reflect.DeepEqual(option.AllowedValues, testCase.allowedValues) {
				t.Fatalf("allowed values = %#v, want %#v", option.AllowedValues, testCase.allowedValues)
			}
			if (option.Minimum == nil) != (testCase.minimum == nil) || option.Minimum != nil && *option.Minimum != *testCase.minimum {
				t.Fatalf("minimum = %#v, want %#v", option.Minimum, testCase.minimum)
			}
			if option.MinimumExclusive != testCase.minimumExclusive {
				t.Fatalf("minimum exclusive = %t, want %t", option.MinimumExclusive, testCase.minimumExclusive)
			}
		})
	}

	alignment, ok := layoutOptionByID("org.eclipse.elk.alignment")
	if !ok || alignment.Editable || alignment.RendererSupport == "supported" || reflect.DeepEqual(alignment.Targets, []string{"PARENTS"}) {
		t.Fatalf("node-targeted alignment was incorrectly exposed as a root option: %#v", alignment)
	}
	if _, ok := layoutOptionByID("org.eclipse.elk.spacing.baseValue"); ok {
		t.Fatal("non-canonical spacing base option unexpectedly exists")
	}
}

func TestParentLayoutOptionTrancheAcceptsPinnedValuesAndRejectsUnsafeValues(t *testing.T) {
	valid := map[string]any{
		"org.eclipse.elk.aspectRatio":                            1.6,
		"org.eclipse.elk.layered.spacing.baseValue":              0,
		"org.eclipse.elk.layered.spacing.edgeEdgeBetweenLayers":  10.0,
		"org.eclipse.elk.layered.layering.strategy":              "NETWORK_SIMPLEX",
		"org.eclipse.elk.layered.cycleBreaking.strategy":         "GREEDY",
		"org.eclipse.elk.layered.crossingMinimization.strategy":  "LAYER_SWEEP",
		"org.eclipse.elk.layered.nodePlacement.strategy":         "BRANDES_KOEPF",
		"org.eclipse.elk.layered.compaction.connectedComponents": true,
	}
	if _, err := validateLayoutProfile(LayoutProfile{Algorithm: "layered", Options: valid}); err != nil {
		t.Fatalf("valid parent tranche rejected: %v", err)
	}

	cases := []struct {
		name    string
		profile LayoutProfile
		code    analysis.ErrorCode
	}{
		{name: "aspect ratio lower bound is exclusive", profile: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.aspectRatio": 0.0}}, code: analysis.ErrInvalidOptions},
		{name: "base spacing lower bound", profile: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.layered.spacing.baseValue": -1.0}}, code: analysis.ErrInvalidOptions},
		{name: "edge spacing lower bound", profile: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.layered.spacing.edgeEdgeBetweenLayers": -1.0}}, code: analysis.ErrInvalidOptions},
		{name: "bad layering enum", profile: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.layered.layering.strategy": "NOT_A_STRATEGY"}}, code: analysis.ErrInvalidOptions},
		{name: "algorithm incompatibility", profile: LayoutProfile{Algorithm: "force", Options: map[string]any{"org.eclipse.elk.layered.nodePlacement.strategy": "SIMPLE"}}, code: analysis.ErrInvalidOptions},
		{name: "node-targeted option stays unsupported", profile: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.alignment": "CENTER"}}, code: analysis.ErrUnsupportedOption},
		{name: "incorrect option identifier", profile: LayoutProfile{Algorithm: "layered", Options: map[string]any{"org.eclipse.elk.spacing.baseValue": 10.0}}, code: analysis.ErrUnsupportedOption},
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

func TestLayoutProfileValidationPreservesCallerAndCanonicalizesAliases(t *testing.T) {
	profile := LayoutProfile{Algorithm: "layered", Options: map[string]any{
		"elk.layered.layering.strategy": "NETWORK_SIMPLEX",
	}}
	validated, err := validateLayoutProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := profile.Options["elk.layered.layering.strategy"]; !ok || len(profile.Options) != 1 {
		t.Fatalf("caller profile was mutated: %#v", profile)
	}
	if validated.Options["org.eclipse.elk.layered.layering.strategy"] != "NETWORK_SIMPLEX" {
		t.Fatalf("validated profile did not canonicalize option: %#v", validated)
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
