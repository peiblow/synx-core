package parser

import (
	"fmt"

	"github.com/peiblow/vvm/ast"
	"github.com/peiblow/vvm/lexer"
)

func parse_block(p *parser) ast.BlockStmt {
	p.expect(lexer.OPEN_CURLY)

	body := make([]ast.Stmt, 0)
	for p.hasTokens() && p.currentTokenType() != lexer.CLOSE_CURLY {
		body = append(body, parse_stmt(p))
	}

	p.expect(lexer.CLOSE_CURLY)

	return ast.BlockStmt{
		Body: body,
	}
}

func parse_stmt(p *parser) ast.Stmt {
	stmt_fn, exists := stmt_lu[p.currentTokenType()]

	if exists {
		return stmt_fn(p)
	}

	expr := parse_expr(p, defalt_bp)

	return ast.ExpressionStmt{
		Expression: expr,
	}
}

func parse_arguments(p *parser) []ast.ArgsStmt {
	p.expect(lexer.OPEN_PAREN)

	body := []ast.ArgsStmt{}
	for p.currentTokenType() != lexer.CLOSE_PAREN {
		expr := parse_expr(p, defalt_bp)
		p.expect(lexer.COLON)
		exprType := parse_expr(p, defalt_bp)

		body = append(body, ast.ArgsStmt{
			ArgName: expr,
			ArgType: exprType,
		})

		if p.currentTokenType() == lexer.COMMA {
			p.advance()
		}
	}

	p.expect(lexer.CLOSE_PAREN)
	return body
}

func parse_import_stmt(p *parser) ast.Stmt {
	p.expect(lexer.IMPORT)
	importId := p.expect(lexer.IDENTIFIER).Literal
	p.advance()

	path := p.expect(lexer.STRING).Literal

	return ast.ImportStmt{
		Identifier: importId,
		Path:       path,
	}
}

func parse_library_stmt(p *parser) ast.Stmt {
	p.expect(lexer.LIBRARY)
	libName := p.expect(lexer.IDENTIFIER).Literal
	body := parse_block(p)

	return ast.LibraryStmt{
		Identifier: libName,
		Body:       body.Body,
	}
}

func parse_contract_decl(p *parser) ast.Stmt {
	p.expect(lexer.CONTRACT)
	contractName := p.currentToken().Literal
	p.advance()

	body := parse_block(p)

	return ast.ContractStmt{
		Identifier: contractName,
		Body:       body.Body,
	}
}

func parse_var_decl(p *parser) ast.Stmt {
	var assignmentValue ast.Expr
	var varType ast.Type

	isConst := p.advance().Type == lexer.CONST
	varName := p.expectError(lexer.IDENTIFIER, "Inside variable declaration expected to find variable name").Literal

	if p.currentTokenType() == lexer.COLON {
		p.advance()
		varType = parse_type(p, defalt_bp)
	}

	if p.currentTokenType() != lexer.ASSIGNMENT && isConst {
		panic("A constant should be initilized with an default value")
	} else {
		p.expect(lexer.ASSIGNMENT)
		assignmentValue = parse_expr(p, assignment)
	}

	return ast.VarDeclStmt{
		Identifier:    varName,
		AssignedValue: assignmentValue,
		Constant:      isConst,
		ExplicityType: varType,
	}
}

func parse_if_stmt(p *parser) ast.Stmt {
	p.expect(lexer.IF)
	p.expect(lexer.OPEN_PAREN)
	condition := parse_expr(p, defalt_bp)
	p.expect(lexer.CLOSE_PAREN)

	thenBlock := parse_block(p)
	var elseBlock ast.Stmt
	if p.currentTokenType() == lexer.ELSE {
		p.advance()
		if p.currentTokenType() == lexer.IF {
			elseBlock = parse_if_stmt(p)
		} else {
			elseBlock = parse_block(p)
		}
	}

	return ast.IfStmt{
		Condition: condition,
		Then:      thenBlock,
		Else:      elseBlock,
	}
}

func parse_while_loop_stmt(p *parser) ast.Stmt {
	p.advance()
	p.expect(lexer.OPEN_PAREN)
	cond := parse_expr(p, defalt_bp)
	p.expect(lexer.CLOSE_PAREN)
	body := parse_block(p).Body

	return ast.WhileStmt{
		Condition: cond,
		Body:      body,
	}
}

