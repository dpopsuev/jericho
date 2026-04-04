// protocol.go — ACP JSON-RPC message types.
//
// Defines the wire format for the Agent Client Protocol: initialize,
// session/new, session/prompt, session/update notifications.
package acp

import "encoding/json"

// Protocol version.
const ProtocolVersion = 1

// JSON-RPC request/response envelope.
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      int              `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Params  *json.RawMessage `json:"params,omitempty"`
	Error   *jsonRPCError    `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// --- Initialize ---

type initializeParams struct {
	ProtocolVersion int        `json:"protocolVersion"`
	ClientInfo      clientInfo `json:"clientInfo"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type initializeResult struct {
	ProtocolVersion   int             `json:"protocolVersion"`
	AgentInfo         agentInfo       `json:"agentInfo"`
	AgentCapabilities json.RawMessage `json:"agentCapabilities,omitempty"`
}

type agentInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// --- Session ---

type newSessionParams struct {
	CWD        string `json:"cwd"`
	MCPServers []any  `json:"mcpServers"` // required by ACP spec, empty array if none
}

type newSessionResult struct {
	SessionID string `json:"sessionId"`
}

// --- Prompt ---

type promptParams struct {
	SessionID string        `json:"sessionId"`
	Prompt    []promptBlock `json:"prompt"`
}

type promptBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// --- Session Update (notification) ---

type sessionUpdateNotification struct {
	SessionID string        `json:"sessionId"`
	Update    sessionUpdate `json:"update"`
}

type sessionUpdate struct {
	SessionUpdate string        `json:"sessionUpdate"`
	Content       *contentBlock `json:"content,omitempty"` // for agent_message_chunk
	ToolCallID    string        `json:"toolCallId,omitempty"`
	Title         string        `json:"title,omitempty"`
	Status        string        `json:"status,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}
