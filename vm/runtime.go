package vm

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/peiblow/vvm/compiler"
)

type Runtime struct {
	contracts map[string]*compiler.ContractArtifact
	baseDir   string
	mu        sync.RWMutex
}

func NewRuntime() *Runtime {
	return &Runtime{
		contracts: make(map[string]*compiler.ContractArtifact),
		baseDir:   ".",
	}
}

// SetBaseDir sets the directory that module imports are resolved against.
// Defaults to "." — local mode points it at the contract's own directory.
func (r *Runtime) SetBaseDir(dir string) {
	r.baseDir = dir
}

type WireMessage struct {
	Type string
	ID   string
	Data json.RawMessage
}

type WireResponse struct {
	Type    string
	ID      string
	Success bool
	Data    interface{}
	Error   interface{}
}

type DeployRequest struct {
	Hash         string `json:"hash"`
	ContractName string `json:"contract_name"`
	Version      string `json:"version"`
	Owner        string `json:"owner"`
	Source       []byte `json:"source"`
}

type ExecRequest struct {
	ArtifactHash     string                 `json:"contract_id"`
	ContractArtifact json.RawMessage        `json:"contract_artifact"`
	Function         string                 `json:"function"`
	Args             map[string]interface{} `json:"args"`
	CoreArgs         map[string]interface{} `json:"core_args"`
}

