package live

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	MCPProtocolVersion     = "2025-06-18"
	MCPServerName          = "arch-view"
	MCPServerVersion       = "0.0.1"
	DefaultMCPMessageBytes = 4 << 20
)

type MCPServer struct {
	Adapter         *LocalQueryAdapter
	MaxMessageBytes int
}

func NewMCPServer(adapter *LocalQueryAdapter) *MCPServer {
	return &MCPServer{Adapter: adapter, MaxMessageBytes: DefaultMCPMessageBytes}
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpRPCError    `json:"error,omitempty"`
}

type mcpRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Serve runs the newline-delimited JSON-RPC stdio transport. It writes only
// protocol messages to out; diagnostics belong to the caller's stderr.
func (server *MCPServer) Serve(ctx context.Context, input io.Reader, output io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if server == nil || server.Adapter == nil || server.Adapter.Session == nil {
		return newLiveError(ErrorLiveConfigInvalid, "MCP adapter is unavailable", nil)
	}
	limit := server.MaxMessageBytes
	if limit < 1 {
		limit = DefaultMCPMessageBytes
	}
	scanner := bufio.NewScanner(input)
	bufferSize := 4096
	if limit < bufferSize {
		bufferSize = limit
	}
	scanner.Buffer(make([]byte, bufferSize), limit)
	stopClose := make(chan struct{})
	if closer, ok := input.(io.Closer); ok {
		go func() {
			select {
			case <-ctx.Done():
				_ = closer.Close()
			case <-stopClose:
			}
		}()
	}
	defer close(stopClose)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var request mcpRequest
		if err := json.Unmarshal(line, &request); err != nil {
			if writeErr := writeMCPResponse(output, mcpResponse{JSONRPC: "2.0", Error: &mcpRPCError{Code: -32700, Message: "parse error", Data: map[string]any{"error": err.Error()}}}); writeErr != nil {
				return writeErr
			}
			continue
		}
		if request.JSONRPC != "2.0" || strings.TrimSpace(request.Method) == "" {
			if len(request.ID) > 0 {
				if err := writeMCPResponse(output, mcpResponse{JSONRPC: "2.0", ID: request.ID, Error: &mcpRPCError{Code: -32600, Message: "invalid request"}}); err != nil {
					return err
				}
			}
			continue
		}
		if isMCPNotification(request) {
			continue
		}
		result, rpcErr := server.handleRequest(ctx, request)
		response := mcpResponse{JSONRPC: "2.0", ID: request.ID, Result: result, Error: rpcErr}
		if err := writeMCPResponse(output, response); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return writeMCPResponse(output, mcpResponse{JSONRPC: "2.0", Error: &mcpRPCError{
				Code: -32600, Message: "request exceeds the maximum message size",
				Data: map[string]any{"code": "mcp_message_too_large", "max_bytes": limit},
			}})
		}
		return err
	}
	return nil
}

func (server *MCPServer) handleRequest(ctx context.Context, request mcpRequest) (any, *mcpRPCError) {
	switch request.Method {
	case "initialize":
		return map[string]any{"protocolVersion": MCPProtocolVersion, "capabilities": map[string]any{"tools": map[string]any{}, "resources": map[string]any{}}, "serverInfo": map[string]any{"name": MCPServerName, "version": MCPServerVersion}, "instructions": "Use status or ensure_current_snapshot before reading facts. Responses carry one immutable revision and explicit coverage."}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": mcpToolCatalog()}, nil
	case "resources/list":
		return map[string]any{"resources": mcpResourceCatalog(server.Adapter.Session.validated.Config.SessionID)}, nil
	case "resources/read":
		return server.readResource(ctx, request.Params)
	case "tools/call":
		return server.callTool(ctx, request.Params)
	default:
		return nil, &mcpRPCError{Code: -32601, Message: "method not found", Data: map[string]any{"method": request.Method}}
	}
}

func (server *MCPServer) callTool(ctx context.Context, raw json.RawMessage) (any, *mcpRPCError) {
	var input struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}
	if err := json.Unmarshal(raw, &input); err != nil || strings.TrimSpace(input.Name) == "" {
		return nil, &mcpRPCError{Code: -32602, Message: "tools/call requires a tool name", Data: map[string]any{"error": errorString(err)}}
	}
	envelope, err := server.executeTool(ctx, input.Name, input.Arguments)
	if err != nil {
		code, message, details := liveErrorDetails(err)
		value := map[string]any{"code": code, "message": message, "details": details}
		encoded, _ := json.Marshal(value)
		return map[string]any{"content": []map[string]any{{"type": "text", "text": string(encoded)}}, "isError": true, "structuredContent": value}, nil
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return nil, &mcpRPCError{Code: -32603, Message: "tool result could not be encoded"}
	}
	return map[string]any{"content": []map[string]any{{"type": "text", "text": string(encoded)}}, "structuredContent": envelope}, nil
}

