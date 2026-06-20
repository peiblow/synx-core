package lexer

import "fmt"

type TokenType int

const (
	EOF TokenType = iota
	NULL
	TRUE
	FALSE
	NUMBER
	HEX_NUMBER
	STRING
	IDENTIFIER
	LIBRARY
	IMPORT
	CONTRACT
	AGENT
	POLICY
	TYPE
	EMIT
	REQUIRE
	HASH
	GET_ENV
	NONCE
	// Grouping & Braces
	OPEN_BRACKET
	CLOSE_BRACKET
	OPEN_CURLY
	CLOSE_CURLY
	OPEN_PAREN
	CLOSE_PAREN
	// Equivalence
	ASSIGNMENT
	EQUALS
	NOT_EQUALS
	NOT
	// Conditional
	LESS
	LESS_EQUALS
	GREATER
	GREATER_EQUALS
	// Logical
	OR
	AND
	// Symbols
	DOT
	DOT_DOT
	SEMI_COLON
	BREAK_LINE
	COLON
	QUESTION
	COMMA
	// Shorthand
	PLUS_PLUS
	MINUS_MINUS
	PLUS_EQUALS
	MINUS_EQUALS
	NULLISH_ASSIGNMENT // ??=
	// Maths
	PLUS
	DASH
	SLASH
	STAR
	PERCENT
	// Reserved Keywords
	LET
	CONST
	FN
	IF
	ELSE
	FOREACH
	WHILE
	FOR
	TRY
	ERROR
	CATCH
	RETURN
	THIS
	// Misc
	NUM_TOKENS
)

// IsKeyword returns true if the token type is a reserved keyword
// (including synx-specific keywords like contract, agent, hash, nonce, etc.).
func IsKeyword(tp TokenType) bool {
	return (tp >= CONTRACT && tp <= NONCE) || (tp >= LET && tp <= THIS) ||
		tp == NULL || tp == TRUE || tp == FALSE
}

// BUG FIX: `true`, `false`, and `null` were defined as TokenTypes but were
// missing from reserved_lu, so they were being emitted as IDENTIFIER tokens
// instead of their correct types.
var reserved_lu map[string]TokenType = map[string]TokenType{
	// Control flow
	"fn":      FN,
	"if":      IF,
	"else":    ELSE,
	"foreach": FOREACH,
	"while":   WHILE,
	"for":     FOR,
	"try":     TRY,
	"catch":   CATCH,
	"Error":   ERROR,
	"return":  RETURN,
	"this":    THIS,
	// Variables
	"const": CONST,
	"let":   LET,
	// Synx-specific
	"library":  LIBRARY,
	"import":   IMPORT,
	"contract": CONTRACT,
	"agent":    AGENT,
	"policy":   POLICY,
	"type":     TYPE,
	"emit":     EMIT,
	"require":  REQUIRE,
	"nonce":    NONCE,
	"hash":     HASH,
	"getEnv":   GET_ENV,
	// Literals — were missing, caused `true`/`false`/`null` to tokenize as IDENTIFIER
	"true":  TRUE,
	"false": FALSE,
	"null":  NULL,
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
}

func NewToken(t TokenType, value string, line int) Token {
	return Token{
		Type:    t,
		Literal: value,
		Line:    line,
	}
}

func (token Token) Debug() {
	if token.Type == IDENTIFIER || token.Type == NUMBER || token.Type == STRING {
		fmt.Printf("%s(%s)\n", TokenTypeString(token.Type), token.Literal)
	} else {
		fmt.Printf("%s()\n", TokenTypeString(token.Type))
	}
}

func TokenTypeString(tp TokenType) string {
	switch tp {
	case EOF:
		return "eof"
	case NULL:
		return "null"
	case NUMBER:
		return "number"
	case HEX_NUMBER:
		return "hex_number"
	case STRING:
		return "string"
	case TRUE:
		return "true"
	case FALSE:
		return "false"
	case IDENTIFIER:
		return "identifier"
	case LIBRARY:
		return "library"
	case IMPORT:
		return "import"
	case CONTRACT:
		return "contract"
	case AGENT:
		return "agent"
	case POLICY:
		return "policy"
	case TYPE:
		return "type"
	case EMIT:
		return "emit"
	case GET_ENV:
		return "get_env"
	case HASH:
		return "hash"
	case NONCE:
		return "nonce"
	case REQUIRE:
		return "require"
	case OPEN_BRACKET:
		return "open_bracket"
	case CLOSE_BRACKET:
		return "close_bracket"
	case OPEN_CURLY:
		return "open_curly"
	case CLOSE_CURLY:
		return "close_curly"
	case OPEN_PAREN:
		return "open_paren"
	case CLOSE_PAREN:
		return "close_paren"
	case ASSIGNMENT:
		return "assignment"
	case EQUALS:
		return "equals"
	case NOT_EQUALS:
		return "not_equals"
	case NOT:
		return "not"
	case LESS:
		return "less"
	case LESS_EQUALS:
		return "less_equals"
	case GREATER:
		return "greater"
	case GREATER_EQUALS:
		return "greater_equals"
	case OR:
		return "or"
	case AND:
		return "and"
	case DOT:
		return "dot"
	case DOT_DOT:
		return "dot_dot"
	case SEMI_COLON:
		return "semi_colon"
	case BREAK_LINE:
		return "break_line"
	case COLON:
		return "colon"
	case QUESTION:
		return "question"
	case COMMA:
		return "comma"
	case PLUS_PLUS:
		return "plus_plus"
	case MINUS_MINUS:
		return "minus_minus"
	case PLUS_EQUALS:
		return "plus_equals"
	case MINUS_EQUALS:
		return "minus_equals"
	case NULLISH_ASSIGNMENT:
		return "nullish_assignment"
	case PLUS:
		return "plus"
	case DASH:
		return "dash"
	case SLASH:
		return "slash"
	case STAR:
		return "star"
	case PERCENT:
		return "percent"
	case FN:
		return "fn"
	case IF:
		return "if"
	case ELSE:
		return "else"
	case FOREACH:
		return "foreach"
	case FOR:
		return "for"
	case TRY:
		return "try"
	case ERROR:
		return "Error"
	case CATCH:
		return "catch"
	case WHILE:
		return "while"
	case LET:
		return "let"
	case CONST:
		return "const"
	case RETURN:
		return "return"
	case THIS:
		return "this"
	default:
		return fmt.Sprintf("unknown(%d)", tp)
	}
}
