package main

import (
	"context"
	"flag"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	exporter "github.com/buffo/arch-view/internal/export"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/viewer/layout"
)

func runAnalyze(host *analysis.Host, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view analyze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	project := fs.String("project", "", "project root")
	analyzerRuntime := fs.String("analyzer-runtime", analyzerRuntimeAuto, "analyzer runtime: auto, packaged, in-process, or explicit")
	allowUntrustedPlugin := fs.Bool("allow-untrusted-plugin", false, "allow an explicitly supplied local descriptor")
	language := fs.String("language", "", "explicit analyzer language")
	analyzerID := fs.String("analyzer", "", "explicit analyzer id")
	scope := fs.String("scope", "", "cached aggregate scope id; omit for All")
	module := fs.String("module", "", "explicit Go module path or workspace-relative directory")
	crate := fs.String("crate", "", "explicit Rust package name or workspace-relative crate directory")
	target := fs.String("target", "", "explicit Rust target triple or target selector")
	config := fs.String("config", "", "explicit TypeScript tsconfig path")
	includeJS := fs.Bool("include-js", false, "include JavaScript and JSX files")
	includeTests := fs.Bool("include-tests", false, "include test files")
	includeExamples := fs.Bool("include-examples", false, "include Rust examples and benches")
	includeGenerated := fs.Bool("include-generated", false, "include generated files")
	includeExternal := fs.Bool("include-external", false, "retain non-local reference detail")
	safeMode := fs.Bool("safe-mode", true, "disable target-code execution and tool-assisted execution")
	pythonVersion := fs.String("python-version", "", "Python major/minor version for static analysis")
	includeStubs := fs.Bool("include-stubs", false, "include Python .pyi stub files")
	platform := fs.String("platform", "", "Clojure reader-conditional platform: clj, cljs, or both")
	runtime := fs.String("runtime", "auto", "TypeScript runtime context: auto, esm, or cjs")
	format := fs.String("format", "analysis-json", "output format")
	output := fs.String("output", "", "output file, or - for stdout")
	referenceVisibility := fs.String("reference-visibility", "hidden", "visual reference visibility: hidden, aggregated, or expanded")
	deterministic := fs.Bool("deterministic", true, "produce deterministic output")
	overwrite := fs.Bool("overwrite", false, "replace an existing export output")
	embedSource := fs.Bool("embed-source", false, "embed source contents; unsupported in v1")
	var buildTags stringList
	var features stringList
	var excludes stringList
	var sourceRoots stringList
	var plugins stringList
	var viewPath stringList
	var referenceScopes stringList
	fs.Var(&buildTags, "build-tag", "explicit Go build tag; repeatable")
	fs.Var(&features, "feature", "explicit Rust Cargo feature; repeatable")
	fs.Var(&features, "features", "explicit Rust Cargo feature; repeatable")
	fs.Var(&excludes, "exclude", "repository-relative exclusion glob; repeatable")
	fs.Var(&sourceRoots, "source-root", "explicit analyzer source root; repeatable")
	fs.Var(&plugins, "plugin", "external analyzer descriptor; repeatable")
	fs.Var(&viewPath, "view-path", "hierarchy segment for visual export; repeatable")
	fs.Var(&referenceScopes, "reference-scope", "reference scope for visual export; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "analyze does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *project == "" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "analyze requires --project", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *output == "" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "analyze requires --output", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	formatValue := strings.ToLower(strings.TrimSpace(*format))
	if formatValue != "analysis-json" && !supportedExportFormat(formatValue) {
		err := analysis.NewHostError(analysis.ErrUnsupportedOption, "analysis format is unsupported", map[string]any{"format": *format, "supported": []string{"analysis-json", exporter.FormatJSON, exporter.FormatHTML, exporter.FormatSVG}})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if formatValue != "analysis-json" && !*deterministic {
		err := analysis.NewHostError(analysis.ErrUnsupportedOption, "non-deterministic export is unsupported", map[string]any{"deterministic": false})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if formatValue == "analysis-json" && (*referenceVisibility != "hidden" || len(viewPath) > 0 || len(referenceScopes) > 0 || *overwrite || !*deterministic || *embedSource) {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "export options require json, html, or svg format", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *scope != "" && (*analyzerID != "" || *language != "") {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "--scope cannot be combined with an explicit analyzer or language selection", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	configuredHost, _, err := configureCommandHost(host, commandRuntimeOptions{
		Mode:                 *analyzerRuntime,
		ModeProvided:         runtimeModeWasProvided(fs),
		AllowUntrustedPlugin: *allowUntrustedPlugin,
		PluginPaths:          []string(plugins),
		AnalyzerID:           *analyzerID,
		Language:             *language,
	})
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	host = configuredHost
	cliOptions := collectAnalyzerCLIOptions(fs, analyzerCLIFlags{
		module:           module,
		crate:            crate,
		features:         &features,
		target:           target,
		config:           config,
		includeJS:        includeJS,
		includeTests:     includeTests,
		includeExamples:  includeExamples,
		includeGenerated: includeGenerated,
		includeExternal:  includeExternal,
		safeMode:         safeMode,
		buildTags:        &buildTags,
		excludes:         &excludes,
		sourceRoots:      &sourceRoots,
		pythonVersion:    pythonVersion,
		includeStubs:     includeStubs,
		platform:         platform,
		runtime:          runtime,
	})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if *analyzerID == "" && *language == "" {
		run, err := runCombinedAnalysis(ctx, host, *project, cliOptions)
		if err != nil {
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
		if *scope != "" {
			if _, err := run.SelectAnalysisScope(*scope); err != nil {
				writeError(stderr, err)
				return analysis.ExitCodeForError(err)
			}
		}
		return writeCombinedAnalysisOutput(run, *format, *output, *project, *scope, *referenceVisibility, []string(viewPath), []string(referenceScopes), *overwrite, *embedSource, ctx, stdout, stderr)
	}
	result, err := runConfiguredSingleAnalysis(ctx, host, *project, *analyzerID, *language, cliOptions)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if formatValue != "analysis-json" {
		value, err := canonical.Normalize(result)
		if err != nil {
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
		var layoutProfile *layout.LayoutProfile
		if formatValue == exporter.FormatHTML {
			profile := layout.NewSession(*project).Response().Layout
			layoutProfile = &profile
		}
		return writeExport(value, exporter.Request{Format: formatValue, OutputPath: *output, ViewPath: []string(viewPath), ReferenceVisibility: *referenceVisibility, ReferenceScopes: []string(referenceScopes), LayoutProfile: layoutProfile, Overwrite: *overwrite, EmbedSource: *embedSource, Context: ctx}, stdout, stderr)
	}
	if *output == "-" {
		return writeJSON(stdout, result)
	}
	if err := writeFileJSON(*output, result); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	return analysis.ExitCodeForStatus(result.Status)
}
