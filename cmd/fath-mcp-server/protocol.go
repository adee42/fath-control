// Minimal JSON-RPC 2.0 / MCP stdio transport. Only the subset of the Model
// Context Protocol actually needed to serve tools is implemented:
// initialize, notifications/initialized, ping, tools/list and tools/call.
//
// Requests/responses are newline-delimited JSON objects on stdin/stdout, per
// the MCP stdio transport. Nothing but JSON-RPC may ever be written to
// stdout - all logging goes to stderr.
package main

import "encoding/json"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// isNotification reports whether the request has no ID, meaning no response
// should be sent (per JSON-RPC 2.0).
func (r rpcRequest) isNotification() bool {
	return len(r.ID) == 0
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// JSON-RPC standard error codes used here.
const (
	errParseError     = -32700
	errInvalidRequest = -32600
	errMethodNotFound = -32601
	errInvalidParams  = -32602
	errInternal       = -32603
)

// initializeParams is what the client sends with "initialize". Only the
// fields we actually use are modeled.
type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type initializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      serverInfo     `json:"serverInfo"`
}

// toolDef describes one tool as returned by "tools/list".
type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolsListResult struct {
	Tools []toolDef `json:"tools"`
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolsCallResult struct {
	Content []contentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

func textResult(text string) toolsCallResult {
	return toolsCallResult{Content: []contentBlock{{Type: "text", Text: text}}}
}

func errorResult(text string) toolsCallResult {
	return toolsCallResult{Content: []contentBlock{{Type: "text", Text: text}}, IsError: true}
}
