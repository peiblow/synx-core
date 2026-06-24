package compiler

import (
	"fmt"

	"github.com/peiblow/vvm/ast"
)

func (c *Compiler) compileStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case ast.ImportStmt:
		c.emit(OP_NOP)
	case ast.LibraryStmt:
		c.compileLibrary(s)
	case ast.ContractStmt:
		c.compileContract(s)
	case ast.ExpressionStmt:
		c.compileExpr(s.Expression)
	case ast.VarDeclStmt:
		c.compileVarDecl(s)
	case ast.ReturnStmt:
		c.compileReturn(s)
	case ast.FuncStmt:
		c.compileFunc(s)
	case ast.IfStmt:
		c.compileIf(s)
	case ast.ForStmt:
		c.compileFor(s)
	case ast.WhileStmt:
		c.compileWhile(s)
	case ast.RequireStmt:
		c.compileRequire(s)
	case ast.AgentStmt:
		c.compileAgentStmt(s)
	case ast.PolicyStmt:
		c.compilePolicyStmt(s)
	case ast.TypeDeclareStmt:
		c.compileTypeDeclareStmt(s)
	case ast.EmitStmt:
		c.compileEmitStmt(s)
	case ast.TryCatchStmt:
		c.compileTryCatchStmt(s)
	case ast.ToolStmt:
		c.Tools = append(c.Tools, convertTool(s))
	default:
		fmt.Printf("Unrecognized statement type: %T\n", s)
	}
}

func (c *Compiler) compileLibrary(s ast.LibraryStmt) {
	for _, stmt := range s.Body {
		c.compileStmt(stmt)
	}
}

func (c *Compiler) compileContract(s ast.ContractStmt) {
	for _, stmt := range s.Body {
		c.compileStmt(stmt)
	}
}

func (c *Compiler) compileVarDecl(s ast.VarDeclStmt) {
	if s.AssignedValue == nil {
		return
	}

	if _, exists := c.Symbols[s.Identifier]; exists {
		erroMsg := fmt.Sprintf("Variable %s already declared", s.Identifier)
		errIdx := c.addConst(erroMsg)
		c.emit(OP_CONST, errIdx)
		c.emit(OP_ERR)
	} else if _, exists := c.Symbols[s.Identifier+"/CONST"]; exists {
		erroMsg := fmt.Sprintf("Variable %s already declared as a constant", s.Identifier)
		errIdx := c.addConst(erroMsg)
		c.emit(OP_CONST, errIdx)
		c.emit(OP_ERR)
	}

	if s.Constant {
		c.Symbols[s.Identifier+"/CONST"] = c.Symbols[s.Identifier]
	}

	c.compileExpr(s.AssignedValue)
	slot := c.allocSlot(s.Identifier)
	c.emit(OP_STORE, byte(slot))
}

func (c *Compiler) compileReturn(s ast.ReturnStmt) {
	if s.Value != nil {
		c.compileExpr(s.Value)
	}
	if c.isInFunction {
		c.emit(OP_RET)
	} else {
		c.emit(OP_PRINT)
	}
}

func (c *Compiler) compileFunc(s ast.FuncStmt) {
	funcName := c.extractFuncName(s.Name)

	prevInFunction := c.isInFunction
	c.isInFunction = true

	outerSymbols := c.Symbols
	c.Symbols = make(map[string]int, len(outerSymbols))
	for k, v := range outerSymbols {
		c.Symbols[k] = v
	}

	skipFuncPos := c.currentPos()
	c.emit(OP_JMP, 0, 0)

	funcMeta := FunctionMeta{
		Addr:    c.currentPos(),
		Args:    []int{},
		ArgMeta: []ArgMeta{},
	}

	slots, argMeta := c.compileFuncArgs(s.Arguments)
	funcMeta.Args = slots
	funcMeta.ArgMeta = argMeta

	c.Functions[funcName] = funcMeta
	c.FunctionName[c.currentPos()] = funcName

	c.compileFuncBody(s.Body)

	c.emitFuncReturn(s.ReturnType)

	c.patchJump(skipFuncPos+1, c.currentPos())
	c.isInFunction = prevInFunction
	c.Symbols = outerSymbols
}

func (c *Compiler) extractFuncName(name ast.Expr) string {
	switch n := name.(type) {
	case ast.SymbolExpr:
		return n.Value
	case ast.ExpressionStmt:
		if sym, ok := n.Expression.(ast.SymbolExpr); ok {
			return sym.Value
		}
	}
	panic("Unsupported function name expression type")
}

func (c *Compiler) compileFuncArgs(args []ast.ArgsStmt) ([]int, []ArgMeta) {
	var slots []int
	var argMeta []ArgMeta

	for _, arg := range args {
		argName := ""
		if symExpr, ok := arg.ArgName.(ast.SymbolExpr); ok {
			argName = symExpr.Value
		} else {
			panic("Function argument name must be a symbol")
		}
		slot := c.allocSlot(argName)
		slots = append(slots, slot)

		// Extract type name for type checking
		typeName := ""
		if symExpr, ok := arg.ArgType.(ast.SymbolExpr); ok {
			typeName = symExpr.Value
		}

		argMeta = append(argMeta, ArgMeta{
			Name:     argName,
			Slot:     slot,
			TypeName: typeName,
		})
	}

	return slots, argMeta
}

