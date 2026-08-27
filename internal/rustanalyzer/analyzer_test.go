package rustanalyzer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model/canonical"
)

func TestManifestAndCargoDetection(t *testing.T) {
	analyzer := New()
	manifest := analyzer.Manifest()
	if manifest.ID != "org.archview.rust" || manifest.Language != "rust" || len(manifest.DetectionMarkers) != 1 || manifest.DetectionMarkers[0].Value != "Cargo.toml" {
		t.Fatalf("manifest = %#v", manifest)
	}
	if err := analysis.ValidateManifest(manifest); err != nil {
		t.Fatalf("manifest validation: %v", err)
	}
	root := t.TempDir()
	candidate, err := analyzer.Detect(context.Background(), analysis.DetectRequest{ProjectRoot: root})
	if err != nil {
		t.Fatalf("detect without marker: %v", err)
	}
	if candidate.Confidence != 0 {
		t.Fatalf("candidate without Cargo.toml = %#v", candidate)
	}
	writeRustFile(t, root, "Cargo.toml", "[package]\nname = \"demo\"\nedition = \"2021\"\n")
	candidate, err = analyzer.Detect(context.Background(), analysis.DetectRequest{ProjectRoot: root})
	if err != nil || candidate.Confidence != 1 || len(candidate.MatchedMarkers) != 1 {
		t.Fatalf("Cargo candidate = %#v, error=%v", candidate, err)
	}
}

func TestSingleCrateDiscoversModulesEvidenceAndRelationships(t *testing.T) {
	root := t.TempDir()
	writeRustFile(t, root, "Cargo.toml", `[package]
name = "demo"
edition = "2021"

[dependencies]
serde = { version = "1", features = ["derive"] }

[dependencies.table-dep]
version = "2"
default-features = false
features = ["alloc"]

[dev-dependencies]
pretty_assertions = "1"

[build-dependencies]
cc = "1"
`)
	writeRustFile(t, root, "src/lib.rs", `pub mod service;
mod inline {
    pub mod nested;
    pub use crate::service::Thing;
}
use crate::service::Thing;
use std::fmt;
use serde::Serialize;
`)
	writeRustFile(t, root, "src/service.rs", `pub struct Thing;
use super::inline::Thing as InlineThing;
`)
	writeRustFile(t, root, "src/inline/nested.rs", `pub fn nested() {}
`)
	result := runRust(t, root, nil)
	if result.Status != analysis.StatusComplete {
		t.Fatalf("result status = %s, diagnostics=%#v", result.Status, result.Diagnostics)
	}
	if len(result.Modules) != 4 {
		t.Fatalf("modules = %#v", result.Modules)
	}
	moduleKinds := make(map[string]string)
	for _, module := range result.Modules {
		moduleKinds[module.DisplayName] = module.Kind
		if len(module.SourceReferenceIDs) == 0 {
			t.Errorf("module %s has no source evidence", module.DisplayName)
		}
		if module.Hierarchy == nil {
			t.Errorf("module %s has nil hierarchy", module.DisplayName)
		}
	}
	if moduleKinds["demo"] != "crate" || moduleKinds["demo::service"] != "file_module" || moduleKinds["demo::inline"] != "inline_module" || moduleKinds["demo::inline::nested"] != "file_module" {
		t.Fatalf("module kinds = %#v", moduleKinds)
	}
	for _, source := range result.SourceReferences {
		if source.Kind == "module_declaration" && strings.HasSuffix(source.Symbol, "::nested") && source.Symbol != "demo::inline::nested" {
			t.Fatalf("nested module source symbol = %#v", source)
		}
	}
	if len(result.SourceReferences) < 5 {
		t.Fatalf("source references = %#v", result.SourceReferences)
	}
	seenKinds := map[string]bool{}
	for _, relationship := range result.Relationships {
		if kind, ok := relationship.Metadata["kind"].(string); ok {
			seenKinds[kind] = true
		}
	}
	if !seenKinds["use"] || !seenKinds["pub_use"] || !seenKinds["dependency"] {
		t.Fatalf("relationship kinds = %#v", seenKinds)
	}
	if !hasReferenceScope(result, "standard_library") || !hasReferenceScope(result, "external") {
		t.Fatalf("references = %#v", result.References)
	}
	if !hasReferenceName(result, "table-dep") {
		t.Fatalf("dotted Cargo dependency table was not emitted: %#v", result.References)
	}
	for _, reference := range result.References {
		if reference.Scope != "standard_library" && reference.Scope != "external" {
			t.Fatalf("non-canonical dependency scope = %#v", reference)
		}
	}
	for _, source := range result.SourceReferences {
		if filepath.IsAbs(source.Path) || strings.HasPrefix(filepath.ToSlash(source.Path), "../") {
			t.Fatalf("source path escaped project: %#v", source)
		}
	}
	for _, module := range result.Modules {
		if module.DisplayName == "demo" {
			declared, _ := module.Metadata["declared_modules"].([]string)
			if !containsString(declared, "service") || !containsString(declared, "inline") {
				t.Fatalf("root declared modules = %#v", module.Metadata)
			}
		}
	}
}