func parse_for_loop_stmt(p *parser) ast.Stmt {
	p.advance()
	p.expect(lexer.OPEN_PAREN)

	init := parse_stmt(p)
	p.expect(lexer.SEMI_COLON)

	cond := parse_expr(p, defalt_bp)
	p.expect(lexer.SEMI_COLON)

	post := parse_stmt(p)
	p.expect(lexer.CLOSE_PAREN)

	body := parse_block(p).Body

	return ast.ForStmt{
		Init:      init,
		Condition: cond,
		Post:      post,
		Body:      body,
	}
}

func parse_func_stmt(p *parser) ast.Stmt {
	var returnType ast.Type

	p.expect(lexer.FN)
	name := ast.ExpressionStmt{Expression: ast.SymbolExpr{Value: p.advance().Literal}}

	args := parse_arguments(p)

	if p.currentTokenType() != lexer.COLON {
		panic("Functions should have a Return Type specified")
	} else {
		p.advance()
		returnType = parse_type(p, defalt_bp)
	}

	body := parse_block(p)

	return ast.FuncStmt{
		Name:       name,
		Arguments:  args,
		Body:       body,
		ReturnType: returnType,
	}
}

func parse_return_stmt(p *parser) ast.Stmt {
	p.expect(lexer.RETURN)

	if p.currentTokenType() == lexer.CLOSE_CURLY || p.currentTokenType() == lexer.EOF {
		return ast.ReturnStmt{Value: nil}
	}

	value := parse_expr(p, defalt_bp)

	return ast.ReturnStmt{
		Value: value,
	}
}

func parse_require_stmt(p *parser) ast.Stmt {
	p.expect(lexer.REQUIRE)
	p.expect(lexer.OPEN_PAREN)

	condition := parse_expr(p, defalt_bp)
	p.expect(lexer.SEMI_COLON)

	message := parse_expr(p, defalt_bp)

	p.expect(lexer.CLOSE_PAREN)
	return ast.RequireStmt{
		Condition: condition,
		Message:   message,
	}
}

