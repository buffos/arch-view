package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/live"
	"github.com/buffo/arch-view/internal/quality"
	qualitypolicy "github.com/buffo/arch-view/internal/quality/policy"
)

const liveCLIResponseSchema = "arch-view.live-cli/v1"

func runLiveCommand(host *analysis.Host, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printLiveUsage(stdout)
		return 2
	}
	switch args[0] {
	case "start":
		return runLiveStart(host, args[1:], stdout, stderr)
	case "status":
		return runLiveStatus(args[1:], stdout, stderr)
	case "wait":
		return runLiveWait(args[1:], false, stdout, stderr)
	case "ensure-current":
		return runLiveWait(args[1:], true, stdout, stderr)
	default:
		printLiveUsage(stderr)
		return 2
	}
}

func printLiveUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, "arch-view live start --project <path> [--port <n>] [--session-id <id>] [--quality-profile <profile.json>] [--no-source-index]")
	_, _ = fmt.Fprintln(writer, "arch-view live status --endpoint <http://127.0.0.1:port/v1/live/<session-id>>")
	_, _ = fmt.Fprintln(writer, "arch-view live wait --endpoint <url> [--consistency latest_ready|require_current] [--timeout <duration>]")
	_, _ = fmt.Fprintln(writer, "arch-view live ensure-current --endpoint <url> [--timeout <duration>]")
}

