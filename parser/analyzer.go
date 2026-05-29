package parser

import (
	"fmt"
	"strings"

	"github.com/peiblow/vvm/ast"
)

// ─────────────────────────────────────────────────────────────────────────────
// Built-in types always valid in Synx
// ─────────────────────────────────────────────────────────────────────────────
var builtinTypes = map[string]bool{
	"String":  true,
	"Address": true,
	"UInt":    true,
	"Bool":    true,
	"Void":    true,
	"Event":   true,
}

var builtinFunctions = map[string]bool{
	"len":     true,
	"print":   true,
	"require": true,
	"emit":    true,
}

// ─────────────────────────────────────────────────────────────────────────────
// SemanticError
// ─────────────────────────────────────────────────────────────────────────────
type SemanticError struct {
	Message string
}

func (e SemanticError) Error() string {
	return fmt.Sprintf("[semantic] %s", e.Message)
}

// ─────────────────────────────────────────────────────────────────────────────
// AnalysisResult
// ─────────────────────────────────────────────────────────────────────────────
type AnalysisResult struct {
	Errors []SemanticError
}

func (r AnalysisResult) HasErrors() bool {
	return len(r.Errors) > 0
}

func (r AnalysisResult) Format() string {
	if !r.HasErrors() {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d semantic error(s) found:\n", len(r.Errors)))
	for _, e := range r.Errors {
		sb.WriteString("  " + e.Error() + "\n")
	}
	return sb.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Analyzer
// ─────────────────────────────────────────────────────────────────────────────
type Analyzer struct {
	errors            []SemanticError
	userTypes         map[string]map[string]string
	declaredFunctions map[string]int
	declaredAgents    map[string]bool
	declaredPolicies  map[string]bool
}

func newAnalyzer() *Analyzer {
	return &Analyzer{
		userTypes:         make(map[string]map[string]string),
		declaredFunctions: make(map[string]int),
		declaredAgents:    make(map[string]bool),
		declaredPolicies:  make(map[string]bool),
	}
}

func (a *Analyzer) addError(format string, args ...interface{}) {
	a.errors = append(a.errors, SemanticError{
		Message: fmt.Sprintf(format, args...),
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Entry point — receives the ast.BlockStmt from parser.Parse()
// ─────────────────────────────────────────────────────────────────────────────
func Analyze(block ast.BlockStmt) AnalysisResult {
	a := newAnalyzer()

	for _, stmt := range block.Body {
		if contract, ok := stmt.(ast.ContractStmt); ok {
			a.analyzeContract(contract)

			return AnalysisResult{Errors: a.errors}
		}
	}

	a.addError("no 'contract' declaration found at top level")
	return AnalysisResult{Errors: a.errors}
}

func AnalyzeProgram(main ast.BlockStmt, modules []ast.BlockStmt) AnalysisResult {
	a := newAnalyzer()

	for _, mod := range modules {
		a.registerModule(mod)
	}

	for _, stmt := range main.Body {
		if contract, ok := stmt.(ast.ContractStmt); ok {
			a.analyzeContract(contract)
			return AnalysisResult{Errors: a.errors}
		}
	}

	a.addError("no 'contract' declaration found at top level")
	return AnalysisResult{Errors: a.errors}
}

func (a *Analyzer) registerModule(mod ast.BlockStmt) {
	for _, node := range mod.Body {
		if lib, ok := node.(ast.LibraryStmt); ok {
			for _, inner := range lib.Body {
				a.registerDecl(inner)
			}
			continue
		}
		a.registerDecl(node)
	}
}

func (a *Analyzer) registerDecl(node ast.Stmt) {
	switch s := node.(type) {
	case ast.TypeDeclareStmt:
		a.registerType(s)
	case ast.AgentStmt:
		a.declaredAgents[symbolName(s.Identifier)] = true
	case ast.PolicyStmt:
		a.declaredPolicies[symbolName(s.Identifier)] = true
	case ast.FuncStmt:
		a.declaredFunctions[symbolName(s.Name)] = len(s.Arguments)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Contract — two-pass analysis
// ─────────────────────────────────────────────────────────────────────────────
func (a *Analyzer) analyzeContract(contract ast.ContractStmt) {
	if contract.Identifier == "" {
		a.addError("contract is missing a name")
	}

	for _, node := range contract.Body {
		a.registerDecl(node)
	}

	for _, node := range contract.Body {
		switch s := node.(type) {
		case ast.TypeDeclareStmt:
			a.analyzeTypeDecl(s)
		case ast.AgentStmt:
			a.analyzeAgent(s)
		case ast.PolicyStmt:
			a.analyzePolicy(s)
		case ast.FuncStmt:
			a.analyzeFunc(s)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Type declarations
// ─────────────────────────────────────────────────────────────────────────────
func (a *Analyzer) registerType(s ast.TypeDeclareStmt) {
	name := symbolName(s.Name)
	if name == "" {
		a.addError("type declaration is missing a name")
		return
	}
	if _, exists := a.userTypes[name]; exists {
		a.addError("type '%s' is declared more than once", name)
		return
	}
	fields := make(map[string]string)
	for fieldName, fieldTypeExpr := range s.Fields {
		fields[fieldName] = symbolName(fieldTypeExpr)
	}
	a.userTypes[name] = fields
}

func (a *Analyzer) analyzeTypeDecl(s ast.TypeDeclareStmt) {
	typeName := symbolName(s.Name)

	if len(s.Fields) == 0 {
		a.addError("type '%s' has no fields", typeName)
		return
	}

	for fieldName, fieldTypeExpr := range s.Fields {
		fieldTypeName := symbolName(fieldTypeExpr)
		if fieldTypeName == "" {
			a.addError("type '%s': field '%s' is missing a type annotation", typeName, fieldName)
			continue
		}
		a.validateTypeName(
			fieldTypeName,
			fmt.Sprintf("type '%s', field '%s'", typeName, fieldName),
		)
	}
}

func (a *Analyzer) analyzeParamType(t ast.Type, context string) {
	if t == nil {
		a.addError("%s: missing type annotation", context)
		return
	}
	switch v := t.(type) {
	case ast.SymbolType:
		a.validateTypeName(v.Name, context)
	case ast.ArrayType:
		a.analyzeParamType(v.Underlying, context)
	}
}

func (a *Analyzer) validateTypeName(typeName string, context string) {
	base := strings.TrimSuffix(typeName, "[]")

	if builtinTypes[base] {
		return
	}
	if _, exists := a.userTypes[base]; exists {
		return
	}
	a.addError(
		"%s: unknown type '%s' — must be a built-in (String, Address, UInt, Int, Bool) or declared with 'type'",
		context, typeName,
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Agent declarations
// ─────────────────────────────────────────────────────────────────────────────
func (a *Analyzer) analyzeAgent(s ast.AgentStmt) {
	name := symbolName(s.Identifier)
	if name == "" {
		a.addError("agent declaration is missing a name")
	}
	if symbolName(s.Version) == "" {
		a.addError("agent '%s': missing 'version' field", name)
	}
	if symbolName(s.Owner) == "" {
		a.addError("agent '%s': missing 'owner' field", name)
	}
	if symbolName(s.Purpose) == "" {
		a.addError("agent '%s': missing 'purpose' field", name)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Policy declarations
// ─────────────────────────────────────────────────────────────────────────────
func (a *Analyzer) analyzePolicy(s ast.PolicyStmt) {
	name := symbolName(s.Identifier)
	if name == "" {
		a.addError("policy declaration is missing a name")
	}
	if len(s.Rules) == 0 {
		a.addError("policy '%s' has no rules defined", name)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Function declarations
// ─────────────────────────────────────────────────────────────────────────────
func (a *Analyzer) analyzeFunc(s ast.FuncStmt) {
	fnName := symbolName(s.Name)

	// ── Return type ───────────────────────────────────────────────────────────
	if s.ReturnType == nil {
		a.addError("fn '%s': missing return type — did you forget ': ReturnType'?", fnName)
	} else {
		a.analyzeParamType(s.ReturnType, fmt.Sprintf("fn '%s' return type", fnName))
	}

	// ── Parameters ────────────────────────────────────────────────────────────
	localScope := make(map[string]bool)

	for _, arg := range s.Arguments {
		paramName := symbolName(arg.ArgName)
		paramType := symbolName(arg.ArgType)

		if paramName == "" {
			a.addError("fn '%s': a parameter is missing its name", fnName)
			continue
		}
		if paramType == "" {
			a.addError("fn '%s': parameter '%s' is missing a type", fnName, paramName)
			continue
		}
		a.validateTypeName(paramType,
			fmt.Sprintf("fn '%s', param '%s'", fnName, paramName))

		localScope[paramName] = true
	}

	// ── Body ──────────────────────────────────────────────────────────────────
	if body, ok := s.Body.(ast.BlockStmt); ok {
		for _, node := range body.Body {
			a.analyzeStmt(node, fnName, localScope)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Statements
// ─────────────────────────────────────────────────────────────────────────────
func (a *Analyzer) analyzeStmt(node ast.Stmt, fnName string, scope map[string]bool) {
	switch s := node.(type) {

	case ast.RequireStmt:
		a.analyzeExpr(s.Condition, fnName, scope)
		if symbolName(s.Message) == "" {
			if _, isStr := s.Message.(ast.StringExpr); !isStr {
				a.addError("fn '%s': require() is missing an error message", fnName)
			}
		}

	case ast.ReturnStmt:
		if s.Value != nil {
			a.analyzeExpr(s.Value, fnName, scope)
		}

	case ast.VarDeclStmt:
		if s.AssignedValue != nil {
			a.analyzeExpr(s.AssignedValue, fnName, scope)
		}
		scope[s.Identifier] = true

	case ast.ExpressionStmt:
		a.analyzeExpr(s.Expression, fnName, scope)

	case ast.EmitStmt:
		if symbolName(s.EventName) == "" {
			if _, isStr := s.EventName.(ast.StringExpr); !isStr {
				a.addError("fn '%s': emit() is missing an event name", fnName)
			}
		}
		if s.Arguments != nil {
			a.analyzeExpr(s.Arguments, fnName, scope)
		}

	case ast.IfStmt:
		a.analyzeExpr(s.Condition, fnName, scope)
		if s.Then != nil {
			a.analyzeStmt(s.Then, fnName, copyScope(scope))
		}
		if s.Else != nil {
			a.analyzeStmt(s.Else, fnName, copyScope(scope))
		}

	case ast.BlockStmt:
		for _, inner := range s.Body {
			a.analyzeStmt(inner, fnName, scope)
		}

	case ast.ForStmt:
		forScope := copyScope(scope)
		if s.Init != nil {
			a.analyzeStmt(s.Init, fnName, forScope)
		}
		if s.Condition != nil {
			a.analyzeExpr(s.Condition, fnName, forScope)
		}
		if s.Post != nil {
			a.analyzeStmt(s.Post, fnName, forScope)
		}
		for _, inner := range s.Body {
			a.analyzeStmt(inner, fnName, forScope)
		}

	case ast.WhileStmt:
		a.analyzeExpr(s.Condition, fnName, scope)
		for _, inner := range s.Body {
			a.analyzeStmt(inner, fnName, scope)
		}

	case ast.ArrayItemAssignmentStmt:
		a.analyzeExpr(s.Name, fnName, scope)
		a.analyzeExpr(s.Index, fnName, scope)
		a.analyzeExpr(s.Value, fnName, scope)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Expressions
// ─────────────────────────────────────────────────────────────────────────────
func (a *Analyzer) analyzeExpr(expr ast.Expr, fnName string, scope map[string]bool) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {

	case ast.SymbolExpr:
		if !scope[e.Value] &&
			!a.declaredAgents[e.Value] &&
			!a.declaredPolicies[e.Value] &&
			!a.isFunctionDeclared(e.Value) &&
			!builtinFunctions[e.Value] &&
			!builtinTypes[e.Value] {
			a.addError("fn '%s': undefined identifier '%s'", fnName, e.Value)
		}

	case ast.BinaryExpr:
		a.analyzeExpr(e.Left, fnName, scope)
		a.analyzeExpr(e.Right, fnName, scope)

	case ast.CallExpr:
		callee := symbolName(e.Calle)
		if callee != "" {
			if builtinFunctions[callee] {
				// Built-in functions (len, print, require, emit) are always valid —
				// skip arity check since they have variable signatures in the runtime.
			} else if paramCount, exists := a.declaredFunctions[callee]; !exists {
				a.addError("fn '%s': call to undefined function '%s'", fnName, callee)
			} else if len(e.Arguments) != paramCount {
				a.addError("fn '%s': '%s' expects %d argument(s), got %d",
					fnName, callee, paramCount, len(e.Arguments))
			}
		}
		for _, arg := range e.Arguments {
			a.analyzeExpr(arg, fnName, scope)
		}

	case ast.MemberExpr:
		a.analyzeExpr(e.Object, fnName, scope)

	case ast.AssignmentExpr:
		a.analyzeExpr(e.Right, fnName, scope)
		if sym, ok := e.Left.(ast.SymbolExpr); ok {
			scope[sym.Value] = true
		} else {
			a.analyzeExpr(e.Left, fnName, scope)
		}

	case ast.ArrayLiteralExpr:
		for _, item := range e.Items {
			a.analyzeExpr(item, fnName, scope)
		}

	case ast.ArrayAccessItemExpr:
		a.analyzeExpr(e.Array, fnName, scope)
		a.analyzeExpr(e.Index, fnName, scope)

	case ast.ObjectAssignmentExpr:
		for _, prop := range e.Fields {
			a.analyzeExpr(prop.Value, fnName, scope)
		}

	case ast.PrefixExpr:
		a.analyzeExpr(e.RightExpr, fnName, scope)

	case ast.IncDecExpr:
		a.analyzeExpr(e.Left, fnName, scope)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────
func symbolName(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case ast.SymbolExpr:
		return e.Value
	case ast.StringExpr:
		return e.Value
	case ast.NumberExpr:
		return fmt.Sprintf("%v", e.Value)
	case ast.ExpressionStmt:
		return symbolName(e.Expression)
	}
	return ""
}

func typeToString(t ast.Type) string {
	if t == nil {
		return ""
	}
	switch v := t.(type) {
	case ast.SymbolType:
		return v.Name
	case ast.ArrayType:
		return typeToString(v.Underlying)
	}
	return ""
}

func copyScope(scope map[string]bool) map[string]bool {
	child := make(map[string]bool, len(scope))
	for k, v := range scope {
		child[k] = v
	}
	return child
}

func (a *Analyzer) isFunctionDeclared(name string) bool {
	if builtinFunctions[name] {
		return true
	}
	_, ok := a.declaredFunctions[name]
	return ok
}