func TestWorkspaceRequiresExplicitCrateAndSupportsNameOrRelativePath(t *testing.T) {
	root := t.TempDir()
	writeRustFile(t, root, "Cargo.toml", `[workspace]
members = ["crates/api", "crates/worker",]
`)
	writeRustFile(t, root, "crates/api/Cargo.toml", `[package]
name = "api"
edition = "2021"
`)
	writeRustFile(t, root, "crates/api/src/lib.rs", `pub fn api() {}`)
	writeRustFile(t, root, "crates/worker/Cargo.toml", `[package]
name = "worker"
edition = "2021"
`)
	writeRustFile(t, root, "crates/worker/src/lib.rs", `pub fn worker() {}`)

	_, err := ResolveProject(root, effectiveRustOptions(t, nil))
	var hostErr *analysis.HostError
	if !errors.As(err, &hostErr) || hostErr.Code != analysis.ErrModuleSelection {
		t.Fatalf("ambiguous workspace error = %v", err)
	}
	project, err := ResolveProject(root, effectiveRustOptions(t, map[string]any{"crate": "api"}))
	if err != nil || project.PackageName != "api" || project.RelativeCrateRoot != "crates/api" {
		t.Fatalf("name-selected project = %#v, error=%v", project, err)
	}
	project, err = ResolveProject(root, effectiveRustOptions(t, map[string]any{"crate": "crates/worker"}))
	if err != nil || project.PackageName != "worker" {
		t.Fatalf("path-selected project = %#v, error=%v", project, err)
	}
	if runtime.GOOS == "windows" {
		project, err = ResolveProject(root, effectiveRustOptions(t, map[string]any{"crate": "CRATES/WORKER"}))
		if err != nil || project.PackageName != "worker" {
			t.Fatalf("case-insensitive path-selected project = %#v, error=%v", project, err)
		}
	}
	_, err = ResolveProject(root, effectiveRustOptions(t, map[string]any{"crate": "../outside"}))
	if !errors.As(err, &hostErr) || hostErr.Code != analysis.ErrUnsupportedProject {
		t.Fatalf("escaping crate selection error = %v", err)
	}
}

func TestWorkspaceRootSelfMemberIsNotDuplicated(t *testing.T) {
	root := t.TempDir()
	writeRustFile(t, root, "Cargo.toml", `[package]
name = "root_crate"
edition = "2021"

[workspace]
members = ["."]
`)
	writeRustFile(t, root, "src/lib.rs", `pub fn root() {}
`)
	project, err := ResolveProject(root, effectiveRustOptions(t, nil))
	if err != nil || project.PackageName != "root_crate" {
		t.Fatalf("root self-member selection = %#v, error=%v", project, err)
	}
}