func runLiveStart(base *analysis.Host, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view live start", flag.ContinueOnError)
	fs.SetOutput(stderr)
	project := fs.String("project", "", "project root to analyze")
	sessionID := fs.String("session-id", "", "opaque live session identity")
	port := fs.Int("port", 0, "loopback TCP port; 0 chooses an available port")
	qualityProfilePath := fs.String("quality-profile", "", "quality profile JSON file")
	noSourceIndex := fs.Bool("no-source-index", false, "disable source-index extraction")
	noWatch := fs.Bool("no-watch", false, "disable filesystem watching")
	allowPolicyWrites := fs.Bool("allow-policy-writes", false, "allow explicitly authorized quality profile/baseline writes")
	policyToken := fs.String("policy-token", "", "opaque authorization accepted for policy writes")
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
	var buildTags stringList
	var features stringList
	var excludes stringList
	var sourceRoots stringList
	var plugins stringList
	var watchRoots stringList
	fs.Var(&buildTags, "build-tag", "explicit Go build tag; repeatable")
	fs.Var(&features, "feature", "explicit Rust Cargo feature; repeatable")
	fs.Var(&excludes, "exclude", "repository-relative exclusion glob; repeatable")
	fs.Var(&sourceRoots, "source-root", "explicit analyzer source root; repeatable")
	fs.Var(&plugins, "plugin", "external analyzer descriptor; repeatable")
	fs.Var(&watchRoots, "watch-root", "repository-relative live watch root; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		writeError(stderr, analysis.NewHostError(analysis.ErrInvalidRequest, "live start does not accept positional arguments", nil))
		return 2
	}
	if strings.TrimSpace(*project) == "" {
		writeError(stderr, analysis.NewHostError(analysis.ErrInvalidRequest, "live start requires --project", nil))
		return 2
	}
	if *port < 0 || *port > 65535 {
		writeError(stderr, analysis.NewHostError(analysis.ErrInvalidRequest, "live start --port must be between 0 and 65535", nil))
		return analysis.ExitCodeForError(analysis.NewHostError(analysis.ErrInvalidRequest, "invalid port", nil))
	}
	if *allowPolicyWrites && strings.TrimSpace(*policyToken) == "" {
		writeError(stderr, analysis.NewHostError(analysis.ErrInvalidRequest, "--allow-policy-writes requires --policy-token", nil))
		return analysis.ExitCodeForError(analysis.NewHostError(analysis.ErrInvalidRequest, "policy token is required", nil))
	}
	projectRoot, err := filepath.Abs(*project)
	if err != nil {
		writeError(stderr, err)
		return 2
	}
	profileResolver, selectedProfile, err := liveProfileResolver(projectRoot, *qualityProfilePath)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	catalog := quality.NewDefaultCatalog()
	policyStore, err := qualitypolicy.NewFileStore(projectRoot)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	policyService := qualitypolicy.NewService(policyStore, catalog)
	configuredHost, _, err := configureCommandHost(base, commandRuntimeOptions{Mode: *analyzerRuntime, ModeProvided: runtimeModeWasProvided(fs), AllowUntrustedPlugin: *allowUntrustedPlugin, PluginPaths: []string(plugins), AnalyzerID: *analyzerID, Language: *language})
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	cliOptions := collectAnalyzerCLIOptions(fs, analyzerCLIFlags{module: module, crate: crate, features: &features, target: target, config: config, includeJS: includeJS, includeTests: includeTests, includeExamples: includeExamples, includeGenerated: includeGenerated, includeExternal: includeExternal, safeMode: safeMode, buildTags: &buildTags, excludes: &excludes, sourceRoots: &sourceRoots, pythonVersion: pythonVersion, includeStubs: includeStubs, platform: platform, runtime: runtime})
	analyzerIDs, err := liveAnalyzerSelection(configuredHost, *analyzerID, *language)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *sessionID == "" {
		*sessionID = "live-" + time.Now().UTC().Format("20060102t150405.000000000")
		*sessionID = strings.ReplaceAll(*sessionID, ".", "-")
	}
	roots := []live.WatchRoot{}
	for _, root := range watchRoots {
		roots = append(roots, live.WatchRoot{Path: root, Recursive: true})
	}
	if len(roots) == 0 {
		roots = []live.WatchRoot{{Path: ".", Recursive: true}}
	}
	allowed := []live.Operation{live.OperationStatus, live.OperationSearch, live.OperationEvidence, live.OperationSourceContext, live.OperationEnsureCurrent, live.OperationQualityEvaluate, live.OperationQualityProfileRead, live.OperationBaselineRead}
	if *allowPolicyWrites {
		allowed = append(allowed, live.OperationQualityProfileWrite, live.OperationBaselineWrite)
	}
	configValue := live.LiveSessionConfig{SchemaVersion: live.LiveSchemaVersion, SessionID: *sessionID, RepositoryRoot: ".", WatchRoots: roots, AnalyzerIDs: analyzerIDs, SourceIndexRequest: live.SourceIndexRequest{Enabled: !*noSourceIndex}, WatchPolicy: live.WatchPolicy{DebounceMS: 250, MaxParallelScopes: live.DefaultMaxParallelScopes, RescanIntervalMS: 30000}, FreshnessPolicy: live.FreshnessPolicy{DefaultConsistency: live.ConsistencyLatestReady, SettleMS: 250, MaxWaitMS: int(live.DefaultFreshnessMaxWait / time.Millisecond), MaxStabilityRetries: live.DefaultStabilityRetries, ReconcileIntervalMS: 30000}, PermissionPolicy: live.PermissionPolicy{DefaultMode: "read_only", AllowedOperations: allowed, Transport: live.TransportLocalHTTP, AuditPolicyWrites: true}}
	if selectedProfile != nil {
		configValue.QualityRequest = &live.QualityRequest{ProfileID: selectedProfile.ProfileID, ProfileVersion: selectedProfile.ProfileVersion}
	}
	options := live.SessionOptions{AnalyzerRegistry: configuredHost.RegistrySnapshot(), AnalyzerOptionsByID: splitOptionsByAnalyzer(configuredHost.RegistrySnapshot(), cliOptions), QualityCatalog: catalog, Profiles: profileResolver, PolicyService: live.NewFileQualityPolicyService(policyService), StartWatcher: !*noWatch}
	liveScanner := live.NewMultiAnalyzerScanner(configuredHost, catalog, profileResolver)
	liveScanner.AnalyzerOptionsByID = splitOptionsByAnalyzer(configuredHost.RegistrySnapshot(), cliOptions)
	options.Scanner = liveScanner
	if *allowPolicyWrites {
		token := *policyToken
		options.PolicyAuthorizer = live.PolicyAuthorizerFunc(func(_ context.Context, value live.PolicyAuthorization) error {
			if value.Authorization != token {
				return errors.New("authorization token does not match the configured local grant")
			}
			return nil
		})
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	session, err := live.StartLiveSession(ctx, configValue, projectRoot, options)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	defer func() { _ = session.Close() }()
	listener, err := net.Listen("tcp", "127.0.0.1:"+fmt.Sprint(*port))
	if err != nil {
		writeError(stderr, analysis.WrapHostError(analysis.ErrHostFailure, "live session could not bind its loopback port", err, nil))
		return analysis.ExitCodeForError(err)
	}
	defer func() { _ = listener.Close() }()
	endpoint := "http://" + listener.Addr().String() + "/v1/live/" + url.PathEscape(*sessionID)
	server := live.NewHTTPServer(live.NewLocalQueryAdapter(session), live.HTTPServerOptions{Transport: live.TransportLocalHTTP, SessionID: *sessionID})
	httpServer := &http.Server{Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second}
	result := map[string]any{"schema_version": liveCLIResponseSchema, "status": "started", "session_id": *sessionID, "endpoint": endpoint, "status_endpoint": endpoint + "/status"}
	if err := writeCommandJSON(stdout, result); err != nil {
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "Press Ctrl+C to stop.")
	go func() {
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownContext)
		_ = session.Close()
	}()
	if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		writeError(stderr, analysis.WrapHostError(analysis.ErrHostFailure, "live session HTTP bridge failed", err, nil))
		return analysis.ExitCodeForError(err)
	}
	return 0
}

