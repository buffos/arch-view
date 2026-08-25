package analysis

import (
	"errors"
	"testing"
)

func TestRegistryListsCompatibleAnalyzersDeterministically(t *testing.T) {
	registry := NewRegistry()
	for _, analyzer := range []*fakeAnalyzer{
		{manifest: validManifest("org.example.zeta", "zeta")},
		{manifest: validManifest("org.example.alpha", "alpha")},
	} {
		if err := registry.Register(analyzer); err != nil {
			t.Fatalf("register analyzer: %v", err)
		}
	}
	manifests := registry.ListManifests()
	if got := manifests[0].ID; got != "org.example.alpha" {
		t.Fatalf("first manifest = %q, want alpha", got)
	}
	if got := manifests[1].ID; got != "org.example.zeta" {
		t.Fatalf("second manifest = %q, want zeta", got)
	}
	if err := registry.Register(&fakeAnalyzer{manifest: validManifest("org.example.alpha", "other")}); err == nil {
		t.Fatal("duplicate registration succeeded")
	} else if ErrorCodeOf(err) != ErrDuplicateAnalyzer {
		t.Fatalf("duplicate error code = %q, want %q", ErrorCodeOf(err), ErrDuplicateAnalyzer)
	}
}

func TestRegistryRejectsMalformedAndIncompatibleManifests(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Manifest)
		want ErrorCode
	}{
		{
			name: "incompatible api",
			edit: func(manifest *Manifest) { manifest.APIVersion = "arch-view.analyzer/v2" },
			want: ErrAPIIncompatible,
		},
		{
			name: "malformed api",
			edit: func(manifest *Manifest) { manifest.APIVersion = "not-an-api" },
			want: ErrInvalidManifest,
		},
		{
			name: "invalid option default",
			edit: func(manifest *Manifest) {
				manifest.Options[0].Default = "yes"
			},
			want: ErrInvalidManifest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := validManifest("org.example.test", "test")
			test.edit(&manifest)
			err := NewRegistry().Register(&fakeAnalyzer{manifest: manifest})
			if err == nil {
				t.Fatal("invalid manifest was registered")
			}
			if ErrorCodeOf(err) != test.want {
				t.Fatalf("error code = %q, want %q", ErrorCodeOf(err), test.want)
			}
		})
	}
}

func TestRegistryRejectsTypedNilAnalyzer(t *testing.T) {
	var analyzer *fakeAnalyzer
	if err := NewRegistry().Register(analyzer); ErrorCodeOf(err) != ErrInvalidManifest {
		t.Fatalf("typed nil error code = %q, want %q", ErrorCodeOf(err), ErrInvalidManifest)
	}
}

func TestValidateManifestReportsAPIErrorsAsHostErrors(t *testing.T) {
	err := ValidateManifest(Manifest{ID: "org.example.test", Version: "1.0.0", Language: "go", APIVersion: "arch-view.analyzer/v9", DetectionMarkers: []DetectionMarker{{Kind: "file", Value: "go.mod", Weight: 1}}})
	if err == nil {
		t.Fatal("expected incompatible API error")
	}
	var hostErr *HostError
	if !errors.As(err, &hostErr) {
		t.Fatalf("error type = %T, want *HostError", err)
	}
}
