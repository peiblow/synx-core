package compiler

import (
	"fmt"
	"reflect"

	"github.com/peiblow/vvm/ast"
)

type ArgMeta struct {
	Name     string `json:"name"`
	Slot     int    `json:"slot"`
	TypeName string `json:"type_name"`
}

type FunctionMeta struct {
	Addr    int       `json:"addr"`
	Args    []int     `json:"args"`
	ArgMeta []ArgMeta `json:"arg_meta"`
}

type TypeMeta struct {
	Fields map[string]string `json:"fields"`
}

type ToolAction struct {
	Type      string   `json:"type"`
	Method    string   `json:"method,omitempty"`
	Url       string   `json:"url,omitempty"`
	Operation string   `json:"operation,omitempty"`
	Path      string   `json:"path,omitempty"`
	Command   string   `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Agent     string            `json:"agent,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
}

type ToolStepInput struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ToolStep struct {
	Function string          `json:"function"`
	Input    []ToolStepInput `json:"input,omitempty"`
	Action   *ToolAction     `json:"action,omitempty"`
}

type ToolStmt struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Steps       []ToolStep `json:"steps"`
}

type ModelStmt struct {
	Provider    string  `json:"provider"`
	Name        string  `json:"name"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

type BehaviorStmt struct {
	SystemPrompt string `json:"system_prompt"`
	MaxSteps     int    `json:"max_steps"`
	OnDeny       string `json:"on_deny"`
	OnError      string `json:"on_error"`
}

type SkillStmt struct {
	Name    string   `json:"name"`
	Content string   `json:"content"`
	Uses    []string `json:"uses"`
}

type TriggerStmt struct {
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

type AgentInfo struct {
	Hash     string `json:"hash"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Purpose  string `json:"purpose"`
	Tools    []ToolStmt
	Model    ModelStmt
	Behavior BehaviorStmt
	Skills   []SkillStmt   `json:"skills"`
	Triggers []TriggerStmt `json:"triggers"`
}

type Compiler struct {
	BaseDir      string
	Code         []byte
	Symbols      map[string]int
	ConstPool    []interface{}
	Functions    map[string]FunctionMeta
	FunctionName map[int]string
	Types        map[string]TypeMeta
	AgentInfo    AgentInfo
	NextSlot     int
	isInFunction bool
}

const GlobalScopeSlot = 0

func New(baseDir string) *Compiler {
	return &Compiler{
		BaseDir:      baseDir,
		Code:         []byte{},
		Symbols:      map[string]int{"this": GlobalScopeSlot},
		ConstPool:    make([]interface{}, 0),
		Functions:    make(map[string]FunctionMeta),
		FunctionName: make(map[int]string),
		Types:        make(map[string]TypeMeta),
		NextSlot:     GlobalScopeSlot + 1,
	}
}

type ContractArtifact struct {
	Bytecode     []byte                  `json:"bytecode"`
	ConstPool    []interface{}           `json:"const_pool"`
	Functions    map[string]FunctionMeta `json:"functions"`
	FunctionName map[int]string          `json:"function_name"`
	Types        map[string]TypeMeta     `json:"types"`
	InitStorage  map[int]interface{}     `json:"init_storage"`
	AgentInfo    AgentInfo               `json:"agent_info"`
}

func (c *Compiler) Artifact() *ContractArtifact {
	return &ContractArtifact{
		Bytecode:     c.Code,
		ConstPool:    c.ConstPool,
		Functions:    c.Functions,
		FunctionName: c.FunctionName,
		Types:        c.Types,
		InitStorage:  make(map[int]interface{}),
		AgentInfo:    c.AgentInfo,
	}
}

func (c *Compiler) GetExpectedArgType(funcName string, argIndex int) string {
	if meta, ok := c.Functions[funcName]; ok {
		if argIndex < len(meta.ArgMeta) {
			return meta.ArgMeta[argIndex].TypeName
		}
	}
	return ""
}

func (c *Compiler) GetActualType(expr ast.Expr) string {
	switch e := expr.(type) {
	case ast.NumberExpr:
		if e.Value == float64(int(e.Value)) {
			return "Int"
		}
		return "Float"
	case ast.StringExpr:
		return "String"
	case ast.SymbolExpr:
		if _, ok := c.Types[e.Value]; ok {
			return e.Value
		}
		return "Unknown"
	case ast.ObjectAssignmentExpr:
		if sym, ok := e.Name.(ast.SymbolExpr); ok {
			return sym.Value
		}
		return "Object"
	default:
		return "Unknown"
	}
}

func (c *Compiler) TypesAreCompatible(expectedType, actualType string) bool {
	if expectedType == actualType {
		return true
	}

	if actualType == "Unknown" {
		return true
	}

	if actualType == "Object" {
		if _, isCustomType := c.Types[expectedType]; isCustomType {
			return true
		}
	}

	primitiveAliases := map[string][]string{
		"Int":     {"UInt", "Float", "Number"},
		"UInt":    {"Int", "Float", "Number"},
		"Float":   {"Int", "UInt", "Number"},
		"Number":  {"Int", "UInt", "Float"},
		"String":  {},
		"Address": {"String", "Int", "UInt"},
		"Proof":   {"String"},
	}

	if aliases, ok := primitiveAliases[expectedType]; ok {
		for _, alias := range aliases {
			if alias == actualType {
				return true
			}
		}
	}

	if _, isCustomType := c.Types[expectedType]; isCustomType {
		switch actualType {
		case "Int", "UInt", "Float", "Number", "String":
			return false
		}
	}

	return false
}