func parse_agent_stmt(p *parser) ast.Stmt {
	p.expect(lexer.AGENT)
	agentName := parse_expr(p, defalt_bp)
	p.expect(lexer.OPEN_CURLY)

	var version, owner, purpose ast.Expr
	var tools []ast.ToolStmt
	var model ast.ModelStmt
	var behavior ast.BehaviorStmt
	var skills ast.SkillsStmt
	var triggers []ast.Trigger

	for p.currentTokenType() != lexer.CLOSE_CURLY {
		if p.currentTokenType() == lexer.MODEL {
			stmt := parse_model_stmt(p)
			model = stmt.(ast.ModelStmt)
			continue
		}

		if p.currentTokenType() == lexer.BEHAVIOR {
			stmt := parse_behavior_stmt(p)
			behavior = stmt.(ast.BehaviorStmt)
			continue
		}

		if p.currentTokenType() == lexer.SKILLS {
			stmt := parse_skills_stmt(p)
			skills = stmt.(ast.SkillsStmt)
			continue
		}

		if p.currentTokenType() == lexer.TRIGGER {
			triggers = append(triggers, parse_trigger_stmt(p))
			continue
		}

		if p.currentTokenType() == lexer.TOOL {
			stmt := parse_tool_stmt(p)
			tools = append(tools, stmt.(ast.ToolStmt))
			continue
		}

		field := p.expectIdentifierOrKeyword("expected field in agent declaration")
		p.expect(lexer.COLON)

		switch field {
		case "version":
			version = parse_expr(p, defalt_bp)
		case "owner":
			owner = parse_expr(p, defalt_bp)
		case "purpose":
			purpose = parse_expr(p, defalt_bp)
		default:
			panic(fmt.Sprintf("[linha %d] unknown agent field: %s", p.currentToken().Line, field))
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return ast.AgentStmt{
		Identifier: agentName,
		Version:    version,
		Owner:      owner,
		Purpose:    purpose,
		Tools:      tools,
		Model:      model,
		Behavior:   behavior,
		Skills:     skills,
		Triggers:   triggers,
	}
}

func parse_trigger_stmt(p *parser) ast.Trigger {
	p.expect(lexer.TRIGGER)
	p.expect(lexer.OPEN_CURLY)

	var triggerType ast.Expr
	fields := make(map[string]ast.Expr)

	for p.currentTokenType() != lexer.CLOSE_CURLY {
		field := p.expectIdentifierOrKeyword("expected field in trigger declaration")
		p.expect(lexer.COLON)

		value := parse_expr(p, defalt_bp)
		if field == "type" {
			triggerType = value
		} else {
			fields[field] = value
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return ast.Trigger{
		Type:   triggerType,
		Fields: fields,
	}
}

func parse_policy_stmt(p *parser) ast.Stmt {
	p.expect(lexer.POLICY)
	policyName := parse_expr(p, defalt_bp)

	p.expect(lexer.OPEN_CURLY)

	rules := make(map[string]ast.Expr)
	for p.currentTokenType() != lexer.CLOSE_CURLY {
		ruleKey := p.expectError(lexer.IDENTIFIER, "Expected rule identifier in policy declaration").Literal
		p.expect(lexer.COLON)
		ruleValue := parse_expr(p, defalt_bp)
		rules[ruleKey] = ruleValue
	}

	p.expect(lexer.CLOSE_CURLY)

	return ast.PolicyStmt{
		Identifier: policyName,
		Rules:      rules,
	}
}

func parse_type_stmt(p *parser) ast.Stmt {
	p.expect(lexer.TYPE)
	typeName := parse_expr(p, defalt_bp)

	p.expect(lexer.OPEN_CURLY)

	fields := make(map[string]ast.Expr)
	for p.currentTokenType() != lexer.CLOSE_CURLY {
		fieldKey := p.expectIdentifierOrKeyword("Expected type identifier in type declaration")
		p.expect(lexer.COLON)
		fieldType := parse_expr(p, defalt_bp)
		fields[fieldKey] = fieldType
	}

	p.expect(lexer.CLOSE_CURLY)

	return ast.TypeDeclareStmt{
		Name:   typeName,
		Fields: fields,
	}
}

func parse_emit_stmt(p *parser) ast.Stmt {
	p.expect(lexer.EMIT)
	p.expect(lexer.OPEN_PAREN)

	eventName := parse_expr(p, defalt_bp)

	var args ast.Expr
	if p.currentTokenType() == lexer.COMMA {
		p.advance()
		args = parse_expr(p, defalt_bp)
	}

	p.expect(lexer.CLOSE_PAREN)

	return ast.EmitStmt{
		EventName: eventName,
		Arguments: args,
	}
}

func parse_try_stmt(p *parser) ast.Stmt {
	p.expect(lexer.TRY)
	tryBlock := parse_block(p)

	var catchVar string
	var catchBlock ast.BlockStmt
	if p.currentTokenType() == lexer.CATCH {
		p.expect(lexer.CATCH)
		p.expect(lexer.OPEN_PAREN)
		catchVar = p.expectIdentifierOrKeyword("Expected error variable name in catch statement")
		p.expect(lexer.CLOSE_PAREN)

		catchBlock = parse_block(p)
	}

	return ast.TryCatchStmt{
		TryBlock:   tryBlock.Body,
		CatchVar:   catchVar,
		CatchBlock: catchBlock.Body,
	}
}

func parse_tool_stmt(p *parser) ast.Stmt {
	p.expect(lexer.TOOL)
	toolName := parse_expr(p, defalt_bp)
	p.expect(lexer.OPEN_CURLY)

	var description ast.Expr
	var steps []ast.ToolStep

	for p.currentTokenType() != lexer.CLOSE_CURLY {
		field := p.expectIdentifierOrKeyword("expected field in tool declaration")
		p.expect(lexer.COLON)

		switch field {
		case "description":
			description = parse_expr(p, defalt_bp)
		case "steps":
			steps = parse_tool_steps(p)
		default:
			panic(fmt.Sprintf("[linha %d] unknown tool field: %s", p.currentToken().Line, field))
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return ast.ToolStmt{
		Name:        toolName,
		Description: description,
		Steps:       steps,
	}
}

func parse_tool_steps(p *parser) []ast.ToolStep {
	p.expect(lexer.OPEN_BRACKET)

	var steps []ast.ToolStep
	for p.currentTokenType() != lexer.CLOSE_BRACKET {
		steps = append(steps, parse_tool_step(p))
		if p.currentTokenType() == lexer.COMMA {
			p.advance()
		}
	}

	p.expect(lexer.CLOSE_BRACKET)
	return steps
}

func parse_tool_step(p *parser) ast.ToolStep {
	p.expect(lexer.OPEN_CURLY)

	var step ast.ToolStep
	for p.currentTokenType() != lexer.CLOSE_CURLY {
		field := p.expectIdentifierOrKeyword("expected field in tool step")
		p.expect(lexer.COLON)

		switch field {
		case "function":
			step.Function = parse_member_ref(p)
		case "input":
			step.Input = parse_tool_step_input(p)
		case "action":
			step.Action = parse_tool_action(p)
		default:
			panic(fmt.Sprintf("[linha %d] unknown step field: %s", p.currentToken().Line, field))
		}

		if p.currentTokenType() == lexer.COMMA {
			p.advance()
		}
	}

	p.expect(lexer.CLOSE_CURLY)
	return step
}

func parse_member_ref(p *parser) string {
	ref := p.expect(lexer.IDENTIFIER).Literal
	for p.currentTokenType() == lexer.DOT {
		p.advance()
		ref += "." + p.expect(lexer.IDENTIFIER).Literal
	}
	return ref
}

func parse_tool_action(p *parser) ast.ToolAction {
	p.expect(lexer.OPEN_CURLY)

	var actionType string
	var fields = make(map[string]any)
	for p.currentTokenType() != lexer.CLOSE_CURLY {
		field := p.expectIdentifierOrKeyword("expected field in tool action")
		p.expect(lexer.COLON)

		switch p.currentTokenType() {
		case lexer.STRING:
			tok := p.expectError(lexer.STRING, "")
			fields[field] = tok.Literal
		default:
			fields[field] = parse_expr(p, defalt_bp)
		}

		if field == "type" {
			actionType = fields[field].(string)
		}

		if p.currentTokenType() == lexer.COMMA {
			p.advance()
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return build_tool_action(actionType, fields)
}

func parse_tool_step_input(p *parser) []ast.ToolStepInput {
	p.expect(lexer.OPEN_CURLY)

	var inputs []ast.ToolStepInput
	for p.currentTokenType() != lexer.CLOSE_CURLY {
		inputName := p.expectIdentifierOrKeyword("expected field in tool step input")
		p.expect(lexer.COLON)
		inputType := parse_expr(p, defalt_bp)

		inputs = append(inputs, ast.ToolStepInput{Name: inputName, Type: inputType})

		if p.currentTokenType() == lexer.COMMA {
			p.advance()
		}
	}

	p.expect(lexer.CLOSE_CURLY)
	return inputs
}

func parse_model_stmt(p *parser) ast.Stmt {
	p.expect(lexer.MODEL)
	p.expect(lexer.OPEN_CURLY)

	var provider, name, temperature, maxTokens ast.Expr

	for p.currentTokenType() != lexer.CLOSE_CURLY {
		field := p.expectIdentifierOrKeyword("expected field in model declaration")
		p.expect(lexer.COLON)

		switch field {
		case "provider":
			provider = parse_expr(p, defalt_bp)
		case "name":
			name = parse_expr(p, defalt_bp)
		case "temperature":
			temperature = parse_expr(p, defalt_bp)
		case "maxTokens":
			maxTokens = parse_expr(p, defalt_bp)
		default:
			panic(fmt.Sprintf("[linha %d] unknown model field: %s", p.currentToken().Line, field))
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return ast.ModelStmt{
		Name:        name,
		Provider:    provider,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
}

func parse_behavior_stmt(p *parser) ast.Stmt {
	p.expect(lexer.BEHAVIOR)
	p.expect(lexer.OPEN_CURLY)

	var systemPrompt, maxSteps, onDeny, onError ast.Expr

	for p.currentTokenType() != lexer.CLOSE_CURLY {
		field := p.expectIdentifierOrKeyword("expected field in behavior declaration")
		p.expect(lexer.COLON)

		switch field {
		case "systemPrompt":
			systemPrompt = parse_expr(p, defalt_bp)
		case "maxSteps":
			maxSteps = parse_expr(p, defalt_bp)
		case "onDeny":
			onDeny = parse_expr(p, defalt_bp)
		case "onError":
			onError = parse_expr(p, defalt_bp)
		default:
			panic(fmt.Sprintf("[linha %d] unknown behavior field: %s", p.currentToken().Line, field))
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return ast.BehaviorStmt{
		SystemPrompt: systemPrompt,
		MaxSteps:     maxSteps,
		OnDeny:       onDeny,
		OnError:      onError,
	}
}

func parse_skills_stmt(p *parser) ast.Stmt {
	p.expect(lexer.SKILLS)
	p.expect(lexer.OPEN_BRACKET)

	var skills []ast.Skill
	for p.currentTokenType() != lexer.CLOSE_BRACKET {
		skill := parse_skill(p)
		skills = append(skills, skill)

		if p.currentTokenType() == lexer.COMMA {
			p.advance()
		}
	}

	p.expect(lexer.CLOSE_BRACKET)

	return ast.SkillsStmt{
		Skills: skills,
	}
}

func parse_skill(p *parser) ast.Skill {
	p.expect(lexer.OPEN_CURLY)

	var name, content ast.Expr
	var uses []ast.Expr

	for p.currentTokenType() != lexer.CLOSE_CURLY {
		field := p.expectIdentifierOrKeyword("expected field in skill declaration")
		p.expect(lexer.COLON)

		switch field {
		case "name":
			name = parse_expr(p, defalt_bp)
		case "content":
			content = parse_expr(p, defalt_bp)
		case "uses":
			uses = parse_skill_uses(p)
		default:
			panic(fmt.Sprintf("[linha %d] unknown skill field: %s", p.currentToken().Line, field))
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return ast.Skill{
		Name:    name,
		Content: content,
		Uses:    uses,
	}
}

func parse_skill_uses(p *parser) []ast.Expr {
	p.expect(lexer.OPEN_BRACKET)

	var uses []ast.Expr
	for p.currentTokenType() != lexer.CLOSE_BRACKET {
		use := parse_expr(p, defalt_bp)
		uses = append(uses, use)

		if p.currentTokenType() == lexer.COMMA {
			p.advance()
		}
	}

	p.expect(lexer.CLOSE_BRACKET)
	return uses
}

func build_tool_action(actionType string, fields map[string]any) ast.ToolAction {
	switch actionType {

	case "http":
		return ast.HttpAction{
			Method:  fields["method"].(string),
			Url:     fields["url"].(ast.Expr),
			Headers: asObjectProps(fields["headers"]),
			Body:    asExprOrNil(fields["body"]),
		}

	case "filesystem":
		return ast.FilesystemAction{
			Operation: fields["operation"].(string),
			Path:      fields["path"].(ast.Expr),
		}

	case "shell":
		return ast.ShellAction{
			Command: fields["command"].(string),
			Args:    asExprList(fields["args"]),
		}

	case "dispatch":
		return ast.DispatchAction{
			Agent: asExpr(fields["agent"]),
		}

	default:
		panic(fmt.Sprintf("unknown tool action type: %s", actionType))
	}
}

func asExprList(v any) []ast.Expr {
	switch e := v.(type) {
	case []ast.Expr:
		return e
	case ast.ArrayLiteralExpr:
		return e.Items
	default:
		panic(fmt.Sprintf("expected array of expressions, got %T", v))
	}
}

func asObjectProps(v any) []ast.ObjectPropertyExpr {
	if obj, ok := v.(ast.ObjectAssignmentExpr); ok {
		return obj.Fields
	}
	return nil
}

func asExprOrNil(v any) ast.Expr {
	if e, ok := v.(ast.Expr); ok {
		return e
	}
	return nil
}

func asExpr(v any) ast.Expr {
	switch e := v.(type) {
	case ast.Expr:
		return e
	case string:
		return ast.StringExpr{Value: e}
	default:
		panic(fmt.Sprintf("expected string or expression, got %T", v))
	}
}
