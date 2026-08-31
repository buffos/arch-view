package live

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	TransportStdio             = "stdio"
	TransportLocalHTTP         = "local_http"
	TransportAuthenticatedHTTP = "authenticated_http"
	DefaultHTTPRequestBytes    = 1 << 20
	DefaultHTTPResponseBytes   = 4 << 20
)

type HTTPServerOptions struct {
	Transport        string
	AuthToken        string
	AllowedOrigins   []string
	MaxRequestBytes  int64
	MaxResponseBytes int
	SessionID        string
}

type HTTPServer struct {
	adapter *LocalQueryAdapter
	options HTTPServerOptions
}

func NewHTTPServer(adapter *LocalQueryAdapter, options HTTPServerOptions) *HTTPServer {
	if options.MaxRequestBytes == 0 {
		options.MaxRequestBytes = DefaultHTTPRequestBytes
	}
	if options.MaxResponseBytes == 0 {
		options.MaxResponseBytes = DefaultHTTPResponseBytes
	}
	if options.Transport == "" {
		options.Transport = TransportLocalHTTP
	}
	return &HTTPServer{adapter: adapter, options: options}
}

func (server *HTTPServer) Handler() http.Handler {
	return http.HandlerFunc(server.serveHTTP)
}

func (server *HTTPServer) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	if server == nil || server.adapter == nil || server.adapter.Session == nil {
		writeLiveHTTPError(writer, http.StatusServiceUnavailable, newLiveError(ErrorLiveConfigInvalid, "live HTTP adapter is unavailable", nil))
		return
	}
	origin, err := server.authorizeRequest(request, request.Method == http.MethodOptions)
	if origin != "" {
		writer.Header().Set("Access-Control-Allow-Origin", origin)
		writer.Header().Set("Vary", "Origin")
		writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Arch-View-Token")
		writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	}
	if err != nil {
		writeLiveHTTPError(writer, http.StatusForbidden, err)
		return
	}
	pathValue := request.URL.EscapedPath()
	if pathValue == "" {
		pathValue = request.URL.Path
	}
	pathParts, ok := splitLivePath(pathValue)
	if !ok || pathParts[0] != server.adapter.Session.validated.Config.SessionID || server.options.SessionID != "" && pathParts[0] != server.options.SessionID {
		writeLiveHTTPError(writer, http.StatusNotFound, newLiveError("mcp_session_not_found", "the requested live session is unavailable", nil))
		return
	}
	if request.Method == http.MethodOptions {
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	if request.Body != nil && server.options.MaxRequestBytes > 0 {
		request.Body = http.MaxBytesReader(writer, request.Body, server.options.MaxRequestBytes)
	}
	if err := server.dispatch(writer, request, pathParts[1:]); err != nil {
		writeLiveHTTPError(writer, httpStatusForLiveError(err), err)
	}
}

func (server *HTTPServer) authorizeRequest(request *http.Request, preflight bool) (string, error) {
	transport := server.options.Transport
	origin, err := server.authorizeOrigin(request)
	if err != nil {
		return "", err
	}
	if transport == TransportAuthenticatedHTTP && !preflight {
		if strings.TrimSpace(server.options.AuthToken) == "" {
			return origin, newLiveError(ErrorQualityPolicyPermissionDenied, "authenticated HTTP transport is not configured with credentials", nil)
		}
		provided := strings.TrimSpace(request.Header.Get("X-Arch-View-Token"))
		if provided == "" {
			provided = strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
		}
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(server.options.AuthToken)) != 1 {
			return origin, newLiveError(ErrorQualityPolicyPermissionDenied, "HTTP authentication failed", nil)
		}
	}
	if transport != TransportLocalHTTP && transport != TransportAuthenticatedHTTP && transport != TransportStdio {
		return "", newLiveError(ErrorLiveConfigInvalid, "HTTP transport mode is unsupported", map[string]any{"transport": transport})
	}
	return origin, nil
}