func (c *Compiler) compileFuncBody(body ast.Stmt) {
	if block, ok := body.(ast.BlockStmt); ok {
		c.compileBlock(block)
	} else {
		c.compileStmt(body)
	}
}

func (c *Compiler) emitFuncReturn(returnType ast.Type) {
	c.emit(OP_RET)
}

func (c *Compiler) compileIf(s ast.IfStmt) {
	c.compileExpr(s.Condition)

	jmpIfPos := c.currentPos()
	c.emit(OP_JMP_IF, 0, 0)

	c.compileIfBlock(s.Then)

	if s.Else != nil {
		jmpPos := c.currentPos()
		c.emit(OP_JMP, 0, 0)

		elsePos := c.currentPos()
		c.compileIfBlock(s.Else)

		c.patchJump(jmpIfPos+1, elsePos)
		c.patchJump(jmpPos+1, c.currentPos())
	} else {
		c.patchJump(jmpIfPos+1, c.currentPos())
	}
}

func (c *Compiler) compileIfBlock(block ast.Stmt) {
	if b, ok := block.(ast.BlockStmt); ok {
		for _, stmt := range b.Body {
			c.compileStmt(stmt)
		}
	} else {
		c.compileStmt(block)
	}
}

func (c *Compiler) compileFor(s ast.ForStmt) {
	c.compileStmt(s.Init)

	condPos := c.currentPos()
	c.compileExpr(s.Condition)
	c.emit(OP_JMP_IF, 0, 0)
	jmpExitPos := c.currentPos()

	for _, stmt := range s.Body {
		c.compileStmt(stmt)
	}

	c.compileStmt(s.Post)

	c.emit(OP_JMP, byte(condPos>>8), byte(condPos&0xFF))

	c.patchJump(jmpExitPos-2, c.currentPos())
}

func (c *Compiler) compileWhile(s ast.WhileStmt) {
	condPos := c.currentPos()
	c.compileExpr(s.Condition)
	c.emit(OP_JMP_IF, 0, 0)
	jmpExitPos := c.currentPos()

	for _, stmt := range s.Body {
		c.compileStmt(stmt)
	}

	c.emit(OP_JMP, byte(condPos>>8), byte(condPos&0xFF))

	c.patchJump(jmpExitPos-2, c.currentPos())
}

func (c *Compiler) compileRequire(s ast.RequireStmt) {
	c.compileExpr(s.Condition)

	jmpToErrorPos := c.currentPos()
	c.emit(OP_JMP_IF, 0, 0)

	jmpPastErrorPos := c.currentPos()
	c.emit(OP_JMP, 0, 0)

	errorBlockPos := c.currentPos()
	c.compileExpr(s.Message)
	c.emit(OP_ERR)

	endPos := c.currentPos()

	c.patchJump(jmpToErrorPos+1, errorBlockPos)
	c.patchJump(jmpPastErrorPos+1, endPos)
}

func (c *Compiler) compileRegistryAgentDeclare(s ast.AgentStmt) {
	nameIdx := c.addConst(s.Identifier)
	c.emit(OP_CONST, nameIdx)
	versionIdx := c.addConst(s.Version)
	c.emit(OP_CONST, versionIdx)
	ownerIdx := c.addConst(s.Owner)
	c.emit(OP_CONST, ownerIdx)
	purposeIdx := c.addConst(s.Purpose)
	c.emit(OP_CONST, purposeIdx)

	c.emit(
		OP_AGENT_DECLARE,
		byte(nameIdx),
		byte(versionIdx),
		byte(ownerIdx),
		byte(purposeIdx),
	)
}

func (c *Compiler) compileAgentStmt(s ast.AgentStmt) {
	c.compileRegistryAgentDeclare(s)

	agentName := s.Identifier.(ast.SymbolExpr).Value
	c.allocSlot(agentName)

	identifierIdx := c.findConst(s.Identifier)
	if identifierIdx == 255 {
		identifierIdx = c.addConst(s.Identifier)
	}

	c.emit(OP_CONST, identifierIdx)
	c.emit(OP_AGENT_GET, byte(identifierIdx))

	versionIdx := c.addConst(s.Version)
	ownerIdx := c.addConst(s.Owner)

	c.emit(OP_CONST, versionIdx)
	c.emit(OP_CONST, ownerIdx)

	c.emit(OP_AGENT_VALIDATE)

	c.emit(OP_STORE, byte(c.Symbols[agentName]))

	for _, t := range s.Tools {
		c.Tools = append(c.Tools, convertTool(t))
	}
}

