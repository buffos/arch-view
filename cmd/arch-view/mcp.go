package main

import (
	"context"
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
	"strconv"
	"strings"
	"time"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/live"
	"github.com/buffo/arch-view/internal/quality"
	qualitypolicy "github.com/buffo/arch-view/internal/quality/policy"
)

func runMCPCommand(base *analysis.Host, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	project := fs.String("project", "", "project root to analyze")
	transport := fs.String("transport", live.TransportStdio, "transport: stdio, local_http, or authenticated_http")
	port := fs.Int("port", 0, "loopback HTTP port for an HTTP transport; 0 chooses an available port")
	authToken := fs.String("auth-token", "", "required bearer token for authenticated_http")
	var allowedOrigins stringList
	fs.Var(&allowedOrigins, "origin", "allowed browser Origin for HTTP; repeatable")
	sessionID := fs.String("session-id", "", "opaque live session identity")
	qualityProfilePath := fs.String("quality-profile", "", "quality profile JSON file")
	noSourceIndex := fs.Bool("no-source-index", false, "disable source-index extraction")
	noWatch := fs.Bool("no-watch", false, "disable filesystem watching")
	allowPolicyWrites := fs.Bool("allow-policy-writes", false, "allow explicitly authorized quality profile/baseline writes")
	policyToken := fs.String("policy-token", "", "opaque authorization accepted for policy writes")
	analyzerRuntime := fs.String("analyzer-runtime", analyzerRuntimeAuto, "analyzer runtime: auto, packaged, in-process, or explicit")
	allowUntrustedPlugin := fs.Bool("allow-untrusted-plugin", false, "allow an explicitly supplied local descriptor")
	var plugins stringList
	fs.Var(&plugins, "plugin", "external analyzer descriptor; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*project) == "" {
		_, _ = fmt.Fprintln(stderr, "arch-view mcp requires --project and accepts no positional arguments")
		return 2
	}
	if *allowPolicyWrites && strings.TrimSpace(*policyToken) == "" {
		_, _ = fmt.Fprintln(stderr, "--allow-policy-writes requires --policy-token")
		return 2
	}
	transportValue := strings.ToLower(strings.TrimSpace(*transport))
	if transportValue != live.TransportStdio && transportValue != live.TransportLocalHTTP && transportValue != live.TransportAuthenticatedHTTP {
		_, _ = fmt.Fprintf(stderr, "unsupported MCP transport %q; use stdio, local_http, or authenticated_http\n", *transport)
		return 2
	}
	if *port < 0 || *port > 65535 {
		_, _ = fmt.Fprintln(stderr, "--port must be between 0 and 65535")
		return 2
	}
	if transportValue == live.TransportAuthenticatedHTTP && strings.TrimSpace(*authToken) == "" {
		_, _ = fmt.Fprintln(stderr, "--auth-token is required for authenticated_http")
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
	configuredHost, _, err := configureCommandHost(base, commandRuntimeOptions{Mode: *analyzerRuntime, ModeProvided: runtimeModeWasProvided(fs), AllowUntrustedPlugin: *allowUntrustedPlugin, PluginPaths: []string(plugins)})
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if strings.TrimSpace(*sessionID) == "" {
		*sessionID = "mcp-" + time.Now().UTC().Format("20060102t150405.000000000")
		*sessionID = strings.ReplaceAll(*sessionID, ".", "-")
	}
	allowed := []live.Operation{live.OperationStatus, live.OperationSearch, live.OperationEvidence, live.OperationSourceContext, live.OperationEnsureCurrent, live.OperationQualityEvaluate, live.OperationQualityProfileRead, live.OperationBaselineRead}
	if *allowPolicyWrites {
		allowed = append(allowed, live.OperationQualityProfileWrite, live.OperationBaselineWrite)
	}
	config := live.LiveSessionConfig{SchemaVersion: live.LiveSchemaVersion, SessionID: *sessionID, RepositoryRoot: ".", WatchRoots: []live.WatchRoot{{Path: ".", Recursive: true}}, SourceIndexRequest: live.SourceIndexRequest{Enabled: !*noSourceIndex}, WatchPolicy: live.WatchPolicy{DebounceMS: 250, MaxParallelScopes: live.DefaultMaxParallelScopes, RescanIntervalMS: 30000}, FreshnessPolicy: live.FreshnessPolicy{DefaultConsistency: live.ConsistencyLatestReady, SettleMS: 250, MaxWaitMS: int(live.DefaultFreshnessMaxWait / time.Millisecond), MaxStabilityRetries: live.DefaultStabilityRetries, ReconcileIntervalMS: 30000}, PermissionPolicy: live.PermissionPolicy{DefaultMode: "read_only", AllowedOperations: allowed, Transport: transportValue, AuditPolicyWrites: true}}
	if selectedProfile != nil {
		config.QualityRequest = &live.QualityRequest{ProfileID: selectedProfile.ProfileID, ProfileVersion: selectedProfile.ProfileVersion}
	}
	options := live.SessionOptions{AnalyzerRegistry: configuredHost.RegistrySnapshot(), QualityCatalog: catalog, Profiles: resolver, PolicyService: live.NewFileQualityPolicyService(qualitypolicy.NewService(store, catalog)), StartWatcher: !*noWatch}
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
	session, err := live.StartLiveSession(ctx, config, root, options)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	defer func() { _ = session.Close() }()
	if transportValue != live.TransportStdio {
		return serveMCPHTTP(ctx, session, transportValue, *authToken, []string(allowedOrigins), *port, stdout, stderr)
	}
	returnCode := live.NewMCPServer(live.NewLocalQueryAdapter(session)).Serve(ctx, os.Stdin, stdout)
	if returnCode != nil && !errors.Is(returnCode, context.Canceled) {
		writeError(stderr, returnCode)
		return 1
	}
	return 0
}

func serveMCPHTTP(ctx context.Context, session *live.LiveSession, transport, authToken string, origins []string, port int, stdout, stderr io.Writer) int {
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		writeError(stderr, analysis.WrapHostError(analysis.ErrHostFailure, "MCP HTTP transport could not bind its loopback port", err, nil))
		return 1
	}
	defer func() { _ = listener.Close() }()
	sessionID := session.Config().SessionID
	endpoint := "http://" + listener.Addr().String() + "/v1/live/" + url.PathEscape(sessionID)
	server := live.NewHTTPServer(live.NewLocalQueryAdapter(session), live.HTTPServerOptions{Transport: transport, AuthToken: authToken, AllowedOrigins: origins, SessionID: sessionID})
	httpServer := &http.Server{Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second}
	if err := writeCommandJSON(stdout, map[string]any{"schema_version": liveCLIResponseSchema, "status": "started", "transport": transport, "session_id": sessionID, "endpoint": endpoint, "status_endpoint": endpoint + "/status"}); err != nil {
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "Press Ctrl+C to stop.")
	go func() {
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownContext)
	}()
	if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		writeError(stderr, analysis.WrapHostError(analysis.ErrHostFailure, "MCP HTTP transport failed", err, nil))
		return 1
	}
	return 0
}