func (c *Compiler) ValidateObjectAgainstType(obj ast.ObjectAssignmentExpr, typeName string) error {
	typeMeta, exists := c.Types[typeName]
	if !exists {
		return fmt.Errorf("unknown type '%s'", typeName)
	}

	providedFields := make(map[string]ast.Expr)
	for _, field := range obj.Fields {
		if key, ok := field.Key.(ast.SymbolExpr); ok {
			providedFields[key.Value] = field.Value
		}
	}

	for fieldName, expectedFieldType := range typeMeta.Fields {
		providedValue, exists := providedFields[fieldName]
		if !exists {
			return fmt.Errorf("missing field '%s' of type '%s' in object literal for type '%s'",
				fieldName, expectedFieldType, typeName)
		}

		actualFieldType := c.GetActualType(providedValue)
		if !c.TypesAreCompatible(expectedFieldType, actualFieldType) {
			return fmt.Errorf("field '%s' has type '%s', expected '%s' for type '%s'",
				fieldName, actualFieldType, expectedFieldType, typeName)
		}
	}

	return nil
}

func (c *Compiler) ValidateFunctionCall(funcName string, args []ast.Expr) error {
	funcMeta, exists := c.Functions[funcName]
	if !exists {
		return nil
	}

	if len(args) != len(funcMeta.ArgMeta) {
		return fmt.Errorf("function '%s' expects %d argument(s), got %d",
			funcName, len(funcMeta.ArgMeta), len(args))
	}

	for i, arg := range args {
		expectedType := funcMeta.ArgMeta[i].TypeName
		actualType := c.GetActualType(arg)

		if actualType == "Object" {
			if _, isCustomType := c.Types[expectedType]; isCustomType {
				if objExpr, ok := arg.(ast.ObjectAssignmentExpr); ok {
					if err := c.ValidateObjectAgainstType(objExpr, expectedType); err != nil {
						return err
					}
					continue
				}
			}
		}

		if !c.TypesAreCompatible(expectedType, actualType) {
			return fmt.Errorf("type mismatch in argument %d of function '%s': expected '%s', got '%s'",
				i+1, funcName, expectedType, actualType)
		}
	}

	return nil
}

func (c *Compiler) emit(opcodes ...byte) {
	c.Code = append(c.Code, opcodes...)
}

func (c *Compiler) addConst(val interface{}) byte {
	if isComparableConst(val) {
		for i, v := range c.ConstPool {
			if isComparableConst(v) && v == val {
				return byte(i)
			}
		}
	}
	idx := len(c.ConstPool)
	c.ConstPool = append(c.ConstPool, val)
	if idx > 255 {
		panic(fmt.Sprintf("ConstPool overflow: %d entries — OP_CONST operand is 1 byte (max 256). Reduce contract size or implement OP_LCONST.", idx+1))
	}
	return byte(idx)
}

// Slices and maps panic when compared with ==; only dedupe values whose dynamic
// type is comparable (strings, ints, floats, comparable structs like ast.SymbolExpr).
func isComparableConst(v interface{}) bool {
	if v == nil {
		return true
	}
	return reflect.TypeOf(v).Comparable()
}

func (c *Compiler) findConst(val interface{}) byte {
	for i, v := range c.ConstPool {
		if v == val {
			return byte(i)
		}
	}
	return byte(255)
}

func (c *Compiler) allocSlot(name string) int {
	slot := c.NextSlot
	c.Symbols[name] = slot
	c.NextSlot++
	return slot
}

func (c *Compiler) getSlot(name string) int {
	slot, ok := c.Symbols[name]
	if !ok {
		slot = c.allocSlot(name)
	}
	return slot
}

func (c *Compiler) GetFuncArgs(addr int) []int {
	funcName := c.FunctionName[addr]
	return c.Functions[funcName].Args
}

func (c *Compiler) CompileProgram(modules []ast.BlockStmt, main ast.BlockStmt) {
	for _, m := range modules {
		for _, stmt := range m.Body {
			c.compileStmt(stmt)
		}
	}
	c.CompileBlock(main)
}

func (c *Compiler) CompileBlock(block ast.BlockStmt) {
	for _, stmt := range block.Body {
		c.compileStmt(stmt)
	}
	c.emit(OP_HALT)
}

func (c *Compiler) compileBlock(block ast.BlockStmt) {
	for _, stmt := range block.Body {
		c.compileStmt(stmt)
	}
}

func (c *Compiler) currentPos() int {
	return len(c.Code)
}

func (c *Compiler) patchJump(pos int, target int) {
	c.Code[pos] = byte(target >> 8)     // high byte
	c.Code[pos+1] = byte(target & 0xFF) // low byte
}
