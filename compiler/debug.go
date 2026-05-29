package compiler

import "fmt"

// Disassemble retorna uma representação legível do bytecode
func (c *Compiler) Disassemble() string {
	var output string
	for i := 0; i < len(c.Code); i++ {
		op := c.Code[i]
		name, ok := OpcodeNames[op]
		if !ok {
			name = fmt.Sprintf("UNKNOWN_%02X", op)
		}

		if HasOperand(op) && i+1 < len(c.Code) {
			i++
			output += fmt.Sprintf("%04d: %s %d\n", i-1, name, c.Code[i])
		} else {
			output += fmt.Sprintf("%04d: %s\n", i, name)
		}
	}
	return output
}

// PrintBytecode imprime o bytecode de forma legível
func (c *Compiler) PrintBytecode() {
	fmt.Print(c.Disassemble())
}

// PrintSymbols imprime a tabela de símbolos
func (c *Compiler) PrintSymbols() {
	fmt.Println("=== Symbol Table ===")
	for name, slot := range c.Symbols {
		fmt.Printf("  %s -> slot %d\n", name, slot)
	}
}

// PrintFunctions imprime as funções registradas
func (c *Compiler) PrintFunctions() {
	fmt.Println("=== Functions ===")
	for name, meta := range c.Functions {
		fmt.Printf("  %s @ %d (args: %v)\n", name, meta.Addr, meta.Args)
	}
}

// PrintConstPool imprime o pool de constantes
func (c *Compiler) PrintConstPool() {
	fmt.Println("=== Const Pool ===")
	for i, val := range c.ConstPool {
		fmt.Printf("  [%d] %v (%T)\n", i, val, val)
	}
}

// Debug imprime todas as informações de debug
func (c *Compiler) Debug() {
	fmt.Println("\n========== COMPILER DEBUG ==========")
	c.PrintSymbols()
	fmt.Println()
	c.PrintFunctions()
	fmt.Println()
	c.PrintConstPool()
	fmt.Println()
	fmt.Println("=== Bytecode ===")
	c.PrintBytecode()
	fmt.Println("=====================================")
}
