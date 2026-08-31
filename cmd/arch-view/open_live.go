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
	"path/filepath"
	"strings"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/live"
	"github.com/buffo/arch-view/internal/quality"
	qualitypolicy "github.com/buffo/arch-view/internal/quality/policy"
	"github.com/buffo/arch-view/internal/viewer"
)

func runOpenLive(base *analysis.Host, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view open --live", flag.ContinueOnError)
	fs.SetOutput(stderr)
	project := fs.String("project", "", "project root to analyze")
	_ = fs.Bool("live", false, "keep a watched live session current while the viewer is open")
	port := fs.Int("port", 0, "loopback TCP port; 0 chooses an available port")
	sessionID := fs.String("session-id", "", "opaque live session identity")
	qualityProfilePath := fs.String("quality-profile", "", "quality profile JSON file")
	noSourceIndex := fs.Bool("no-source-index", false, "disable source-index extraction")
	noWatch := fs.Bool("no-watch", false, "disable filesystem watching")
	analyzerRuntime := fs.String("analyzer-runtime", analyzerRuntimeAuto, "analyzer runtime: auto, packaged, in-process, or explicit")
	allowUntrustedPlugin := fs.Bool("allow-untrusted-plugin", false, "allow an explicitly supplied local descriptor")
	language := fs.String("language", "", "explicit analyzer language")
	analyzerID := fs.String("analyzer", "", "explicit analyzer id")
	module := fs.String("module", "", "explicit Go module path or workspace-relative directory")
	crate := fs.String("crate", "", "explicit Rust package name or workspace-relative crate directory")
	target := fs.String("target", "", "explicit Rust target triple or target selector")
	tsConfig := fs.String("config", "", "explicit TypeScript tsconfig path")
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
	var buildTags stringList
	var features stringList
	var excludes stringList
	var sourceRoots stringList
	var plugins stringList
	fs.Var(&buildTags, "build-tag", "explicit Go build tag; repeatable")
	fs.Var(&features, "feature", "explicit Rust Cargo feature; repeatable")
	fs.Var(&excludes, "exclude", "repository-relative exclusion glob; repeatable")
	fs.Var(&sourceRoots, "source-root", "explicit analyzer source root; repeatable")
	fs.Var(&plugins, "plugin", "external analyzer descriptor; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*project) == "" {
		writeError(stderr, analysis.NewHostError(analysis.ErrInvalidRequest, "open --live requires --project and accepts no positional arguments", nil))
		return 2
	}
	if *port < 0 || *port > 65535 {
		writeError(stderr, analysis.NewHostError(analysis.ErrInvalidRequest, "open --live --port must be between 0 and 65535", nil))
		return 2
	}
	root, err := filepath.Abs(*project)
	if err != nil {
		writeError(stderr, err)
		return 1
	}
	resolver, selectedProfile, err := liveProfileResolver(root, *qualityProfilePath)
	if err != nil {
		writeError(stderr, err)
		return 1
	}
	catalog := quality.NewDefaultCatalog()
	store, err := qualitypolicy.NewFileStore(root)
	if err != nil {
		writeError(stderr, err)
		return 1
	}
	configuredHost, _, err := configureCommandHost(base, commandRuntimeOptions{Mode: *analyzerRuntime, ModeProvided: runtimeModeWasProvided(fs), AllowUntrustedPlugin: *allowUntrustedPlugin, PluginPaths: []string(plugins), AnalyzerID: *analyzerID, Language: *language})
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	cliOptions := collectAnalyzerCLIOptions(fs, analyzerCLIFlags{module: module, crate: crate, features: &features, target: target, config: tsConfig, includeJS: includeJS, includeTests: includeTests, includeExamples: includeExamples, includeGenerated: includeGenerated, includeExternal: includeExternal, safeMode: safeMode, buildTags: &buildTags, excludes: &excludes, sourceRoots: &sourceRoots, pythonVersion: pythonVersion, includeStubs: includeStubs, platform: platform, runtime: runtime})
	analyzerIDs, err := liveAnalyzerSelection(configuredHost, *analyzerID, *language)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *sessionID == "" {
		*sessionID = "viewer-" + time.Now().UTC().Format("20060102t150405.000000000")
		*sessionID = strings.ReplaceAll(*sessionID, ".", "-")
	}
	configValue := live.LiveSessionConfig{SchemaVersion: live.LiveSchemaVersion, SessionID: *sessionID, RepositoryRoot: ".", WatchRoots: []live.WatchRoot{{Path: ".", Recursive: true}}, AnalyzerIDs: analyzerIDs, SourceIndexRequest: live.SourceIndexRequest{Enabled: !*noSourceIndex}, WatchPolicy: live.WatchPolicy{DebounceMS: 250, MaxParallelScopes: live.DefaultMaxParallelScopes, RescanIntervalMS: 30000}, FreshnessPolicy: live.FreshnessPolicy{DefaultConsistency: live.ConsistencyLatestReady, SettleMS: 250, MaxWaitMS: int(live.DefaultFreshnessMaxWait / time.Millisecond), MaxStabilityRetries: live.DefaultStabilityRetries, ReconcileIntervalMS: 30000}, PermissionPolicy: live.PermissionPolicy{DefaultMode: "read_only", Transport: live.TransportLocalHTTP, AllowedOperations: []live.Operation{live.OperationStatus, live.OperationSearch, live.OperationEvidence, live.OperationSourceContext, live.OperationEnsureCurrent, live.OperationQualityEvaluate, live.OperationQualityProfileRead, live.OperationBaselineRead}, AuditPolicyWrites: true}}
	if selectedProfile != nil {
		configValue.QualityRequest = &live.QualityRequest{ProfileID: selectedProfile.ProfileID, ProfileVersion: selectedProfile.ProfileVersion}
	}
	options := live.SessionOptions{AnalyzerRegistry: configuredHost.RegistrySnapshot(), AnalyzerOptionsByID: splitOptionsByAnalyzer(configuredHost.RegistrySnapshot(), cliOptions), QualityCatalog: catalog, Profiles: resolver, PolicyService: live.NewFileQualityPolicyService(qualitypolicy.NewService(store, catalog)), StartWatcher: !*noWatch}
	liveScanner := live.NewMultiAnalyzerScanner(configuredHost, catalog, resolver)
	liveScanner.AnalyzerOptionsByID = splitOptionsByAnalyzer(configuredHost.RegistrySnapshot(), cliOptions)
	options.Scanner = liveScanner
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	session, err := live.StartLiveSession(ctx, configValue, root, options)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	defer func() { _ = session.Close() }()
	server, err := viewer.NewLiveServer(session, viewer.ServerOptions{SourceRoot: root})
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:"+fmt.Sprint(*port))
	if err != nil {
		writeError(stderr, err)
		return 1
	}
	defer func() { _ = listener.Close() }()
	_, _ = fmt.Fprintf(stdout, "arch-view live viewer listening at http://%s/\n", listener.Addr().String())
	_, _ = fmt.Fprintln(stdout, "Live session:", *sessionID)
	_, _ = fmt.Fprintln(stdout, "Press Ctrl+C to stop.")
	httpServer := &http.Server{Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownContext)
	}()
	if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		writeError(stderr, err)
		return 1
	}
	return 0
}
