package clojureanalyzer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestStaticDependencyKindsResolutionMetadataAndEvidence(t *testing.T) {
	root := t.TempDir()
	writeClojureFile(t, root, "deps.edn", "{:paths [\"src\"]}\n")
	writeClojureFile(t, root, "src/app/a.clj", `(ns app.a
  (:require [app.b :as b]
            [app.b :refer [one two]]
            [clojure.string :as str]
            [app.missing :as missing]
            [app.macros :include-macros true]
            [app.macros :refer-macros [macro-other]])
  (:use [app.c :only [legacy]])
  (:require-macros [app.macros :refer [macro-fn]]))
`)
	writeClojureFile(t, root, "src/app/b.clj", "(ns app.b)\n")
	writeClojureFile(t, root, "src/app/c.clj", "(ns app.c)\n")
	writeClojureFile(t, root, "src/app/macros.clj", "(ns app.macros)\n")

	result := runClojureAnalysis(t, root, nil)
	if result.Status != analysis.StatusPartial {
		t.Fatalf("status = %s, diagnostics = %#v", result.Status, result.Diagnostics)
	}
	if len(result.Modules) != 4 {
		t.Fatalf("modules = %#v", result.Modules)
	}
	if len(result.Relationships) != 5 {
		t.Fatalf("relationships = %#v", result.Relationships)
	}
	if len(result.References) != 2 {
		t.Fatalf("references = %#v", result.References)
	}
	if !hasDiagnostic(result.Diagnostics, "clojure_unresolved_dependency") {
		t.Fatalf("unresolved diagnostic absent: %#v", result.Diagnostics)
	}

	require := findClojureRelationship(result.Relationships, "clj:app.a", "clj:app.b", "require")
	if require == nil || require.ToModuleID != "clj:app.b" || require.ToReferenceID != "" {
		t.Fatalf("require relationship = %#v", require)
	}
	if !metadataContainsString(require.Metadata, "aliases", "b") || !metadataContainsString(require.Metadata, "referred_symbols", "one") || len(require.SourceReferenceIDs) != 2 {
		t.Fatalf("merged require metadata = %#v", require)
	}
	if require.Confidence == nil || require.Confidence.Score != 1 {
		t.Fatalf("resolved confidence = %#v", require.Confidence)
	}

	use := findClojureRelationship(result.Relationships, "clj:app.a", "clj:app.c", "use")
	if use == nil || !metadataContainsString(use.Metadata, "referred_symbols", "legacy") {
		t.Fatalf("use relationship = %#v", use)
	}
	macro := findClojureRelationship(result.Relationships, "clj:app.a", "clj:app.macros", "macro")
	if macro == nil || !metadataContainsString(macro.Metadata, "referred_symbols", "macro-fn") || !metadataContainsString(macro.Metadata, "referred_symbols", "macro-other") {
		t.Fatalf("macro relationship = %#v", macro)
	}
	standard := findClojureRelationshipByScope(result.Relationships, result.References, "clj:app.a", "clojure.string", "standard_library")
	if standard == nil {
		t.Fatalf("standard-library relationship missing: %#v", result.Relationships)
	}
	unresolved := findClojureRelationshipByScope(result.Relationships, result.References, "clj:app.a", "app.missing", "unresolved")
	if unresolved == nil || unresolved.ToReferenceID == "" || unresolved.Confidence == nil || unresolved.Confidence.Score != 0.2 {
		t.Fatalf("unresolved relationship = %#v", unresolved)
	}
	for _, relationship := range result.Relationships {
		if len(relationship.SourceReferenceIDs) == 0 {
			t.Fatalf("relationship lacks evidence: %#v", relationship)
		}
	}
	result.RunID = "test-run"
	if err := analysis.ValidateAnalysisResult(result, New().Manifest(), root); err != nil {
		t.Fatalf("ValidateAnalysisResult() error = %v", err)
	}
}

func TestStaticDependenciesAreDeterministicAndCancellationIsHonored(t *testing.T) {
	root := t.TempDir()
	writeClojureFile(t, root, "deps.edn", "{:paths [\"src\"]}\n")
	writeClojureFile(t, root, "src/app/a.clj", "(ns app.a (:require [app.b :as b]))\n")
	writeClojureFile(t, root, "src/app/b.clj", "(ns app.b)\n")
	options := testClojureOptions(t, nil)
	project, err := ResolveProject(root, options)
	if err != nil {
		t.Fatal(err)
	}
	firstDiscovery, err := Discover(context.Background(), project, options)
	if err != nil {
		t.Fatal(err)
	}
	secondDiscovery, err := Discover(context.Background(), project, options)
	if err != nil {
		t.Fatal(err)
	}
	first := BuildResult(project, firstDiscovery, New().Manifest())
	second := BuildResult(project, secondDiscovery, New().Manifest())
	firstData := mustMarshalClojure(t, first)
	secondData := mustMarshalClojure(t, second)
	if string(firstData) != string(secondData) {
		t.Fatalf("repeated dependency result differs:\n%s\n%s", firstData, secondData)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Discover(ctx, project, options); err == nil {
		t.Fatal("Discover(cancelled) returned nil error")
	}
}

func runClojureAnalysis(t *testing.T, root string, cli map[string]any) analysis.AnalysisResult {
	t.Helper()
	options := testClojureOptions(t, cli)
	project, err := ResolveProject(root, options)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := Discover(context.Background(), project, options)
	if err != nil {
		t.Fatal(err)
	}
	return BuildResult(project, discovery, New().Manifest())
}

func findClojureRelationship(relationships []analysis.RelationshipObservation, from, target, kind string) *analysis.RelationshipObservation {
	for index := range relationships {
		relationship := &relationships[index]
		if relationship.FromModuleID != from || relationship.Type != "depends_on" {
			continue
		}
		if relationship.ToModuleID == target && metadataContainsString(relationship.Metadata, "dependency_kinds", kind) {
			return relationship
		}
	}
	return nil
}

func findClojureRelationshipByScope(relationships []analysis.RelationshipObservation, references []analysis.Reference, from, target, scope string) *analysis.RelationshipObservation {
	for index := range relationships {
		relationship := &relationships[index]
		if relationship.FromModuleID != from || relationship.ToReferenceID == "" || relationship.Metadata["target_namespace"] != target {
			continue
		}
		for _, reference := range references {
			if reference.ID == relationship.ToReferenceID && reference.Scope == scope {
				return relationship
			}
		}
	}
	return nil
}

func mustMarshalClojure(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
