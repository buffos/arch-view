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
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
	"github.com/buffo/arch-view/internal/viewer"
)

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
	var viewerOptions viewer.ServerOptions
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
		result, err := host.Run(ctx, analysis.RunRequest{ProjectRoot: *project, Language: *language, AnalyzerID: *analyzerID, CLIOptions: cliOptions, ProjectOptions: map[string]any{}})
		if err != nil {
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
		value, err = canonical.Normalize(result)
		if err != nil {
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
		baseOptions := make(map[string]any, len(cliOptions))
		for key, option := range cliOptions {
			baseOptions[key] = option
		}
		viewerOptions = viewer.ServerOptions{
			SourceRoot: *project,
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
					languageValue = *language
				}
				result, err := host.Run(reanalysisContext, analysis.RunRequest{ProjectRoot: request.ProjectRoot, Language: languageValue, AnalyzerID: *analyzerID, CLIOptions: options, ProjectOptions: map[string]any{}})
				if err != nil {
					return model.Model{}, err
				}
				return canonical.Normalize(result)
			},
		}
	}
	server, err := viewer.NewServer(value, viewerOptions)
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
