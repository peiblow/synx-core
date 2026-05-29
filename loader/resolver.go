package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/peiblow/vvm/ast"
	"github.com/peiblow/vvm/lexer"
	"github.com/peiblow/vvm/parser"
)

type Program struct {
	Main    ast.BlockStmt
	Modules []*Module
}

type Module struct {
	Key string
	AST ast.BlockStmt
}

type Loader struct {
	Program *Program
}

func NewLoader() *Loader {
	return &Loader{
		Program: &Program{},
	}
}

func (l *Loader) Resolver(mainSrc []lexer.Token, baseDir string) (*Program, error) {
	mainAst := parser.Parse(mainSrc)

	prog := &Program{Main: mainAst}

	for _, stmt := range mainAst.Body {
		if module, ok := stmt.(ast.ImportStmt); ok {
			if err := l.resolveModule(prog, baseDir, module.Identifier, module.Path); err != nil {
				return nil, err
			}
		}
	}

	return prog, nil
}

func (l *Loader) resolveModule(prog *Program, fromDir, moduleAlias, modulePath string) error {
	content, err := l.load(fromDir, modulePath)
	if err != nil {
		return err
	}

	lexResult := lexer.Tokenize(content)
	if lexResult.HasErrors() {
		errMsg := fmt.Sprintf("lexical errors in module '%s':\n", modulePath)
		for _, e := range lexResult.Errors {
			errMsg += "  " + e.Error() + "\n"
		}
		return fmt.Errorf("%s", errMsg)
	}

	moduleAst := parser.Parse(lexResult.Tokens)

	prog.Modules = append(prog.Modules, &Module{
		Key: moduleAlias,
		AST: ast.BlockStmt{Body: moduleAst.Body},
	})

	for _, stmt := range moduleAst.Body {
		if nestedModule, ok := stmt.(ast.ImportStmt); ok {
			nestedFromDir := filepath.Dir(filepath.Join(fromDir, modulePath))
			if err := l.resolveModule(prog, nestedFromDir, nestedModule.Identifier, nestedModule.Path); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *Loader) load(fromDir, modulePath string) (string, error) {
	fullPath := filepath.Join(fromDir, modulePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to load module '%s': %w", modulePath, err)
	}

	return string(content), nil
}
