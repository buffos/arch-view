package version

import "testing"

func TestCurrentUsesApplicationReleaseIdentity(t *testing.T) {
	info := Current()
	if info.Application != "arch-view" {
		t.Fatalf("application = %q, want arch-view", info.Application)
	}
	if info.Version != "0.2.0" {
		t.Fatalf("version = %q, want 0.2.0", info.Version)
	}
	if info.Commit != "unknown" {
		t.Fatalf("commit = %q, want unknown", info.Commit)
	}
	if info.BuildDate != "unknown" {
		t.Fatalf("build date = %q, want unknown", info.BuildDate)
	}
	if info.BuildID != "local" {
		t.Fatalf("build ID = %q, want local", info.BuildID)
	}
}

func TestInfoStringIsStableForHumanOutput(t *testing.T) {
	info := Info{
		Application: "arch-view",
		Version:     "0.2.0",
		Commit:      "abc123",
		BuildDate:   "2026-09-01T12:00:00Z",
		BuildID:     "ci-42",
	}
	if got, want := info.String(), "arch-view 0.2.0 (commit abc123, built 2026-09-01T12:00:00Z, build ci-42)"; got != want {
		t.Fatalf("info string = %q, want %q", got, want)
	}
}

func TestLinkerFlagsExposeAllBuildMetadata(t *testing.T) {
	info := Info{
		Application: "arch-view",
		Version:     "0.2.0",
		Commit:      "abc123",
		BuildDate:   "2026-09-01T12:00:00Z",
		BuildID:     "ci-42",
	}
	if got, want := info.LinkerFlags(), "-X=github.com/buffo/arch-view/internal/version.Version=0.2.0 -X=github.com/buffo/arch-view/internal/version.Commit=abc123 -X=github.com/buffo/arch-view/internal/version.BuildDate=2026-09-01T12:00:00Z -X=github.com/buffo/arch-view/internal/version.BuildID=ci-42"; got != want {
		t.Fatalf("linker flags = %q, want %q", got, want)
	}
}
