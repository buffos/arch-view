package clojureanalyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
)

func TestReaderConditionalsSelectPlatformsAndRetainMetadata(t *testing.T) {
	root := t.TempDir()
	writeClojureFile(t, root, "deps.edn", "{:paths [\"src\"]}\n")
	writeClojureFile(t, root, "src/app/conditional.cljc", `(ns app.conditional
  #?(:clj (:require [app.jvm :as j])
     :cljs (:require [app.browser :as b])
     :default (:require [app.default :as d])))
`)
	writeClojureFile(t, root, "src/app/jvm.clj", "(ns app.jvm)\n")
	writeClojureFile(t, root, "src/app/browser.cljs", "(ns app.browser)\n")
	writeClojureFile(t, root, "src/app/default.clj", "(ns app.default)\n")

	all := runClojureAnalysis(t, root, map[string]any{"platform": "both"})
	if all.Status != analysis.StatusComplete {
		t.Fatalf("both status = %s, diagnostics = %#v", all.Status, all.Diagnostics)
	}
	for _, expected := range []string{"app.jvm", "app.browser", "app.default"} {
		relationship := findClojureRelationship(all.Relationships, "clj:app.conditional", "clj:"+expected, "require")
		if relationship == nil || relationship.Metadata["conditional"] != true {
			t.Fatalf("conditional relationship %q = %#v", expected, relationship)
		}
	}
	if !metadataContainsString(findClojureRelationship(all.Relationships, "clj:app.conditional", "clj:app.jvm", "require").Metadata, "platforms", "clj") {
		t.Fatalf("clj platform metadata absent")
	}
	if !metadataContainsString(findClojureRelationship(all.Relationships, "clj:app.conditional", "clj:app.browser", "require").Metadata, "platforms", "cljs") {
		t.Fatalf("cljs platform metadata absent")
	}
	if !metadataContainsString(findClojureRelationship(all.Relationships, "clj:app.conditional", "clj:app.default", "require").Metadata, "platforms", "default") {
		t.Fatalf("default platform metadata absent")
	}

	clj := runClojureAnalysis(t, root, map[string]any{"platform": "clj"})
	if findClojureRelationship(clj.Relationships, "clj:app.conditional", "clj:app.browser", "require") != nil || findClojureRelationship(clj.Relationships, "clj:app.conditional", "clj:app.default", "require") != nil || findClojureRelationship(clj.Relationships, "clj:app.conditional", "clj:app.jvm", "require") == nil {
		t.Fatalf("clj conditional relationships = %#v", clj.Relationships)
	}

	cljs := runClojureAnalysis(t, root, map[string]any{"platform": "cljs"})
	if findClojureRelationship(cljs.Relationships, "clj:app.conditional", "clj:app.jvm", "require") != nil || findClojureRelationship(cljs.Relationships, "clj:app.conditional", "clj:app.default", "require") != nil || findClojureRelationship(cljs.Relationships, "clj:app.conditional", "clj:app.browser", "require") == nil {
		t.Fatalf("cljs conditional relationships = %#v", cljs.Relationships)
	}
}