func (server *MCPServer) executeTool(ctx context.Context, name string, raw json.RawMessage) (QueryEnvelope, error) {
	adapter := server.Adapter
	sessionID := adapter.Session.validated.Config.SessionID
	decodeQuery := func() (QueryRequest, error) {
		var query QueryRequest
		if err := decodeTool(raw, &query); err != nil {
			return QueryRequest{}, err
		}
		if query.SessionID == "" {
			query.SessionID = sessionID
		}
		return query, nil
	}
	switch name {
	case "get_snapshot_status":
		query, err := decodeQuery()
		if err != nil {
			return QueryEnvelope{}, err
		}
		if query.Consistency == "" {
			query.Consistency = ConsistencyLatestReady
		}
		return adapter.GetSnapshotStatus(ctx, query)
	case "ensure_current_snapshot":
		query, err := decodeQuery()
		if err != nil {
			return QueryEnvelope{}, err
		}
		query.Consistency = ConsistencyRequireCurrent
		return adapter.GetSnapshotStatus(ctx, query)
	case "list_scopes":
		query, err := decodeQuery()
		if err != nil {
			return QueryEnvelope{}, err
		}
		return adapter.ListScopes(ctx, query)
	case "find_files":
		query, err := decodeQuery()
		if err != nil {
			return QueryEnvelope{}, err
		}
		return adapter.FindFiles(ctx, query)
	case "find_symbols":
		query, err := decodeQuery()
		if err != nil {
			return QueryEnvelope{}, err
		}
		return adapter.FindSymbols(ctx, query)
	case "get_documentation":
		query, err := decodeQuery()
		if err != nil {
			return QueryEnvelope{}, err
		}
		return adapter.GetDocumentation(ctx, query)
	case "find_text":
		var value TextSearchQuery
		if err := decodeTool(raw, &value); err != nil {
			return QueryEnvelope{}, err
		}
		if value.Consistency == "" {
			value.Consistency = ConsistencyLatestReady
		}
		return adapter.FindText(ctx, value)
	case "get_module_facts":
		var value struct {
			QueryRequest
			ModuleID string `json:"module_id"`
		}
		if err := decodeTool(raw, &value); err != nil {
			return QueryEnvelope{}, err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		return adapter.GetModuleFacts(ctx, value.QueryRequest, value.ModuleID)
	case "get_callers_callees":
		var value struct {
			QueryRequest
			EntityID string `json:"entity_id"`
		}
		if err := decodeTool(raw, &value); err != nil {
			return QueryEnvelope{}, err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		return adapter.GetCallersCallees(ctx, value.QueryRequest, value.EntityID)
	case "get_source_context":
		var value SourceContextRequest
		if err := decodeTool(raw, &value); err != nil {
			return QueryEnvelope{}, err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		return adapter.GetSourceContext(ctx, value)
	case "get_quality_profiles", "get_quality_rules":
		var value QualityCatalogRequest
		if err := decodeTool(raw, &value); err != nil {
			return QueryEnvelope{}, err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		return adapter.GetQualityCatalog(ctx, value)
	case "get_quality_findings":
		var value QualityFindingsRequest
		if err := decodeTool(raw, &value); err != nil {
			return QueryEnvelope{}, err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		return adapter.GetQualityFindings(ctx, value)
	case "get_finding_evidence":
		var value QualityEvidenceRequest
		if err := decodeTool(raw, &value); err != nil {
			return QueryEnvelope{}, err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		return adapter.GetFindingEvidence(ctx, value)
	case "evaluate_quality":
		var value QualityEvaluationRequest
		if err := decodeTool(raw, &value); err != nil {
			return QueryEnvelope{}, err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		return adapter.EvaluateQuality(ctx, value)
	case "compare_quality_reports":
		var value QualityCompareRequest
		if err := decodeTool(raw, &value); err != nil {
			return QueryEnvelope{}, err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		return adapter.CompareQualityReports(ctx, value)
	case "validate_quality_profile", "preview_baseline", "save_quality_profile", "save_quality_profile_as", "create_baseline":
		var value QualityPolicyCommand
		if err := decodeTool(raw, &value); err != nil {
			return QueryEnvelope{}, err
		}
		if value.SessionID == "" {
			value.SessionID = sessionID
		}
		switch name {
		case "validate_quality_profile":
			value.Operation = PolicyValidateProfile
		case "preview_baseline":
			value.Operation = PolicyPreviewBaseline
		case "save_quality_profile":
			value.Operation = PolicySaveProfile
		case "save_quality_profile_as":
			value.Operation = PolicySaveProfileAs
		case "create_baseline":
			value.Operation = PolicyCreateBaseline
		}
		return adapter.Quality.ExecuteQualityPolicyCommand(ctx, value)
	default:
		return QueryEnvelope{}, newLiveError("mcp_operation_unsupported", "MCP tool is unsupported", map[string]any{"tool": name})
	}
}

func (server *MCPServer) readResource(ctx context.Context, raw json.RawMessage) (any, *mcpRPCError) {
	var input struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(raw, &input); err != nil || strings.TrimSpace(input.URI) == "" {
		return nil, &mcpRPCError{Code: -32602, Message: "resources/read requires a URI"}
	}
	prefix := "archview://session/" + server.Adapter.Session.validated.Config.SessionID + "/"
	if !strings.HasPrefix(input.URI, prefix) {
		return nil, &mcpRPCError{Code: -32602, Message: "resource URI is outside the live session"}
	}
	name := strings.TrimPrefix(input.URI, prefix)
	var envelope QueryEnvelope
	var err error
	query := QueryRequest{SessionID: server.Adapter.Session.validated.Config.SessionID, Consistency: ConsistencyLatestReady}
	switch name {
	case "status":
		envelope, err = server.Adapter.GetSnapshotStatus(ctx, query)
	case "scopes":
		envelope, err = server.Adapter.ListScopes(ctx, query)
	case "quality":
		envelope, err = server.Adapter.GetQualityCatalog(ctx, QualityCatalogRequest{QueryRequest: query})
	case "snapshot":
		return nil, &mcpRPCError{Code: -32602, Message: "snapshot resources require a revision suffix"}
	default:
		if strings.HasPrefix(name, "snapshot/") {
			value, parseErr := parsePositiveInt(strings.TrimPrefix(name, "snapshot/"))
			if parseErr != nil {
				return nil, &mcpRPCError{Code: -32602, Message: "snapshot resource revision is invalid"}
			}
			query.Consistency = ConsistencySpecific
			query.Revision = value
			envelope, err = server.Adapter.GetSnapshotStatus(ctx, query)
		} else {
			return nil, &mcpRPCError{Code: -32602, Message: "resource URI is unsupported"}
		}
	}
	if err != nil {
		code, message, details := liveErrorDetails(err)
		return nil, &mcpRPCError{Code: -32000, Message: message, Data: map[string]any{"code": code, "details": details}}
	}
	data, _ := json.Marshal(envelope)
	return map[string]any{"contents": []map[string]any{{"uri": input.URI, "mimeType": "application/json", "text": string(data)}}}, nil
}

func decodeTool(raw json.RawMessage, target any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return newLiveError("mcp_invalid_request", "MCP tool arguments are invalid", map[string]any{"error": err.Error()})
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return newLiveError("mcp_invalid_request", "MCP tool arguments contain trailing JSON", nil)
	}
	return nil
}

func writeMCPResponse(output io.Writer, response mcpResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, string(data))
	return err
}

func isMCPNotification(request mcpRequest) bool {
	return len(request.ID) == 0 || string(request.ID) == "null"
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func parsePositiveInt(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 1 || strings.TrimSpace(value) == "" {
		return 0, errors.New("invalid positive integer")
	}
	return parsed, nil
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func mcpToolCatalog() []mcpTool {
	names := []string{"get_snapshot_status", "ensure_current_snapshot", "list_scopes", "find_files", "find_symbols", "find_text", "get_documentation", "get_module_facts", "get_callers_callees", "get_source_context", "get_quality_profiles", "get_quality_rules", "get_quality_findings", "get_finding_evidence", "evaluate_quality", "compare_quality_reports", "validate_quality_profile", "preview_baseline", "save_quality_profile", "save_quality_profile_as", "create_baseline"}
	result := make([]mcpTool, 0, len(names))
	for _, name := range names {
		result = append(result, mcpTool{Name: name, Description: mcpToolDescription(name), InputSchema: mcpToolInputSchema(name)})
	}
	return result
}

func mcpToolInputSchema(name string) map[string]any {
	stringValue := map[string]any{"type": "string"}
	integerValue := map[string]any{"type": "integer", "minimum": 1}
	booleanValue := map[string]any{"type": "boolean"}
	stringList := map[string]any{"type": "array", "items": stringValue}
	objectValue := map[string]any{"type": "object"}
	queryValue := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
		"path_prefix": stringValue, "language": stringValue, "scope_ids": stringList, "module_ids": stringList,
		"symbol_categories": stringList, "name": stringValue, "documentation_text": stringValue, "rule_ids": stringList,
		"assessment_kinds": stringList, "severities": stringList, "statuses": stringList, "file_ids": stringList,
		"subject_id": stringValue, "case_sensitive": booleanValue,
	}}
	queryProperties := func() map[string]any {
		return map[string]any{"session_id": stringValue, "consistency": map[string]any{"type": "string", "enum": []string{string(ConsistencyLatestReady), string(ConsistencyRequireCurrent), string(ConsistencySpecific)}}, "revision": integerValue, "query": queryValue, "projection": stringList, "max_bytes": integerValue, "max_items": integerValue, "cursor": stringValue}
	}
	properties := queryProperties()
	required := []string{}
	switch name {
	case "find_text":
		properties = map[string]any{"pattern": stringValue, "mode": map[string]any{"type": "string", "enum": []string{"literal", "regex"}}, "consistency": stringValue, "revision": integerValue, "path_glob": stringValue, "language": stringValue, "scope_ids": stringList, "case_sensitive": booleanValue, "max_line_bytes": integerValue, "max_bytes": integerValue, "max_items": integerValue, "cursor": stringValue}
		required = []string{"pattern"}
	case "get_module_facts":
		properties["module_id"] = stringValue
		required = []string{"module_id"}
	case "get_callers_callees":
		properties["entity_id"] = stringValue
		required = []string{"entity_id"}
	case "get_source_context":
		properties = map[string]any{"session_id": stringValue, "consistency": stringValue, "revision": integerValue, "scope_id": stringValue, "entity_id": stringValue, "span": objectValue, "max_lines": integerValue, "max_bytes": integerValue}
	case "get_quality_profiles", "get_quality_rules":
		properties["profile_id"] = stringValue
		properties["profile_version"] = stringValue
	case "get_quality_findings":
		properties["report_id"] = stringValue
	case "get_finding_evidence":
		properties["report_id"] = stringValue
		properties["finding_id"] = stringValue
		properties["include_source_context"] = booleanValue
		properties["max_lines"] = integerValue
		properties["max_context_bytes"] = integerValue
		required = []string{"finding_id"}
	case "evaluate_quality":
		properties = map[string]any{"session_id": stringValue, "consistency": stringValue, "revision": integerValue, "scope_ids": stringList, "profile_id": stringValue, "profile_version": stringValue, "rule_bindings": map[string]any{"type": "array", "items": objectValue}, "persist": booleanValue, "max_bytes": integerValue, "max_items": integerValue}
		required = []string{"profile_id", "profile_version"}
	case "compare_quality_reports":
		properties = map[string]any{"session_id": stringValue, "consistency": stringValue, "previous_revision": integerValue, "current_revision": integerValue, "previous_report_id": stringValue, "current_report_id": stringValue, "max_bytes": integerValue, "max_items": integerValue}
		required = []string{"previous_revision"}
	case "validate_quality_profile", "preview_baseline", "save_quality_profile", "save_quality_profile_as", "create_baseline":
		properties = map[string]any{"operation": stringValue, "session_id": stringValue, "report_id": stringValue, "report_revision": integerValue, "profile_id": stringValue, "profile_version": stringValue, "source_profile_id": stringValue, "source_profile_version": stringValue, "profile": objectValue, "rule_bindings": map[string]any{"type": "array", "items": objectValue}, "file_name": stringValue, "baseline_id": stringValue, "baseline_revision": stringValue, "finding_keys": stringList, "reason": stringValue, "owner": stringValue, "authorization": stringValue, "overwrite": booleanValue}
		if name == "preview_baseline" || name == "create_baseline" {
			required = []string{"baseline_id", "finding_keys", "reason"}
		}
	}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": properties, "required": required}
}

func mcpToolDescription(name string) string {
	if strings.Contains(name, "quality") || strings.Contains(name, "baseline") || strings.Contains(name, "profile") {
		return "Read or explicitly manage the bounded quality policy surface; responses retain revision and coverage metadata."
	}
	if strings.Contains(name, "source") || strings.Contains(name, "text") {
		return "Read a bounded source fact or source context from the selected live revision."
	}
	return "Read a bounded, revision-aware fact from the live analysis session."
}

func mcpResourceCatalog(sessionID string) []map[string]any {
	prefix := "archview://session/" + sessionID + "/"
	return []map[string]any{{"uri": prefix + "status", "name": "Live status"}, {"uri": prefix + "scopes", "name": "Analysis scopes"}, {"uri": prefix + "quality", "name": "Quality catalog"}}
}
