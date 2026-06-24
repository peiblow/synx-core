package ast

type BlockStmt struct {
	Body []Stmt
}

func (n BlockStmt) stmt() {}

type ImportStmt struct {
	Identifier string
	Path       string
}

func (n ImportStmt) stmt() {}

type LibraryStmt struct {
	Identifier string
	Body       []Stmt
}

func (n LibraryStmt) stmt() {}

type ContractStmt struct {
	Identifier string
	Body       []Stmt
}

func (n ContractStmt) stmt() {}

type ExpressionStmt struct {
	Expression Expr
}

func (n ExpressionStmt) stmt() {}

type VarDeclStmt struct {
	Identifier    string
	Constant      bool
	AssignedValue Expr
	ExplicityType Type
}

func (n VarDeclStmt) stmt() {}

type IfStmt struct {
	Condition Expr
	Then      Stmt
	Else      Stmt
}

func (n IfStmt) stmt() {}

type WhileStmt struct {
	Condition Expr
	Body      []Stmt
}

func (n WhileStmt) stmt() {}

type ForStmt struct {
	Init      Stmt
	Condition Expr
	Post      Stmt
	Body      []Stmt
}

func (n ForStmt) stmt() {}

type FuncStmt struct {
	Name       Expr
	Arguments  []ArgsStmt
	Body       Stmt
	ReturnType Type
}

type ArgsStmt struct {
	ArgName Expr
	ArgType Expr
}

func (n FuncStmt) stmt() {}

type ArrayItemAssignmentStmt struct {
	Name  Expr
	Index Expr
	Value Expr
}

func (n ArrayItemAssignmentStmt) stmt() {}

type ReturnStmt struct {
	Value Expr
}

func (n ReturnStmt) stmt() {}

type RequireStmt struct {
	Condition Expr
	Message   Expr
}

func (n RequireStmt) stmt() {}

type AgentStmt struct {
	Identifier Expr
	Hash       Expr
	Version    Expr
	Owner      Expr
	Purpose    Expr
	Tools      []ToolStmt
}

func (n AgentStmt) stmt() {}

type PolicyStmt struct {
	Identifier Expr
	Rules      map[string]Expr
}

func (n PolicyStmt) stmt() {}

type TypeDeclareStmt struct {
	Name   Expr
	Fields map[string]Expr
}

func (n TypeDeclareStmt) stmt() {}

type EmitStmt struct {
	EventName Expr
	Arguments Expr
}

func (n EmitStmt) stmt() {}

type GetEnvStmt struct {
	VariableName Expr
}

func (n GetEnvStmt) stmt() {}

type TryCatchStmt struct {
	TryBlock   []Stmt
	CatchVar   string
	CatchBlock []Stmt
}

func (n TryCatchStmt) stmt() {}

type ToolStmt struct {
	Name        Expr
	Description Expr
	Steps       []ToolStep
}

type ToolStep struct {
	Function string
	Action   ToolAction
}

type ToolAction struct {
	Method string
	Url    Expr
}

func (n ToolStmt) stmt() {}