func liveAnalyzerSelection(host *analysis.Host, analyzerID, language string) ([]string, error) {
	analyzerID = strings.TrimSpace(analyzerID)
	language = strings.TrimSpace(language)
	if analyzerID == "" && language == "" {
		return nil, nil
	}
	if host == nil {
		return nil, analysis.NewHostError(analysis.ErrHostFailure, "live analyzer selection has no host", nil)
	}
	manifests := host.ListManifests()
	if analyzerID != "" {
		for _, manifest := range manifests {
			if manifest.ID != analyzerID {
				continue
			}
			if language != "" && !strings.EqualFold(manifest.Language, language) {
				return nil, analysis.NewHostError(analysis.ErrInvalidRequest, "--analyzer and --language select different analyzers", map[string]any{"analyzer": analyzerID, "language": language})
			}
			return []string{manifest.ID}, nil
		}
		return nil, analysis.NewHostError(analysis.ErrNoAnalyzer, "the requested live analyzer is unavailable", map[string]any{"analyzer_id": analyzerID})
	}
	selected := make([]string, 0)
	for _, manifest := range manifests {
		if strings.EqualFold(manifest.Language, language) {
			selected = append(selected, manifest.ID)
		}
	}
	if len(selected) == 0 {
		return nil, analysis.NewHostError(analysis.ErrNoAnalyzer, "no live analyzer is available for the requested language", map[string]any{"language": language})
	}
	sort.Strings(selected)
	return selected, nil
}

func runLiveStatus(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view live status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	endpoint := fs.String("endpoint", "", "live session endpoint returned by live start")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*endpoint) == "" {
		writeError(stderr, analysis.NewHostError(analysis.ErrInvalidRequest, "live status requires --endpoint", nil))
		return 2
	}
	data, status, err := callLiveEndpoint(context.Background(), liveEndpointURL(*endpoint, "status"), http.MethodGet, nil, 10*time.Second)
	if err != nil {
		writeError(stderr, err)
		return 1
	}
	if err := writeCommandJSON(stdout, map[string]any{"http_status": status, "response": json.RawMessage(data)}); err != nil {
		return 1
	}
	return 0
}

func runLiveWait(args []string, forceCurrent bool, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view live wait", flag.ContinueOnError)
	fs.SetOutput(stderr)
	endpoint := fs.String("endpoint", "", "live session endpoint returned by live start")
	consistency := fs.String("consistency", string(live.ConsistencyLatestReady), "latest_ready or require_current")
	timeout := fs.Duration("timeout", 30*time.Second, "maximum wait duration")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*endpoint) == "" {
		writeError(stderr, analysis.NewHostError(analysis.ErrInvalidRequest, "live wait requires --endpoint", nil))
		return 2
	}
	selected := live.Consistency(*consistency)
	if forceCurrent {
		selected = live.ConsistencyRequireCurrent
	}
	if selected != live.ConsistencyLatestReady && selected != live.ConsistencyRequireCurrent {
		writeError(stderr, analysis.NewHostError(analysis.ErrInvalidRequest, "live wait --consistency must be latest_ready or require_current", nil))
		return 2
	}
	if *timeout <= 0 {
		writeError(stderr, analysis.NewHostError(analysis.ErrInvalidRequest, "live wait --timeout must be positive", nil))
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	data, status, err := waitForLiveEndpoint(ctx, *endpoint, selected)
	if err != nil {
		writeError(stderr, err)
		return 1
	}
	if err := writeCommandJSON(stdout, map[string]any{"http_status": status, "response": json.RawMessage(data)}); err != nil {
		return 1
	}
	if status >= 400 {
		return 1
	}
	return 0
}

