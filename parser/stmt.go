package parser

import (
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

	parse_expr(p, defalt_bp)
	p.expect(lexer.COLON)
	version := parse_expr(p, defalt_bp)

	parse_expr(p, defalt_bp)
	p.expect(lexer.COLON)
	owner := parse_expr(p, defalt_bp)

	parse_expr(p, defalt_bp)
	p.expect(lexer.COLON)
	purpose := parse_expr(p, defalt_bp)

	p.expect(lexer.CLOSE_CURLY)

	return ast.AgentStmt{
		Identifier: agentName,
		Version:    version,
		Owner:      owner,
		Purpose:    purpose,
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