func (server *HTTPServer) authorizeOrigin(request *http.Request) (string, error) {
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		if len(server.options.AllowedOrigins) > 0 {
			return "", newLiveError(ErrorQualityPolicyPermissionDenied, "HTTP origin is required by this transport", nil)
		}
		return "", nil
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", newLiveError(ErrorQualityPolicyPermissionDenied, "HTTP origin is invalid", map[string]any{"origin": origin})
	}
	if len(server.options.AllowedOrigins) > 0 {
		for _, candidate := range server.options.AllowedOrigins {
			if origin == strings.TrimSpace(candidate) {
				return origin, nil
			}
		}
		return "", newLiveError(ErrorQualityPolicyPermissionDenied, "HTTP origin is not allowed", map[string]any{"origin": origin})
	}
	if !strings.EqualFold(parsed.Host, request.Host) {
		return "", newLiveError(ErrorQualityPolicyPermissionDenied, "cross-origin HTTP requests require an explicit origin allowlist", map[string]any{"origin": origin})
	}
	return origin, nil
}

func (server *HTTPServer) dispatch(writer http.ResponseWriter, request *http.Request, parts []string) error {
	if len(parts) == 0 {
		return newLiveError("mcp_route_not_found", "live HTTP route is missing", nil)
	}
	adapter := server.adapter
	sessionID := adapter.Session.validated.Config.SessionID
	queryRequest := func() (QueryRequest, error) {
		var value QueryRequest
		if request.Method == http.MethodGet {
			value.Consistency = Consistency(request.URL.Query().Get("consistency"))
			value.Revision, _ = strconv.Atoi(request.URL.Query().Get("revision"))
			value.MaxBytes, _ = strconv.Atoi(request.URL.Query().Get("max_bytes"))
			value.MaxItems, _ = strconv.Atoi(request.URL.Query().Get("max_items"))
			value.Cursor = request.URL.Query().Get("cursor")
			value.SessionID = sessionID
			return value, nil
		}
		if err := decodeLiveJSON(request, &value); err != nil {
			return QueryRequest{}, err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		if value.SessionID != sessionID {
			return QueryRequest{}, newLiveError("QueryInvalid", "request session_id does not match the live session", nil)
		}
		return value, nil
	}
	textRequest := func() (TextSearchQuery, error) {
		var value TextSearchQuery
		if err := decodeLiveJSON(request, &value); err != nil {
			return TextSearchQuery{}, err
		}
		if value.Consistency == "" {
			value.Consistency = ConsistencyLatestReady
		}
		return value, nil
	}
	writeEnvelope := func(envelope QueryEnvelope) error {
		return writeLiveEnvelope(writer, http.StatusOK, envelope, server.options.MaxResponseBytes)
	}

	switch parts[0] {
	case "status":
		if request.Method != http.MethodGet {
			return methodError(http.MethodGet)
		}
		value, err := queryRequest()
		if err != nil {
			return err
		}
		envelope, err := adapter.GetSnapshotStatus(request.Context(), value)
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	case "ensure-current":
		if request.Method != http.MethodPost {
			return methodError(http.MethodPost)
		}
		value, err := queryRequest()
		if err != nil {
			return err
		}
		value.Consistency = ConsistencyRequireCurrent
		envelope, err := adapter.GetSnapshotStatus(request.Context(), value)
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	case "scopes":
		if request.Method != http.MethodGet && request.Method != http.MethodPost {
			return methodError(http.MethodGet, http.MethodPost)
		}
		value, err := queryRequest()
		if err != nil {
			return err
		}
		envelope, err := adapter.ListScopes(request.Context(), value)
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	case "search":
		if len(parts) < 2 {
			return newLiveError("mcp_route_not_found", "search route is missing a resource", nil)
		}
		if parts[1] == "text" {
			if request.Method != http.MethodPost {
				return methodError(http.MethodPost)
			}
			value, err := textRequest()
			if err != nil {
				return err
			}
			envelope, err := adapter.FindText(request.Context(), value)
			if err != nil {
				return err
			}
			return writeEnvelope(envelope)
		}
		if request.Method != http.MethodPost {
			return methodError(http.MethodPost)
		}
		value, err := queryRequest()
		if err != nil {
			return err
		}
		var envelope QueryEnvelope
		switch parts[1] {
		case "files":
			envelope, err = adapter.FindFiles(request.Context(), value)
		case "symbols":
			envelope, err = adapter.FindSymbols(request.Context(), value)
		case "documentation":
			envelope, err = adapter.GetDocumentation(request.Context(), value)
		default:
			return newLiveError("mcp_route_not_found", "search resource is unsupported", map[string]any{"resource": parts[1]})
		}
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	case "text":
		if request.Method != http.MethodPost {
			return methodError(http.MethodPost)
		}
		value, err := textRequest()
		if err != nil {
			return err
		}
		envelope, err := adapter.FindText(request.Context(), value)
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	case "modules", "module":
		if request.Method != http.MethodGet {
			return methodError(http.MethodGet)
		}
		if len(parts) < 2 {
			return newLiveError("mcp_route_not_found", "module route is missing a module id", nil)
		}
		value, err := queryRequest()
		if err != nil {
			return err
		}
		envelope, err := adapter.GetModuleFacts(request.Context(), value, parts[1])
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	case "callers-callees":
		if request.Method != http.MethodGet {
			return methodError(http.MethodGet)
		}
		if len(parts) < 2 {
			return newLiveError("mcp_route_not_found", "callers-callees route is missing an entity id", nil)
		}
		value, err := queryRequest()
		if err != nil {
			return err
		}
		envelope, err := adapter.GetCallersCallees(request.Context(), value, parts[1])
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	case "source-context":
		if request.Method != http.MethodPost {
			return methodError(http.MethodPost)
		}
		var value SourceContextRequest
		if err := decodeLiveJSON(request, &value); err != nil {
			return err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		envelope, err := adapter.GetSourceContext(request.Context(), value)
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	case "quality":
		return server.dispatchQuality(writer, request, parts[1:], queryRequest, writeEnvelope)
	case "policy":
		if request.Method != http.MethodPost {
			return methodError(http.MethodPost)
		}
		var value QualityPolicyCommand
		if err := decodeLiveJSON(request, &value); err != nil {
			return err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		envelope, err := adapter.Quality.ExecuteQualityPolicyCommand(request.Context(), value)
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	default:
		return newLiveError("mcp_route_not_found", "live HTTP route is unsupported", map[string]any{"route": parts[0]})
	}
}

func (server *HTTPServer) dispatchQuality(writer http.ResponseWriter, request *http.Request, parts []string, queryRequest func() (QueryRequest, error), writeEnvelope func(QueryEnvelope) error) error {
	if len(parts) == 0 {
		if request.Method != http.MethodGet && request.Method != http.MethodPost {
			return methodError(http.MethodGet, http.MethodPost)
		}
		value, err := queryRequest()
		if err != nil {
			return err
		}
		envelope, err := server.adapter.GetQualityCatalog(request.Context(), QualityCatalogRequest{QueryRequest: value, ProfileID: request.URL.Query().Get("profile_id"), ProfileVersion: request.URL.Query().Get("profile_version")})
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	}
	switch parts[0] {
	case "evaluate":
		if request.Method != http.MethodPost {
			return methodError(http.MethodPost)
		}
		var value QualityEvaluationRequest
		if err := decodeLiveJSON(request, &value); err != nil {
			return err
		}
		if value.SessionID == "" {
			value.SessionID = server.adapter.Session.validated.Config.SessionID
		}
		envelope, err := server.adapter.EvaluateQuality(request.Context(), value)
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	case "findings":
		if request.Method != http.MethodPost {
			return methodError(http.MethodPost)
		}
		var value QualityFindingsRequest
		if err := decodeLiveJSON(request, &value); err != nil {
			return err
		}
		if value.SessionID == "" {
			value.SessionID = server.adapter.Session.validated.Config.SessionID
		}
		envelope, err := server.adapter.GetQualityFindings(request.Context(), value)
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	case "evidence":
		if request.Method != http.MethodPost {
			return methodError(http.MethodPost)
		}
		var value QualityEvidenceRequest
		if err := decodeLiveJSON(request, &value); err != nil {
			return err
		}
		if value.SessionID == "" {
			value.SessionID = server.adapter.Session.validated.Config.SessionID
		}
		envelope, err := server.adapter.GetFindingEvidence(request.Context(), value)
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	case "compare":
		if request.Method != http.MethodPost {
			return methodError(http.MethodPost)
		}
		var value QualityCompareRequest
		if err := decodeLiveJSON(request, &value); err != nil {
			return err
		}
		if value.SessionID == "" {
			value.SessionID = server.adapter.Session.validated.Config.SessionID
		}
		envelope, err := server.adapter.CompareQualityReports(request.Context(), value)
		if err != nil {
			return err
		}
		return writeEnvelope(envelope)
	default:
		return newLiveError("mcp_route_not_found", "quality route is unsupported", map[string]any{"route": parts[0]})
	}
}

func splitLivePath(value string) ([]string, bool) {
	trimmed := strings.Trim(value, "/")
	rawParts := strings.Split(trimmed, "/")
	if len(rawParts) < 4 || rawParts[0] != "v1" || rawParts[1] != "live" || strings.TrimSpace(rawParts[2]) == "" {
		return nil, false
	}
	parts := make([]string, len(rawParts))
	for index, rawPart := range rawParts {
		part, err := url.PathUnescape(rawPart)
		if err != nil {
			return nil, false
		}
		parts[index] = part
	}
	return append([]string{parts[2]}, parts[3:]...), true
}

func decodeLiveJSON(request *http.Request, target any) error {
	if request.Body == nil {
		return newLiveError("mcp_invalid_request", "request body is required", nil)
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, http.ErrBodyReadAfterClose) || errors.Is(err, io.EOF) {
			return newLiveError("mcp_invalid_request", "request body must contain JSON", nil)
		}
		return newLiveError("mcp_invalid_request", "request body is invalid JSON", map[string]any{"error": err.Error()})
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return newLiveError("mcp_invalid_request", "request body contains trailing JSON", nil)
	}
	return nil
}

func writeLiveEnvelope(writer http.ResponseWriter, status int, envelope QueryEnvelope, maxBytes int) error {
	data, err := json.Marshal(envelope)
	if err != nil {
		return newLiveError("mcp_serialization_failed", "live response could not be encoded", nil)
	}
	if maxBytes > 0 && len(data) > maxBytes {
		return newLiveError(ErrorQueryBudget, "live HTTP response exceeds the transport byte budget", map[string]any{"max_bytes": maxBytes, "emitted_bytes": len(data)})
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, _ = writer.Write(data)
	return nil
}

func writeLiveHTTPError(writer http.ResponseWriter, status int, err error) {
	code, message, details := liveErrorDetails(err)
	if status < 400 {
		status = http.StatusInternalServerError
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]any{"error": map[string]any{"code": code, "message": message, "details": details}})
}

func liveErrorDetails(err error) (string, string, map[string]any) {
	var queryErr *QueryError
	if errors.As(err, &queryErr) {
		return queryErr.Code, queryErr.Message, queryErr.Details
	}
	if err == nil {
		return "", "", nil
	}
	return "mcp_internal_error", err.Error(), nil
}

func httpStatusForLiveError(err error) int {
	var queryErr *QueryError
	if !errors.As(err, &queryErr) {
		return http.StatusInternalServerError
	}
	switch queryErr.Code {
	case ErrorQueryBudget, ErrorQueryCursor:
		return http.StatusRequestEntityTooLarge
	case ErrorNoReadySnapshot, ErrorRevisionUnavailable, ErrorAnalysisWaitTimeout, ErrorInputUnstable:
		return http.StatusConflict
	case "mcp_route_not_found", "mcp_session_not_found":
		return http.StatusNotFound
	case "mcp_permission_denied", ErrorQualityPolicyPermissionDenied:
		return http.StatusForbidden
	case "mcp_invalid_request", "QueryInvalid":
		return http.StatusBadRequest
	default:
		return http.StatusUnprocessableEntity
	}
}

func methodError(methods ...string) error {
	return newLiveError("mcp_method_not_allowed", fmt.Sprintf("HTTP method is not allowed; use %s", strings.Join(methods, " or ")), nil)
}