func TestRustNestedFileModulesAndGeneratedSourcesStayBounded(t *testing.T) {
	root := t.TempDir()
	writeRustFile(t, root, "Cargo.toml", `[package]
name = "bounded"
edition = "2021"
`)
	writeRustFile(t, root, "src/lib.rs", `mod network;
mod generated;
`)
	writeRustFile(t, root, "src/network.rs", `pub mod http;
`)
	writeRustFile(t, root, "src/network/http.rs", `pub fn get() {}`)
	writeRustFile(t, root, "src/generated.rs", `// @generated by a build tool
pub fn generated() {}`)
	result := runRust(t, root, nil)
	if !hasModule(result, "bounded::network") || !hasModule(result, "bounded::network::http") {
		t.Fatalf("nested file modules = %#v", result.Modules)
	}
	if hasModule(result, "bounded::generated") || !hasDiagnostic(result, "rust_generated_source") {
		t.Fatalf("generated source handling = modules %#v diagnostics %#v", result.Modules, result.Diagnostics)
	}
}

func TestRustInnerAttributesAndMalformedUseRemainRecoverable(t *testing.T) {
	root := t.TempDir()
	writeRustFile(t, root, "Cargo.toml", `[package]
name = "attributes"
edition = "2021"
`)
	writeRustFile(t, root, "src/lib.rs", `#![allow(dead_code)]
pub fn punctuation() {
    let closing = "}";
    let opening = '}';
}
pub mod api;
use super;
`)
	writeRustFile(t, root, "src/api.rs", `pub struct Api;
`)
	result := runRust(t, root, nil)
	if !hasModule(result, "attributes::api") {
		t.Fatalf("inner attribute hid module discovery: %#v", result.Modules)
	}
	if result.Status != analysis.StatusPartial || !hasDiagnostic(result, "rust_unresolved_use") {
		t.Fatalf("malformed use handling = status %s diagnostics %#v", result.Status, result.Diagnostics)
	}
	if hasDiagnostic(result, "rust_syntax_error") {
		t.Fatalf("string punctuation was treated as syntax: %#v", result.Diagnostics)
	}
}

func TestRustGroupedUseStopsAtStatementBoundary(t *testing.T) {
	root := t.TempDir()
	writeRustFile(t, root, "Cargo.toml", `[package]
name = "grouped"
edition = "2021"
`)
	writeRustFile(t, root, "src/lib.rs", `pub mod api;
use crate::api::{Thing, Type as Alias};
pub mod later;
`)
	writeRustFile(t, root, "src/api.rs", `pub struct Thing;
pub struct Type;
`)
	writeRustFile(t, root, "src/later.rs", `pub fn later() {}
`)
	result := runRust(t, root, nil)
	if hasDiagnostic(result, "rust_unresolved_use") {
		t.Fatalf("grouped use consumed following items: %#v", result.Diagnostics)
	}
	useRelationships := 0
	for _, relationship := range result.Relationships {
		if relationship.Metadata["kind"] != "use" {
			continue
		}
		useRelationships++
		paths, _ := relationship.Metadata["import_paths"].([]string)
		if len(paths) != 2 {
			t.Fatalf("grouped use paths = %#v", relationship.Metadata)
		}
	}
	if useRelationships != 1 {
		t.Fatalf("grouped use relationships = %#v", result.Relationships)
	}
}

func TestRustRootModulesResolvePerTargetSourceDirectory(t *testing.T) {
	root := t.TempDir()
	writeRustFile(t, root, "Cargo.toml", `[package]
name = "multi_target"
edition = "2021"
`)
	writeRustFile(t, root, "src/lib.rs", `mod shared;
`)
	writeRustFile(t, root, "src/shared.rs", `pub fn shared() {}
`)
	writeRustFile(t, root, "src/bin/tool.rs", `mod tool_only;
`)
	writeRustFile(t, root, "src/bin/tool_only.rs", `pub fn tool_only() {}
`)
	result := runRust(t, root, nil)
	if !hasModule(result, "multi_target::shared") || !hasModule(result, "multi_target::tool_only") {
		t.Fatalf("target-specific root modules = %#v", result.Modules)
	}
}