func convertTool(t ast.ToolStmt) ToolStmt {
	tool := ToolStmt{
		Name:        exprToString(t.Name),
		Description: exprToString(t.Description),
		Steps:       make([]ToolStep, len(t.Steps)),
	}

	for i, s := range t.Steps {
		step := ToolStep{Function: s.Function}
		if s.Action.Method != "" || s.Action.Url != nil {
			step.Action = &ToolAction{
				Method: s.Action.Method,
				Url:    exprToString(s.Action.Url),
			}
		}
		tool.Steps[i] = step
	}

	return tool
}

func exprToString(e ast.Expr) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case ast.StringExpr:
		return v.Value
	case ast.SymbolExpr:
		return v.Value
	case ast.GetEnvExpr:
		return fmt.Sprintf("getEnv(%s)", exprToString(v.VariableName))
	default:
		return fmt.Sprintf("%v", e)
	}
}

func (c *Compiler) compilePolicyStmt(s ast.PolicyStmt) {
	c.allocSlot(s.Identifier.(ast.SymbolExpr).Value)

	identifierIdx := c.findConst(s.Identifier)
	if identifierIdx == 255 {
		identifierIdx = c.addConst(s.Identifier)
	}
	c.emit(OP_CONST, identifierIdx)

	c.emit(OP_PUSH_OBJECT)
	for key, value := range s.Rules {
		keyIdx := c.addConst(key)
		c.emit(OP_CONST, keyIdx)

		switch v := value.(type) {
		case ast.NumberExpr:
			valIdx := c.addConst(v.Value)
			c.emit(OP_CONST, valIdx)
		case ast.StringExpr:
			valIdx := c.addConst(v.Value)
			c.emit(OP_CONST, valIdx)
		case ast.ArrayLiteralExpr:
			c.compileExpr(v)
		default:
			panic(fmt.Sprintf("Unsupported policy rule value type: %T", v))
		}

		c.emit(OP_SET_PROPERTY)
	}

	c.emit(OP_POLICY_DECLARE, byte(identifierIdx))
	c.emit(OP_STORE, byte(c.Symbols[s.Identifier.(ast.SymbolExpr).Value]))
}

func (c *Compiler) compileTypeDeclareStmt(s ast.TypeDeclareStmt) {
	typeName := s.Name.(ast.SymbolExpr).Value
	c.allocSlot(typeName)

	// Register the type in the type registry
	typeMeta := TypeMeta{
		Fields: make(map[string]string),
	}
	for key, value := range s.Fields {
		if sym, ok := value.(ast.SymbolExpr); ok {
			typeMeta.Fields[key] = sym.Value
		}
	}
	c.Types[typeName] = typeMeta

	identifierIdx := c.findConst(s.Name)
	if identifierIdx == 255 {
		identifierIdx = c.addConst(s.Name)
	}

	c.emit(OP_CONST, identifierIdx)
	c.emit(OP_PUSH_OBJECT)

	for key, value := range s.Fields {
		keyIdx := c.addConst(key)
		c.emit(OP_CONST, keyIdx)

		switch v := value.(type) {
		case ast.NumberExpr:
			valIdx := c.addConst(v.Value)
			c.emit(OP_CONST, valIdx)
		case ast.StringExpr:
			valIdx := c.addConst(v.Value)
			c.emit(OP_CONST, valIdx)
		case ast.SymbolExpr:
			valIdx := c.addConst(v.Value)
			c.emit(OP_CONST, valIdx)
		default:
			panic(fmt.Sprintf("Unsupported type value: %T", v))
		}

		c.emit(OP_SET_PROPERTY)
	}

	c.emit(OP_TYPE_DECLARE, byte(identifierIdx))
	c.emit(OP_STORE, byte(c.Symbols[typeName]))
}

func (c *Compiler) compileEmitStmt(s ast.EmitStmt) {
	c.compileExpr(s.EventName)

	if s.Arguments != nil {
		c.compileExpr(s.Arguments)
	} else {
		c.emit(OP_PUSH_OBJECT)
	}

	c.emit(OP_EMIT, 0)
}

func (c *Compiler) compileTryCatchStmt(s ast.TryCatchStmt) {
	errSlot := c.allocSlot(fmt.Sprintf("__catch_err_%d__", c.NextSlot))

	tryPos := c.currentPos()
	c.emit(OP_TRY, 0, 0, byte(errSlot))
	for _, stmt := range s.TryBlock {
		c.compileStmt(stmt)
	}
	c.emit(OP_END_TRY)

	jmpEndPos := c.currentPos()
	c.emit(OP_JMP, 0, 0)

	handlerPos := c.currentPos()
	prevSlot, hadPrev := c.Symbols[s.CatchVar]
	c.Symbols[s.CatchVar] = errSlot

	for _, stmt := range s.CatchBlock {
		c.compileStmt(stmt)
	}

	if hadPrev {
		c.Symbols[s.CatchVar] = prevSlot
	} else {
		delete(c.Symbols, s.CatchVar)
	}

	endPos := c.currentPos()

	c.patchJump(tryPos+1, handlerPos)
	c.patchJump(jmpEndPos+1, endPos)
}
