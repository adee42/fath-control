// Command fath-mcp-server is a Model Context Protocol server that lets an
// LLM read free@home sensor values and control free@home actuators. It's
// designed to run as the backend of a Claude Desktop extension: connection
// details are supplied via environment variables (set by the extension's
// user_config), and it speaks MCP over stdio.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/adee42/fath-control/internal/fath"
)

const (
	serverName    = "fath-control"
	serverVersion = "1.1.0"
)

func main() {
	host := os.Getenv("FATH_HOST")
	user := os.Getenv("FATH_USER")
	password := os.Getenv("FATH_PASSWORD")
	if host == "" || user == "" || password == "" {
		fmt.Fprintln(os.Stderr, "fath-mcp-server: FATH_HOST, FATH_USER and FATH_PASSWORD must be set (configure them in the extension's settings)")
		os.Exit(1)
	}

	scheme := os.Getenv("FATH_SCHEME")
	if scheme == "" {
		scheme = "https"
	}

	insecure := true
	if v := os.Getenv("FATH_INSECURE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			insecure = b
		}
	}

	client := fath.NewClient(fath.Options{
		Host:               host,
		Scheme:             scheme,
		User:               user,
		Password:           password,
		InsecureSkipVerify: insecure,
		Timeout:            10 * time.Second,
	})

	(&server{client: client}).run()
}

type server struct {
	client *fath.Client
}

// run implements the MCP stdio transport: one JSON-RPC message per line on
// stdin, one JSON-RPC message per line on stdout. Nothing else may ever be
// written to stdout.
func (s *server) run() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			fmt.Fprintf(os.Stderr, "fath-mcp-server: invalid JSON-RPC message: %v\n", err)
			continue
		}
		if resp := s.handle(req); resp != nil {
			writeResponse(out, *resp)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "fath-mcp-server: stdin read error: %v\n", err)
		os.Exit(1)
	}
}

func writeResponse(out *bufio.Writer, resp rpcResponse) {
	b, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fath-mcp-server: failed to encode response: %v\n", err)
		return
	}
	out.Write(b)
	out.WriteByte('\n')
	out.Flush()
}

func (s *server) handle(req rpcRequest) *rpcResponse {
	notify := req.isNotification()

	switch req.Method {
	case "initialize":
		var p initializeParams
		_ = json.Unmarshal(req.Params, &p) // best effort; zero value is fine
		version := p.ProtocolVersion
		if version == "" {
			version = "2025-06-18"
		}
		return okResponse(req, notify, initializeResult{
			ProtocolVersion: version,
			Capabilities:    map[string]any{"tools": map[string]any{}},
			ServerInfo:      serverInfo{Name: serverName, Version: serverVersion},
		})

	case "notifications/initialized", "notifications/cancelled":
		return nil

	case "ping":
		return okResponse(req, notify, map[string]any{})

	case "tools/list":
		return okResponse(req, notify, toolsListResult{Tools: toolDefs})

	case "tools/call":
		if notify {
			return nil
		}
		var p toolsCallParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return errResponse(req, errInvalidParams, "invalid tool call params: "+err.Error())
		}
		return okResponse(req, notify, s.callTool(context.Background(), p))

	default:
		if notify {
			return nil
		}
		return errResponse(req, errMethodNotFound, "method not found: "+req.Method)
	}
}

func okResponse(req rpcRequest, notify bool, result any) *rpcResponse {
	if notify {
		return nil
	}
	return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
}

func errResponse(req rpcRequest, code int, message string) *rpcResponse {
	return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: code, Message: message}}
}