type AgentInfo struct {
	Hash    string `json:"hash"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (r *Runtime) HandleConnection(conn net.Conn) {
	defer conn.Close()

	var frameLength uint32
	if err := binary.Read(conn, binary.BigEndian, &frameLength); err != nil {
		fmt.Println("Error reading frame length:", err)
		return
	}

	payload := make([]byte, frameLength)
	if _, err := io.ReadFull(conn, payload); err != nil {
		fmt.Println("Error reading payload:", err)
		return
	}

	var msg WireMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		fmt.Println("Error unmarshaling message:", err)
		return
	}

	response := r.processMessage(&msg)

	// Serializing response
	respBytes, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Error marshaling response:", err)
		return
	}

	respLength := uint32(len(respBytes))
	if err := binary.Write(conn, binary.BigEndian, respLength); err != nil {
		fmt.Println("Error writing response length:", err)
		return
	}

	if _, err := conn.Write(respBytes); err != nil {
		fmt.Println("Error writing response payload:", err)
		return
	}
}

func (r *Runtime) processMessage(msg *WireMessage) (resp WireResponse) {
	defer func() {
		if rec := recover(); rec != nil {
			resp = WireResponse{
				Type:    "ERROR",
				ID:      msg.ID,
				Success: false,
				Error:   fmt.Sprintf("internal error: %v", rec),
			}
		}
	}()

	switch msg.Type {
	case "DEPLOY":
		return r.HandleDeploy(msg)
	case "EXEC":
		return r.HandleExec(msg)
	case "PING":
		return WireResponse{
			Type:    "PONG",
			ID:      msg.ID,
			Success: true,
		}
	default:
		return WireResponse{
			Type:    "ERROR",
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("unknown message type: %s", msg.Type),
		}
	}
}

func (r *Runtime) HandleDeploy(msg *WireMessage) WireResponse {
	fmt.Printf("Received DEPLOY request with ID %s\n", msg.ID)
	var req DeployRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return WireResponse{
			Type:    "DEPLOY_RESPONSE",
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("invalid deploy request: %v", err),
		}
	}

	artifact, agentInfo, err := Build(string(req.Source), r.baseDir)
	if err != nil {
		return WireResponse{
			Type:    "DEPLOY_RESPONSE",
			ID:      msg.ID,
			Success: false,
			Error:   err.Error(),
		}
	}

	fmt.Println("Successful parse and analysis of contract source. Proceeding to deployment...")

	r.mu.Lock()
	r.contracts[req.Hash] = artifact
	r.mu.Unlock()

	fmt.Println("Successful Deploy! ", req.Hash)

	return WireResponse{
		Type:    "DEPLOY_RESPONSE",
		ID:      msg.ID,
		Success: true,
		Data: map[string]interface{}{
			"contract_hash":     req.Hash,
			"contract_version":  req.Version,
			"contract_name":     req.ContractName,
			"contract_owner":    req.Owner,
			"contract_artifact": artifact,
			"functions":         getFunctionNames(artifact),
			"agent":             agentInfo,
		},
	}
}

func (r *Runtime) HandleExec(msg *WireMessage) WireResponse {
	fmt.Printf("Received EXEC request with ID %s\n", msg.ID)

	var req ExecRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return WireResponse{
			Type:    "EXEC_RESPONSE",
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("invalid exec request: %v", err),
		}
	}

	var artifact compiler.ContractArtifact
	if err := json.Unmarshal(req.ContractArtifact, &artifact); err != nil {
		return WireResponse{
			Type:    "EXEC_RESPONSE",
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("invalid exec request: %v", err),
		}
	}

	if len(artifact.Bytecode) == 0 {
		return WireResponse{
			Type:    "EXEC_RESPONSE",
			ID:      msg.ID,
			Success: false,
			Error:   "empty bytecode",
		}
	}

	funcMeta, funcExists := artifact.Functions[req.Function]
	if !funcExists {
		return WireResponse{
			Type:    "EXEC_RESPONSE",
			ID:      msg.ID,
			Success: false,
			Error:   fmt.Sprintf("function '%s' not found in contract", req.Function),
		}
	}

	var orderedArgs []interface{}
	for _, meta := range funcMeta.ArgMeta {
		val, exists := req.Args[meta.Name]
		if !exists {
			fmt.Println("Provided args:", req.Args)
			fmt.Printf("Missing argument '%s' for function '%s'\n", meta.Name, req.Function)
			return WireResponse{
				Type:    "EXEC_RESPONSE",
				ID:      msg.ID,
				Success: false,
				Error:   fmt.Sprintf("missing argument '%s' for function '%s'", meta.Name, req.Function),
			}
		}

		orderedArgs = append(orderedArgs, val)
	}

	vm := NewFromArtifact(&artifact)

	// CoreArgs become the global scope object, reachable in contracts via
	// `this` (e.g. this.riskScore) — they never mix with positional args.
	vm.SetGlobalScope(req.CoreArgs)

	result := vm.RunFunction(req.Function, orderedArgs...)

	if !result.Success {
		return WireResponse{
			Type:    "EXEC_RESPONSE",
			ID:      msg.ID,
			Success: false,
			Error:   result.Error,
		}
	}

	return WireResponse{
		Type:    "EXEC_RESPONSE",
		ID:      msg.ID,
		Success: true,
		Data: map[string]interface{}{
			"artifact_hash": req.ArtifactHash,
			"function":      req.Function,
			"journal":       result.Journal,
		},
		Error: result.Error,
	}
}

func (r *Runtime) GetContract(contractID string) (*compiler.ContractArtifact, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	artifact, exists := r.contracts[contractID]
	return artifact, exists
}

func getFunctionNames(artifact *compiler.ContractArtifact) []string {
	names := make([]string, 0, len(artifact.Functions))
	for name := range artifact.Functions {
		names = append(names, name)
	}
	return names
}

func getAgents(artifact *compiler.ContractArtifact) (*AgentInfo, error) {
	for _, val := range artifact.InitStorage {
		agentMap, ok := val.(map[string]interface{})
		if !ok {
			continue
		}

		// Agent entries always have a "hash" field generated by execAgentDeclare.
		rawHash, hasHash := agentMap["hash"]
		if !hasHash {
			continue
		}

		hash, ok := rawHash.(string)
		if !ok {
			continue
		}

		name, _ := agentMap["name"].(string)
		version, _ := agentMap["version"].(string)

		return &AgentInfo{
			Hash:    hash,
			Name:    name,
			Version: version,
		}, nil
	}

	return nil, fmt.Errorf("no agent declaration found in contract storage — did you declare an 'agent' block?")
}