func TestSplicedAndMalformedReaderConditionalsRemainStatic(t *testing.T) {
	root := t.TempDir()
	writeClojureFile(t, root, "deps.edn", "{:paths [\"src\"]}\n")
	writeClojureFile(t, root, "src/app/splice.cljc", `(ns app.splice
  #?@(:clj [(:require [app.jvm])]))
`)
	writeClojureFile(t, root, "src/app/bad.cljc", `(ns app.bad
  #?(:wat (:require [app.fake])
     :clj (:require [app.jvm])))
`)
	writeClojureFile(t, root, "src/app/malformed.cljc", `(ns app.malformed
  #?(:clj (:require [app.jvm]) :cljs))
`)
	writeClojureFile(t, root, "src/app/malformed-source.clj", "(ns app.malformed-source)\n(def value [1 2\n")
	writeClojureFile(t, root, "src/app/jvm.clj", "(ns app.jvm)\n")

	result := runClojureAnalysis(t, root, map[string]any{"platform": "clj"})
	if result.Status != analysis.StatusPartial || !hasDiagnostic(result.Diagnostics, "clojure_reader_conditional") {
		t.Fatalf("reader conditional result = %#v", result)
	}
	if findClojureRelationship(result.Relationships, "clj:app.splice", "clj:app.jvm", "require") == nil {
		t.Fatalf("spliced conditional was not extracted: %#v", result.Relationships)
	}
	if findClojureRelationship(result.Relationships, "clj:app.bad", "clj:app.fake", "require") != nil {
		t.Fatalf("unsupported selector fabricated dependency: %#v", result.Relationships)
	}
	if findClojureRelationship(result.Relationships, "clj:app.malformed", "clj:app.jvm", "require") != nil {
		t.Fatalf("malformed conditional fabricated dependency: %#v", result.Relationships)
	}
	if !hasModule(result.Modules, "clj:app.malformed-source") || !hasDiagnostic(result.Diagnostics, "clojure_syntax_error") {
		t.Fatalf("malformed source did not retain usable observations: %#v", result)
	}
}

func TestPolymorphicMetadataAndDynamicLoadingAreSafe(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "MUST-NOT-BE-CREATED")
	writeClojureFile(t, root, "deps.edn", "{:paths [\"src\"]}\n")
	writeClojureFile(t, root, "src/app/poly.clj", `(ns app.poly)
(defprotocol Service (run [this value]))
(defmulti render (fn [value] (:kind value)))
`)
	writeClojureFile(t, root, "src/app/dynamic.clj", "(ns app.dynamic)\n(require 'app.runtime)\n(require \"(spit \\\"secret\\\")\")\n(load-file \"other.clj\")\n(load-string \"(spit \\\""+marker+"\\\" \\\"executed\\\")\")\n(eval (spit \""+marker+"\" \"executed\"))\n")
	writeClojureFile(t, root, "src/app/quoted.clj", "(ns app.quoted)\n'(require 'app.not-runtime)\n(quote (defmulti not-a-multi identity))\n")

	result := runClojureAnalysis(t, root, nil)
	if result.Status != analysis.StatusPartial || !hasDiagnostic(result.Diagnostics, "clojure_dynamic_loading") || !hasDiagnostic(result.Diagnostics, "clojure_dynamic_reference") {
		t.Fatalf("dynamic result = %#v", result)
	}
	poly := findModule(result.Modules, "clj:app.poly")
	if poly == nil || !containsString(poly.Tags, "polymorphic") || poly.Metadata["polymorphic"] != true || !metadataContainsString(poly.Metadata, "polymorphic_forms", "defprotocol:Service") || !metadataContainsString(poly.Metadata, "polymorphic_forms", "defmulti:render") {
		t.Fatalf("polymorphic module = %#v", poly)
	}
	if !sort.StringsAreSorted(poly.SourceReferenceIDs) {
		t.Fatalf("polymorphic evidence is not sorted: %#v", poly.SourceReferenceIDs)
	}
	if len(findClojureDynamicRelationships(result.Relationships, "clj:app.dynamic")) != 4 {
		t.Fatalf("dynamic relationships = %#v", result.Relationships)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret") || strings.Contains(string(data), "spit") {
		t.Fatalf("dynamic source-like string leaked into result: %s", data)
	}
	if len(findClojureDynamicRelationships(result.Relationships, "clj:app.quoted")) != 0 {
		t.Fatalf("quoted forms were reported as dynamic: %#v", result.Relationships)
	}
	quoted := findModule(result.Modules, "clj:app.quoted")
	if quoted == nil || containsString(quoted.Tags, "polymorphic") {
		t.Fatalf("quoted forms were reported as polymorphic: %#v", quoted)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("dynamic source was evaluated: %v", err)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func findClojureDynamicRelationships(relationships []analysis.RelationshipObservation, from string) []analysis.RelationshipObservation {
	result := make([]analysis.RelationshipObservation, 0)
	for _, relationship := range relationships {
		if relationship.FromModuleID == from && relationship.Metadata["target_scope"] == "dynamic" {
			result = append(result, relationship)
		}
	}
	return result
}
