package vm

import (
	"fmt"

	"github.com/peiblow/vvm/ast"
	"github.com/peiblow/vvm/compiler"
	"github.com/peiblow/vvm/lexer"
	"github.com/peiblow/vvm/loader"
	"github.com/peiblow/vvm/parser"
)

func Build(source, baseDir string) (*compiler.ContractArtifact, error) {
	lexResult := lexer.Tokenize(source)
	if lexResult.HasErrors() {
		errMsg := "lexical errors in contract source:\n"
		for _, e := range lexResult.Errors {
			errMsg += "  " + e.Error() + "\n"
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	prog, err := loader.NewLoader().Resolver(lexResult.Tokens, baseDir)
	if err != nil {
		return nil, fmt.Errorf("module resolution error: %w", err)
	}

	mods := make([]ast.BlockStmt, len(prog.Modules))
	for i, m := range prog.Modules {
		mods[i] = m.AST
	}

	analysis := parser.AnalyzeProgram(prog.Main, mods)
	if analysis.HasErrors() {
		return nil, fmt.Errorf("semantic errors in contract source:\n %v", analysis.Errors)
	}

	cmpl := compiler.New(baseDir)
	cmpl.CompileProgram(mods, prog.Main)
	artifact := cmpl.Artifact()

	initVM := NewFromArtifact(artifact)
	if initResult := initVM.Run(); !initResult.Success {
		return nil, fmt.Errorf("initialization failed: %v", initResult.Error)
	}
	artifact.InitStorage = initVM.GetStorage()

	// O hash do agente é computado na inicialização e vive no InitStorage.
	// name/version/purpose já vêm preenchidos do compileAgentStmt — aqui só
	// completamos o AgentInfo com o hash.
	hash, err := agentHashFromStorage(artifact)
	if err != nil {
		return nil, fmt.Errorf("agent metadata error: %w", err)
	}
	artifact.AgentInfo.Hash = hash

	return artifact, nil
}
