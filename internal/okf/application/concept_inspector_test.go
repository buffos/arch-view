package application

import (
	"context"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/profile"
)

func TestConceptInspectorUsesOnlySelectedIndexAndProfile(t *testing.T) {
	registry := profile.NewRegistry()
	inspector := conceptInspector{profiles: registry}
	index := domain.BundleIndex{
		BundleID: "selected", SourceRevision: "source-1", ConceptOrder: []string{"root"},
		Documents: map[string]domain.ConceptDocument{
			"root": {ConceptID: "root", Title: "Source title", Type: "topic", SourcePath: "root.md", Markdown: "**Safe** <script>alert(1)</script>"},
		},
	}
	for _, profileID := range []string{profile.DefaultProfileID, "project:missing"} {
		detail, err := inspector.inspect(context.Background(), index, profileID, "root")
		if err != nil {
			t.Fatal(err)
		}
		if detail.BundleID != "selected" || detail.SourceRevision != "source-1" || detail.Overview["title"] != "Source title" {
			t.Fatalf("detail lost source identity: %+v", detail)
		}
		if detail.RenderedMarkdown.Format != "sanitized_commonmark" {
			t.Fatalf("unexpected renderer: %+v", detail.RenderedMarkdown)
		}
		if strings.Contains(detail.RenderedMarkdown.Content, "<script") || !strings.Contains(detail.RenderedMarkdown.Content, "<strong>Safe</strong>") {
			t.Fatalf("detail bypassed the shared Markdown sanitizer: %s", detail.RenderedMarkdown.Content)
		}
	}
	if _, err := inspector.inspect(context.Background(), index, profile.DefaultProfileID, "foreign"); !hasApplicationCode(err, "okf_concept_not_found") {
		t.Fatalf("foreign concept accepted: %v", err)
	}
}

func TestConceptInspectorInvalidProfileFallsBackWithoutLosingDiagnostics(t *testing.T) {
	registry := profile.NewRegistry()
	registry.SetProjectProfiles([]domain.Profile{{
		ProfileID: "project:broken", Bases: []string{"project:missing-base"},
		Details: domain.DetailSettings{RawConfigured: true, ShowRawMarkdown: false},
	}})
	index := domain.BundleIndex{
		BundleID: "selected", ConceptOrder: []string{"root"},
		Documents: map[string]domain.ConceptDocument{
			"root": {ConceptID: "root", Type: "topic", Markdown: "Source body"},
		},
	}
	detail, err := (conceptInspector{profiles: registry}).inspect(context.Background(), index, "project:broken", "root")
	if err != nil {
		t.Fatal(err)
	}
	if detail.RawMarkdown != "Source body" {
		t.Fatal("invalid profile settings leaked through Neutral fallback")
	}
	for _, diagnostic := range detail.Diagnostics {
		if diagnostic.Code == "okf_profile_not_found" && diagnostic.ProfileID == "project:missing-base" && diagnostic.Recovery != "" {
			return
		}
	}
	t.Fatalf("fallback lost the profile repair diagnostic: %+v", detail.Diagnostics)
}
