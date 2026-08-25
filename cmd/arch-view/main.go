package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/goanalyzer"
)

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	registry := analysis.NewRegistry()
	if err := registry.Register(goanalyzer.New()); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	host := analysis.NewHost(registry)
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "analyzers":
		if len(args) != 1 {
			err := analysis.NewHostError(analysis.ErrInvalidRequest, "analyzers does not accept positional arguments", nil)
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
		return writeJSON(stdout, struct {
			Analyzers []analysis.Manifest `json:"analyzers"`
		}{Analyzers: host.ListManifests()})
	case "analyze":
		return runAnalyze(host, args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "unknown command", map[string]any{"command": args[0]})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
}

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
	format := fs.String("format", "analysis-json", "output format")
	output := fs.String("output", "", "output file, or - for stdout")
	var buildTags stringList
	var excludes stringList
	fs.Var(&buildTags, "build-tag", "explicit Go build tag; repeatable")
	fs.Var(&excludes, "exclude", "repository-relative exclusion glob; repeatable")
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
	if *format != "analysis-json" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "the first analysis slice supports only analysis-json", map[string]any{"format": *format})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	cliOptions := map[string]any{
		"include_tests":     *includeTests,
		"include_generated": *includeGenerated,
		"include_external":  *includeExternal,
		"safe_mode":         *safeMode,
	}
	if *module != "" {
		cliOptions["module"] = *module
	}
	if len(buildTags) > 0 {
		cliOptions["build_tags"] = []string(buildTags)
	}
	if len(excludes) > 0 {
		cliOptions["exclude"] = []string(excludes)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	result, err := host.Run(ctx, analysis.RunRequest{
		ProjectRoot:    *project,
		Language:       *language,
		AnalyzerID:     *analyzerID,
		CLIOptions:     cliOptions,
		ProjectOptions: map[string]any{},
	})
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
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

func writeJSON(writer io.Writer, value any) int {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return 4
	}
	return 0
}

func writeFileJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "analysis result could not be serialized", err, nil)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "analysis result could not be written", err, map[string]any{"output": path})
	}
	return nil
}

func writeError(writer io.Writer, err error) {
	data, marshalErr := analysis.MarshalError(err)
	if marshalErr != nil {
		_, _ = fmt.Fprintln(writer, err)
		return
	}
	_, _ = fmt.Fprintln(writer, string(data))
}

func printUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, "arch-view analyzers")
	_, _ = fmt.Fprintln(writer, "arch-view analyze --project <path> [--language <id>] [--module <path>] --format analysis-json --output <file>")
}
