package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/orchestration"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/viewer"
)

func runOpen(host *analysis.Host, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view open", flag.ContinueOnError)
	fs.SetOutput(stderr)
	modelInput := fs.String("model", "", "canonical model JSON input file")
	project := fs.String("project", "", "project root to analyze before opening")
	analyzerRuntime := fs.String("analyzer-runtime", analyzerRuntimeAuto, "analyzer runtime: auto, packaged, in-process, or explicit")
	allowUntrustedPlugin := fs.Bool("allow-untrusted-plugin", false, "allow an explicitly supplied local descriptor")
	language := fs.String("language", "", "explicit analyzer language")
	analyzerID := fs.String("analyzer", "", "explicit analyzer id")
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
	port := fs.Int("port", 0, "loopback TCP port; 0 chooses an available port")
	var buildTags stringList
	var features stringList
	var excludes stringList
	var sourceRoots stringList
	var plugins stringList
	fs.Var(&buildTags, "build-tag", "explicit Go build tag; repeatable")
	fs.Var(&features, "feature", "explicit Rust Cargo feature; repeatable")
	fs.Var(&features, "features", "explicit Rust Cargo feature; repeatable")
	fs.Var(&excludes, "exclude", "repository-relative exclusion glob; repeatable")
	fs.Var(&sourceRoots, "source-root", "explicit analyzer source root; repeatable")
	fs.Var(&plugins, "plugin", "external analyzer descriptor; repeatable")
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
	if *modelInput != "" && len(plugins) > 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "open --plugin requires --project", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var err error
	var value model.Model
	var viewerOptions viewer.ServerOptions
	var aggregateRun *orchestration.AnalysisRun
	if *modelInput != "" {
		value, err = readModelFile(*modelInput)
		if err != nil {
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
	} else {
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
		if *analyzerID == "" && *language == "" {
			run, combinedErr := runCombinedAnalysis(ctx, host, *project, cliOptions)
			if combinedErr != nil {
				writeError(stderr, combinedErr)
				return analysis.ExitCodeForError(combinedErr)
			}
			aggregateRun = &run
			viewerOptions = projectCombinedViewerOptions(host, *project, cliOptions)
		} else {
			result, runErr := host.Run(ctx, analysis.RunRequest{ProjectRoot: *project, Language: *language, AnalyzerID: *analyzerID, CLIOptions: cliOptions, ProjectOptions: map[string]any{}})
			if runErr != nil {
				writeError(stderr, runErr)
				return analysis.ExitCodeForError(runErr)
			}
			value, runErr = canonical.Normalize(result)
			if runErr != nil {
				writeError(stderr, runErr)
				return analysis.ExitCodeForError(runErr)
			}
			viewerOptions = projectViewerOptions(host, *project, *language, *analyzerID, cliOptions)
		}
	}
	var server *viewer.Server
	if aggregateRun != nil {
		server, err = viewer.NewAggregateServer(*aggregateRun, viewerOptions)
	} else {
		server, err = viewer.NewServer(value, viewerOptions)
	}
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
	httpServer := &http.Server{Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second}
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

func projectViewerOptions(host *analysis.Host, project, language, analyzerID string, cliOptions map[string]any) viewer.ServerOptions {
	baseOptions := make(map[string]any, len(cliOptions))
	for key, option := range cliOptions {
		baseOptions[key] = option
	}
	return viewer.ServerOptions{
		SourceRoot: project,
		Reanalyze: func(reanalysisContext context.Context, request viewer.ReanalysisRequest) (model.Model, error) {
			options := make(map[string]any, len(baseOptions)+len(request.Options))
			for key, option := range baseOptions {
				options[key] = option
			}
			for key, option := range request.Options {
				options[key] = option
			}
			languageValue := request.Language
			if languageValue == "" {
				languageValue = language
			}
			result, err := host.Run(reanalysisContext, analysis.RunRequest{ProjectRoot: request.ProjectRoot, Language: languageValue, AnalyzerID: analyzerID, CLIOptions: options, ProjectOptions: map[string]any{}})
			if err != nil {
				return model.Model{}, err
			}
			return canonical.Normalize(result)
		},
	}
}

func projectCombinedViewerOptions(host *analysis.Host, project string, cliOptions map[string]any) viewer.ServerOptions {
	baseOptions := make(map[string]any, len(cliOptions))
	for key, option := range cliOptions {
		baseOptions[key] = option
	}
	analyze := func(ctx context.Context, request viewer.CombinedAnalysisRequest) (orchestration.AnalysisRun, error) {
		options := make(map[string]any, len(baseOptions)+len(request.CLIOptions))
		for key, option := range baseOptions {
			options[key] = option
		}
		for key, option := range request.CLIOptions {
			options[key] = option
		}
		root := request.ProjectRoot
		if root == "" {
			root = project
		}
		return runCombinedAnalysisWithPolicy(ctx, host, root, options, request.SourceScopePolicy)
	}
	return viewer.ServerOptions{
		SourceRoot:        project,
		AnalyzeCombined:   analyze,
		ReanalyzeCombined: analyze,
	}
}