func TestRustAnalysisDoesNotExecuteBuildScriptAndHonorsScopeOptions(t *testing.T) {
	root := t.TempDir()
	writeRustFile(t, root, "Cargo.toml", `[package]
name = "safe"
version = "0.1.0"
build = "build.rs"

[lib]
path = "src/lib.rs"
`)
	writeRustFile(t, root, "build.rs", `std::fs::write("executed-marker", "bad").unwrap();`)
	writeRustFile(t, root, "src/lib.rs", `#[cfg(test)]
mod tests;
#[cfg(not(test))]
mod production;
#[cfg(any(test, feature = "extra"))]
mod mixed;
pub mod api;
`)
	writeRustFile(t, root, "src/api.rs", `pub fn api() {}`)
	writeRustFile(t, root, "src/production.rs", `pub fn production() {}`)
	writeRustFile(t, root, "src/mixed.rs", `pub fn mixed() {}`)
	writeRustFile(t, root, "tests/integration.rs", `use safe::api;`)
	writeRustFile(t, root, "examples/demo.rs", `fn main() {}`)

	result := runRust(t, root, nil)
	if _, err := os.Stat(filepath.Join(root, "executed-marker")); !os.IsNotExist(err) {
		t.Fatalf("build script side effect exists: %v", err)
	}
	if hasModule(result, "safe::tests") || hasModule(result, "integration") || hasModule(result, "demo") {
		t.Fatalf("default scope included excluded targets: %#v", result.Modules)
	}
	if !hasModule(result, "safe::production") || !hasModule(result, "safe::mixed") {
		t.Fatalf("default scope dropped non-test cfg modules: %#v", result.Modules)
	}
	result = runRust(t, root, map[string]any{"include_tests": true, "include_examples": true})
	if !hasModulePath(result, "tests/integration.rs") && len(result.Modules) < 3 {
		t.Fatalf("expanded scope did not add source observations: %#v", result.Modules)
	}
}

func TestRustPathCfgAndMacroUncertaintyRemainTraceable(t *testing.T) {
	root := t.TempDir()
	writeRustFile(t, root, "Cargo.toml", `[package]
name = "traceable"
edition = "2021"
`)
	writeRustFile(t, root, "src/lib.rs", `#[path = "components/api.rs"]
mod api;
#[cfg(feature = "experimental")]
mod optional;
use crate::{api::{Thing as Renamed}, api};
use missing_crate::Thing;
#[derive(Clone)]
pub struct Derived;
include!("generated.rs");
`)
	writeRustFile(t, root, "src/components/api.rs", `pub struct Thing;
`)
	writeRustFile(t, root, "src/optional.rs", `pub fn optional() {}`)
	result := runRust(t, root, nil)
	if !hasModule(result, "traceable::api") || !hasModule(result, "traceable::optional") {
		t.Fatalf("path/cfg modules = %#v", result.Modules)
	}
	if result.Status != analysis.StatusPartial {
		t.Fatalf("uncertain result status = %s, diagnostics=%#v", result.Status, result.Diagnostics)
	}
	if !hasDiagnostic(result, "rust_cfg_uncertain") || !hasDiagnostic(result, "rust_macro_attribute") || !hasDiagnostic(result, "rust_macro_uncertain") || !hasDiagnostic(result, "rust_unresolved_use") {
		t.Fatalf("uncertainty diagnostics = %#v", result.Diagnostics)
	}
	if !hasReferenceScope(result, "unresolved") {
		t.Fatalf("unresolved references = %#v", result.References)
	}
	for _, module := range result.Modules {
		conditions, _ := module.Metadata["cfg_conditions"].([]string)
		switch module.DisplayName {
		case "traceable":
			if len(conditions) != 0 {
				t.Fatalf("root module inherited child cfg conditions = %#v", module.Metadata)
			}
		case "traceable::optional":
			if !containsString(conditions, `feature = "experimental"`) {
				t.Fatalf("conditional module metadata = %#v", module.Metadata)
			}
		}
	}
	localKinds := map[string]bool{}
	for _, relationship := range result.Relationships {
		if relationship.ToModuleID != "" {
			localKinds[relationship.Metadata["kind"].(string)] = true
		}
	}
	if !localKinds["use"] {
		t.Fatalf("local use relationships = %#v", result.Relationships)
	}
	if _, err := canonical.Normalize(result); err != nil {
		t.Fatalf("canonical Rust normalization: %v", err)
	}
}