func waitForLiveEndpoint(ctx context.Context, endpoint string, consistency live.Consistency) ([]byte, int, error) {
	if consistency == live.ConsistencyRequireCurrent {
		return callLiveEndpointContext(ctx, liveEndpointURL(endpoint, "ensure-current"), http.MethodPost, []byte(`{"consistency":"require_current"}`))
	}
	statusEndpoint := liveEndpointURL(endpoint, "status")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		data, status, err := callLiveEndpointContext(ctx, statusEndpoint, http.MethodGet, nil)
		if err != nil {
			return nil, status, err
		}
		if status >= http.StatusBadRequest {
			return data, status, nil
		}
		var envelope struct {
			Revision            int    `json:"revision"`
			ReturnedConsistency string `json:"returned_consistency"`
			Freshness           struct {
				Status string `json:"status"`
			} `json:"freshness"`
			Result struct {
				State string `json:"state"`
			} `json:"result"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			return nil, status, fmt.Errorf("live status response is invalid JSON: %w", err)
		}
		if envelope.Revision > 0 && envelope.ReturnedConsistency != "unavailable" {
			return data, status, nil
		}
		if envelope.Result.State == string(live.SessionFailed) || envelope.Freshness.Status == string(live.FreshnessFailed) {
			return data, http.StatusConflict, nil
		}
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		case <-ticker.C:
		}
	}
}

// liveEndpointURL accepts either the session endpoint printed by live start or
// one of its status/ensure-current URLs. This makes the CLI safe to use with
// either value copied from the start response.
func liveEndpointURL(endpoint, suffix string) string {
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	for _, knownSuffix := range []string{"/status", "/ensure-current"} {
		if strings.HasSuffix(base, knownSuffix) {
			base = strings.TrimRight(strings.TrimSuffix(base, knownSuffix), "/")
			break
		}
	}
	return base + "/" + strings.TrimLeft(suffix, "/")
}

func callLiveEndpoint(ctx context.Context, endpoint, method string, body []byte, timeout time.Duration) ([]byte, int, error) {
	callContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return callLiveEndpointContext(callContext, endpoint, method, body)
}

func callLiveEndpointContext(ctx context.Context, endpoint, method string, body []byte) ([]byte, int, error) {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(response.Body, live.DefaultHTTPResponseBytes+1))
	if err != nil {
		return nil, response.StatusCode, err
	}
	if len(data) > live.DefaultHTTPResponseBytes {
		return nil, response.StatusCode, fmt.Errorf("live endpoint response exceeds the %d byte limit", live.DefaultHTTPResponseBytes)
	}
	return data, response.StatusCode, nil
}

func writeCommandJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

type commandProfileResolver struct {
	store  *qualitypolicy.FileStore
	loaded map[string]quality.QualityProfile
}

func (resolver *commandProfileResolver) ResolveProfile(ctx context.Context, id, version string) (quality.QualityProfile, error) {
	if resolver != nil {
		if profile, ok := resolver.loaded[id+"\x00"+version]; ok {
			return profile, nil
		}
		if resolver.store != nil {
			return resolver.store.ResolveProfile(ctx, id, version)
		}
	}
	return quality.QualityProfile{}, fmt.Errorf("quality profile %s@%s was not found", id, version)
}

func (resolver *commandProfileResolver) ListProfiles(ctx context.Context) ([]live.QualityProfileInfo, error) {
	if resolver == nil {
		return nil, errors.New("quality profile resolver is unavailable")
	}
	result := []live.QualityProfileInfo{}
	seen := map[string]struct{}{}
	if resolver.store != nil {
		values, err := resolver.store.ListProfiles(ctx)
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			key := value.ProfileID + "\x00" + value.ProfileVersion
			seen[key] = struct{}{}
			result = append(result, live.QualityProfileInfo{ProfileID: value.ProfileID, ProfileVersion: value.ProfileVersion, FileName: value.FileName, Status: value.Status, Reason: value.Reason})
		}
	}
	for key, profile := range resolver.loaded {
		if _, ok := seen[key]; ok {
			continue
		}
		result = append(result, live.QualityProfileInfo{ProfileID: profile.ProfileID, ProfileVersion: profile.ProfileVersion, Status: "available"})
	}
	return result, nil
}

func liveProfileResolver(root, selectedPath string) (*commandProfileResolver, *quality.QualityProfile, error) {
	store, err := qualitypolicy.NewFileStore(root)
	if err != nil {
		return nil, nil, err
	}
	resolver := &commandProfileResolver{store: store, loaded: map[string]quality.QualityProfile{}}
	if strings.TrimSpace(selectedPath) == "" {
		return resolver, nil, nil
	}
	profile, err := readCommandQualityProfile(selectedPath)
	if err != nil {
		return nil, nil, err
	}
	resolver.loaded[profile.ProfileID+"\x00"+profile.ProfileVersion] = profile
	return resolver, &profile, nil
}

func readCommandQualityProfile(path string) (quality.QualityProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return quality.QualityProfile{}, fmt.Errorf("quality profile could not be read: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var profile quality.QualityProfile
	if err := decoder.Decode(&profile); err != nil {
		return quality.QualityProfile{}, fmt.Errorf("quality profile JSON is invalid: %w", err)
	}
	return profile, nil
}
