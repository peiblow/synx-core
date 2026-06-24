package parser

import (
	"github.com/peiblow/vvm/ast"
	"github.com/peiblow/vvm/lexer"
)

type binding_power int

const (
	defalt_bp binding_power = iota
	comma
	assignment
	logical
	relational
	additive
	multiplicative
	unary
	call
	member
	primary
)

type stmt_handler func(p *parser) ast.Stmt
type nud_handler func(p *parser) ast.Expr
type led_handler func(p *parser, left ast.Expr, bp binding_power) ast.Expr

type stmt_lookup map[lexer.TokenType]stmt_handler
type nud_lookup map[lexer.TokenType]nud_handler
type led_lookup map[lexer.TokenType]led_handler
type bp_lookup map[lexer.TokenType]binding_power

var bp_lu = bp_lookup{}
var nud_lu = nud_lookup{}
var led_lu = led_lookup{}
var stmt_lu = stmt_lookup{}

func led(tp lexer.TokenType, bp binding_power, led_fn led_handler) {
	bp_lu[tp] = bp
	led_lu[tp] = led_fn
}

func nud(tp lexer.TokenType, nud_fn nud_handler) {
	nud_lu[tp] = nud_fn
}

func stmt(tp lexer.TokenType, stmt_fn stmt_handler) {
	bp_lu[tp] = defalt_bp
	stmt_lu[tp] = stmt_fn
}

func createTokenLookups() {
	led(lexer.ASSIGNMENT, assignment, parse_assignment)
	led(lexer.PLUS_EQUALS, assignment, parse_assignment)
	led(lexer.MINUS_EQUALS, assignment, parse_assignment)
	led(lexer.PLUS_PLUS, assignment, parse_incdec_expr)
	led(lexer.OPEN_BRACKET, call, parse_array_access_item_expr)
	led(lexer.OPEN_PAREN, call, parse_call_expr)
	led(lexer.DOT, call, parse_member_expr)

	led(lexer.AND, logical, parse_binary_expr)
	led(lexer.OR, logical, parse_binary_expr)
	led(lexer.DOT_DOT, logical, parse_binary_expr)

	led(lexer.LESS, relational, parse_binary_expr)
	led(lexer.LESS_EQUALS, relational, parse_binary_expr)
	led(lexer.GREATER, relational, parse_binary_expr)
	led(lexer.GREATER_EQUALS, relational, parse_binary_expr)
	led(lexer.EQUALS, relational, parse_binary_expr)
	led(lexer.NOT_EQUALS, relational, parse_binary_expr)

	led(lexer.PLUS, additive, parse_binary_expr)
	led(lexer.DASH, additive, parse_binary_expr)
	led(lexer.STAR, multiplicative, parse_binary_expr)
	led(lexer.SLASH, multiplicative, parse_binary_expr)
	led(lexer.PERCENT, multiplicative, parse_binary_expr)

	nud(lexer.NUMBER, parse_primary_expr)
	nud(lexer.HEX_NUMBER, parse_primary_expr)
	nud(lexer.STRING, parse_primary_expr)
	nud(lexer.IDENTIFIER, parse_primary_expr)
	nud(lexer.DASH, parse_prefix_expr)
	nud(lexer.OPEN_PAREN, grouping_expr)
	nud(lexer.OPEN_BRACKET, parse_literal_array_expr)
	nud(lexer.OPEN_CURLY, parse_obj_assignment_expr)
	nud(lexer.NULL, parse_null_expr)
	nud(lexer.THIS, parse_this_expr)
	nud(lexer.GET_ENV, parse_get_env_expr)
	nud(lexer.NONCE, parse_nonce_expr)
	nud(lexer.HASH, parse_hash_expr)
	nud(lexer.TRUE, parse_bool_expr)
	nud(lexer.FALSE, parse_bool_expr)
	nud(lexer.ERROR, parse_error_expr)

	stmt(lexer.IMPORT, parse_import_stmt)
	stmt(lexer.LIBRARY, parse_library_stmt)
	stmt(lexer.CONTRACT, parse_contract_decl)
	stmt(lexer.LET, parse_var_decl)
	stmt(lexer.CONST, parse_var_decl)
	stmt(lexer.IF, parse_if_stmt)
	stmt(lexer.WHILE, parse_while_loop_stmt)
	stmt(lexer.FOR, parse_for_loop_stmt)
	stmt(lexer.FN, parse_func_stmt)
	stmt(lexer.RETURN, parse_return_stmt)
	stmt(lexer.REQUIRE, parse_require_stmt)
	stmt(lexer.AGENT, parse_agent_stmt)
	stmt(lexer.POLICY, parse_policy_stmt)
	stmt(lexer.TYPE, parse_type_stmt)
	stmt(lexer.EMIT, parse_emit_stmt)
	stmt(lexer.TRY, parse_try_stmt)
	stmt(lexer.TOOL, parse_tool_stmt)
}