func TestRustRelativeUnresolvedUseStaysUnresolved(t *testing.T) {
	project := Project{
		PackageName:  "relative",
		Dependencies: []CargoDependency{{Name: "serde"}},
	}
	root := &rustModule{ID: rustModuleID(project.PackageName, nil), Path: []string{}}
	inner := &rustModule{ID: rustModuleID(project.PackageName, []string{"inner"}), Path: []string{"inner"}}
	target, scope, reference := resolveRustUse(project, map[string]*rustModule{"": root, "inner": inner}, rustUseObservation{
		FromModuleID: inner.ID,
		Path:         "super::serde::Serialize",
	})
	if target != nil || scope != "unresolved" || reference.Scope != "unresolved" {
		t.Fatalf("relative unresolved use = target=%#v scope=%q reference=%#v", target, scope, reference)
	}
	target, scope, reference = resolveRustUse(project, map[string]*rustModule{"": root, "inner": inner}, rustUseObservation{
		FromModuleID: inner.ID,
		Path:         "self::missing::Thing",
	})
	if target != nil || scope != "unresolved" || reference.Scope != "unresolved" {
		t.Fatalf("self unresolved use = target=%#v scope=%q reference=%#v", target, scope, reference)
	}
}

func runRust(t *testing.T, root string, values map[string]any) analysis.AnalysisResult {
	t.Helper()
	registry := analysis.NewRegistry()
	if err := registry.Register(New()); err != nil {
		t.Fatalf("register Rust analyzer: %v", err)
	}
	result, err := analysis.NewHost(registry).Run(context.Background(), analysis.RunRequest{ProjectRoot: root, AnalyzerID: "org.archview.rust", CLIOptions: values})
	if err != nil {
		t.Fatalf("Rust analysis: %v", err)
	}
	return result
}

func effectiveRustOptions(t *testing.T, values map[string]any) analysis.EffectiveOptions {
	t.Helper()
	options, err := analysis.ResolveOptions(New().Manifest(), nil, values)
	if err != nil {
		t.Fatalf("Rust options: %v", err)
	}
	return options
}

func writeRustFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create %s: %v", relative, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relative, err)
	}
}

func hasReferenceScope(result analysis.AnalysisResult, scope string) bool {
	for _, reference := range result.References {
		if reference.Scope == scope {
			return true
		}
	}
	return false
}

func hasReferenceName(result analysis.AnalysisResult, name string) bool {
	for _, reference := range result.References {
		if reference.Name == name {
			return true
		}
	}
	return false
}

func hasModule(result analysis.AnalysisResult, displayName string) bool {
	for _, module := range result.Modules {
		if module.DisplayName == displayName {
			return true
		}
	}
	return false
}

func hasModulePath(result analysis.AnalysisResult, relativePath string) bool {
	for _, source := range result.SourceReferences {
		if source.Path == relativePath {
			return true
		}
	}
	return false
}

func hasDiagnostic(result analysis.AnalysisResult, code string) bool {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
