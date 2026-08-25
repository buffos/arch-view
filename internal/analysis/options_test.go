package analysis

import "testing"

func TestResolveOptionsUsesCLIBeforeProjectBeforeDefaults(t *testing.T) {
	manifest := Manifest{
		ID:         "org.example.options",
		Version:    "1.0.0",
		Language:   "go",
		APIVersion: AnalyzerAPIVersion,
		DetectionMarkers: []DetectionMarker{{
			Kind: "file", Value: "go.mod", Weight: 1,
		}},
		Options: []OptionDescriptor{
			{Name: "include_tests", Type: "boolean", Default: false},
			{Name: "build_tags", Type: "string[]", Default: []string{}},
			{Name: "module", Type: "string", Default: nil},
		},
	}
	effective, err := ResolveOptions(
		manifest,
		map[string]any{"include_tests": true, "build_tags": []string{"z", "a"}, "module": "project"},
		map[string]any{"include_tests": false, "module": "cli"},
	)
	if err != nil {
		t.Fatalf("resolve options: %v", err)
	}
	if got := effective.Values["include_tests"]; got != false {
		t.Fatalf("include_tests = %#v, want false", got)
	}
	if got := effective.Sources["include_tests"]; got != "cli" {
		t.Fatalf("include_tests source = %q, want cli", got)
	}
	if got := effective.Values["module"]; got != "cli" {
		t.Fatalf("module = %#v, want cli", got)
	}
	tags, ok := effective.Values["build_tags"].([]string)
	if !ok || tags == nil || len(tags) != 2 || tags[0] != "a" || tags[1] != "z" {
		t.Fatalf("build_tags = %#v, want sorted [a z]", effective.Values["build_tags"])
	}
	if effective.Fingerprint == "" {
		t.Fatal("effective options fingerprint is empty")
	}
}

func TestResolveOptionsKeepsEmptyStringArraysAsArrays(t *testing.T) {
	manifest := validManifest("org.example.options-array", "go")
	manifest.Options = append(manifest.Options, OptionDescriptor{Name: "exclude", Type: "string[]", Default: []string{}})
	effective, err := ResolveOptions(manifest, nil, nil)
	if err != nil {
		t.Fatalf("resolve options: %v", err)
	}
	values, ok := effective.Values["exclude"].([]string)
	if !ok || values == nil || len(values) != 0 {
		t.Fatalf("exclude = %#v, want non-nil empty string slice", effective.Values["exclude"])
	}
}

func TestResolveOptionsRejectsUnknownAndWrongType(t *testing.T) {
	manifest := validManifest("org.example.options", "go")
	if _, err := ResolveOptions(manifest, map[string]any{"unknown": true}, nil); ErrorCodeOf(err) != ErrInvalidOptions {
		t.Fatalf("unknown option error code = %q, want %q", ErrorCodeOf(err), ErrInvalidOptions)
	}
	if _, err := ResolveOptions(manifest, map[string]any{"include_tests": "true"}, nil); ErrorCodeOf(err) != ErrInvalidOptions {
		t.Fatalf("wrong type error code = %q, want %q", ErrorCodeOf(err), ErrInvalidOptions)
	}
}
