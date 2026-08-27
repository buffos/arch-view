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
	language := fs.String("language", "", "explicit analyzer language")
	analyzerID := fs.String("analyzer", "", "explicit analyzer id")
	module := fs.String("module", "", "explicit Go module path or workspace-relative directory")
	includeTests := fs.Bool("include-tests", false, "include test files")
	includeGenerated := fs.Bool("include-generated", false, "include generated files")
	includeExternal := fs.Bool("include-external", false, "retain non-local reference detail")
	safeMode := fs.Bool("safe-mode", true, "disable target-code execution and tool-assisted execution")
	pythonVersion := fs.String("python-version", "", "Python major/minor version for static analysis")
	includeStubs := fs.Bool("include-stubs", false, "include Python .pyi stub files")
	format := fs.String("format", "analysis-json", "output format")
	output := fs.String("output", "", "output file, or - for stdout")
	referenceVisibility := fs.String("reference-visibility", "hidden", "visual reference visibility: hidden, aggregated, or expanded")
	deterministic := fs.Bool("deterministic", true, "produce deterministic output")
	overwrite := fs.Bool("overwrite", false, "replace an existing export output")
	embedSource := fs.Bool("embed-source", false, "embed source contents; unsupported in v1")
	var buildTags stringList
	var excludes stringList
	var sourceRoots stringList
	var viewPath stringList
	var referenceScopes stringList
	fs.Var(&buildTags, "build-tag", "explicit Go build tag; repeatable")
	fs.Var(&excludes, "exclude", "repository-relative exclusion glob; repeatable")
	fs.Var(&sourceRoots, "source-root", "explicit Python source root; repeatable")
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
	cliOptions := collectAnalyzerCLIOptions(fs, analyzerCLIFlags{
		module:           module,
		includeTests:     includeTests,
		includeGenerated: includeGenerated,
		includeExternal:  includeExternal,
		safeMode:         safeMode,
		buildTags:        &buildTags,
		excludes:         &excludes,
		sourceRoots:      &sourceRoots,
		pythonVersion:    pythonVersion,
		includeStubs:     includeStubs,
	})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	result, err := host.Run(ctx, analysis.RunRequest{ProjectRoot: *project, Language: *language, AnalyzerID: *analyzerID, CLIOptions: cliOptions, ProjectOptions: map[string]any{}})
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
