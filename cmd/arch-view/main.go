package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/goanalyzer"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/viewer"
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
	case "open":
		return runOpen(host, args[1:], stdout, stderr)
	case "model":
		return runModel(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "unknown command", map[string]any{"command": args[0]})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
}

func runOpen(host *analysis.Host, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view open", flag.ContinueOnError)
	fs.SetOutput(stderr)
	modelInput := fs.String("model", "", "canonical model JSON input file")
	project := fs.String("project", "", "project root to analyze before opening")
	language := fs.String("language", "", "explicit analyzer language")
	analyzerID := fs.String("analyzer", "", "explicit analyzer id")
	module := fs.String("module", "", "explicit Go module path or workspace-relative directory")
	includeTests := fs.Bool("include-tests", false, "include test files")
	includeGenerated := fs.Bool("include-generated", false, "include generated files")
	includeExternal := fs.Bool("include-external", false, "retain non-local reference detail")
	safeMode := fs.Bool("safe-mode", true, "disable target-code execution and tool-assisted execution")
	port := fs.Int("port", 0, "loopback TCP port; 0 chooses an available port")
	var buildTags stringList
	var excludes stringList
	fs.Var(&buildTags, "build-tag", "explicit Go build tag; repeatable")
	fs.Var(&excludes, "exclude", "repository-relative exclusion glob; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "open does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if (*modelInput == "") == (*project == "") {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "open requires exactly one of --model or --project", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *port < 0 || *port > 65535 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "open --port must be between 0 and 65535", map[string]any{"port": *port})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var value model.Model
	if *modelInput != "" {
		var err error
		value, err = readModelFile(*modelInput)
		if err != nil {
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
	} else {
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
		value, err = model.Normalize(result)
		if err != nil {
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
	}
	server, err := viewer.NewServer(value)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(*port))
	if err != nil {
		hostErr := analysis.WrapHostError(analysis.ErrHostFailure, "local viewer could not bind its loopback port", err, map[string]any{"port": *port})
		writeError(stderr, hostErr)
		return analysis.ExitCodeForError(hostErr)
	}
	defer func() { _ = listener.Close() }()
	address := listener.Addr().String()
	_, _ = fmt.Fprintf(stdout, "arch-view listening at http://%s/\n", address)
	_, _ = fmt.Fprintln(stdout, "Press Ctrl+C to stop.")
	httpServer := &http.Server{
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownContext)
	}()
	if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		hostErr := analysis.WrapHostError(analysis.ErrHostFailure, "local viewer server failed", err, nil)
		writeError(stderr, hostErr)
		return analysis.ExitCodeForError(hostErr)
	}
	return 0
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

func runModel(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model requires a subcommand", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	switch args[0] {
	case "normalize":
		return runModelNormalize(args[1:], stdout, stderr)
	case "validate":
		return runModelValidate(args[1:], stdout, stderr)
	case "projection":
		return runModelProjection(args[1:], stdout, stderr)
	default:
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "unknown model subcommand", map[string]any{"command": args[0]})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
}

func runModelNormalize(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view model normalize", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "analysis JSON input file")
	output := fs.String("output", "", "model JSON output file, or - for stdout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model normalize does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *input == "" || *output == "" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model normalize requires --input and --output", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	data, err := os.ReadFile(*input)
	if err != nil {
		hostErr := analysis.WrapHostError(analysis.ErrInvalidModel, "analysis JSON input could not be read", err, map[string]any{"input": *input})
		writeError(stderr, hostErr)
		return analysis.ExitCodeForError(hostErr)
	}
	var result analysis.AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil {
		hostErr := analysis.WrapHostError(analysis.ErrInvalidModel, "analysis JSON input is invalid", err, map[string]any{"input": *input})
		writeError(stderr, hostErr)
		return analysis.ExitCodeForError(hostErr)
	}
	normalized, err := model.Normalize(result)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *output == "-" {
		return writeJSON(stdout, normalized)
	}
	if err := writeFileJSON(*output, normalized); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	return analysis.ExitCodeForStatus(analysis.AnalysisStatus(normalized.Status))
}

func runModelValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view model validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "model JSON input file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model validate does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *input == "" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model validate requires --input", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	value, err := readModelFile(*input)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if err := model.Validate(value); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	return writeJSON(stdout, struct {
		Valid   bool   `json:"valid"`
		ModelID string `json:"model_id"`
	}{Valid: true, ModelID: value.ModelID})
}

func runModelProjection(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view model projection", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "model JSON input file")
	output := fs.String("output", "", "projection JSON output file, or - for stdout")
	var selectedPath stringList
	fs.Var(&selectedPath, "path", "hierarchy segment; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model projection does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *input == "" || *output == "" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model projection requires --input and --output", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	value, err := readModelFile(*input)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	projection, err := model.BuildHierarchyProjection(value, []string(selectedPath))
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *output == "-" {
		return writeJSON(stdout, projection)
	}
	if err := writeFileJSON(*output, projection); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	return 0
}

func readModelFile(path string) (model.Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Model{}, analysis.WrapHostError(analysis.ErrInvalidModel, "model JSON input could not be read", err, map[string]any{"input": path})
	}
	var value model.Model
	if err := json.Unmarshal(data, &value); err != nil {
		return model.Model{}, analysis.WrapHostError(analysis.ErrInvalidModel, "model JSON input is invalid", err, map[string]any{"input": path})
	}
	return value, nil
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
	_, _ = fmt.Fprintln(writer, "arch-view open --model <model.json> [--port <n>]")
	_, _ = fmt.Fprintln(writer, "arch-view open --project <path> [--language <id>] [--module <path>] [--port <n>]")
	_, _ = fmt.Fprintln(writer, "arch-view model normalize --input <analysis-json> --output <model-json>")
	_, _ = fmt.Fprintln(writer, "arch-view model validate --input <model-json>")
	_, _ = fmt.Fprintln(writer, "arch-view model projection --input <model-json> [--path <segment>] --output <projection-json>")
}
