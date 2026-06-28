//go:build local

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/peiblow/vvm/compiler"
	"github.com/peiblow/vvm/vm"
)

func main() {
	runtime := vm.NewRuntime()
	fmt.Println("Running in local mode with mock runtime")

	contractPath := "contracts/lts/wallet_agent.snx"
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		fmt.Println("Error reading contract file:", err)
		return
	}

	runtime.SetBaseDir(filepath.Dir(contractPath))

	deployReq := vm.DeployRequest{
		Hash:         "mockhash",
		ContractName: "SynxFintech",
		Version:      "1.0.0",
		Owner:        "0xAB1234CD56EF7890",
		Source:       []byte(contract),
	}

	msg := vm.WireMessage{
		Type: "DEPLOY",
		ID:   "mockdeploy1",
		Data: func() json.RawMessage {
			data, _ := json.Marshal(deployReq)
			return data
		}(),
	}
	deployRes := runtime.HandleDeploy(&msg)

	if !deployRes.Success {
		fmt.Printf("DEPLOY failed: %s\n", deployRes.Error)
		return
	}

	contractArtifactRes := deployRes.Data.(map[string]interface{})["contract_artifact"]

	artifactBytes, err := json.Marshal(contractArtifactRes.(*compiler.ContractArtifact))
	if err != nil {
		fmt.Println("Error marshaling artifact:", err)
		return
	}

	execReq := vm.ExecRequest{
		ArtifactHash:     "mockhash",
		ContractArtifact: json.RawMessage(artifactBytes),
		Function:         "verifyTransferAccounts",
		Args: map[string]interface{}{
			"from":       "0xAB1234CD56EF7890",
			"to":         "0xCD56EF7890AB1234",
			"amount":     1000,
			"currency":   "BRL",
			"dailyTotal": 5000,
		},
	}
	execMsg := vm.WireMessage{
		Type: "EXEC",
		ID:   "mockexec1",
		Data: func() json.RawMessage {
			data, _ := json.Marshal(execReq)
			return data
		}(),
	}
	execRes := runtime.HandleExec(&execMsg)
	fmt.Printf("EXEC response: %+v\n", execRes)
}
