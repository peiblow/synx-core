package compiler

// Opcodes da VM
const (
	// Fim do programa
	OP_HALT = 0x00

	OP_TRUE  = 0xFE // valor booleano true
	OP_FALSE = 0xFF // valor booleano false

	// Operações de pilha
	OP_CONST = 0x01 // carrega constante do pool
	OP_PUSH  = 0x02 // empilha valor imediato
	OP_POP   = 0x0E // remove topo da stack
	OP_DUP   = 0x0F // duplica topo da stack
	OP_SWAP  = 0x10 // troca dois valores do topo

	// Operações aritméticas
	OP_ADD = 0x03 // soma valores do topo da stack
	OP_SUB = 0x04 // subtração
	OP_MUL = 0x05 // multiplicação
	OP_DIV = 0x06 // divisão

	// Operações de comparação
	OP_GT      = 0x07 // maior que
	OP_GT_EQ   = 0x08 // maior ou igual
	OP_LT      = 0x09 // menor que
	OP_LT_EQ   = 0x0A // menor ou igual
	OP_EQ      = 0x0B // igual
	OP_DIFF    = 0x0C // diferente
	OP_PLUS_EQ = 0x0D // incremento e atribuição

	// Operações lógicas
	OP_AND = 0x63 // AND lógico (consome dois booleanos, empilha um)
	OP_OR  = 0x64 // OR lógico (consome dois booleanos, empilha um)
	OP_NOT = 0x65 // NOT lógico (consome um booleano, empilha um)

	// I/O
	OP_PRINT   = 0x11 // imprime valor do topo
	OP_GET_ENV = 0x20 // obtém variável de ambiente
	OP_HASH     = 0x21 // calcula hash de valor do topo
	OP_NONCE    = 0x22 // gera nonce aleatório
	OP_CONTAINS = 0x23 // verifica se string contém substring
	OP_REGEX    = 0x24 // verifica se string casa com padrão regex
	OP_NOP      = 0x12 // instrução nula

	// Controle de fluxo
	OP_JMP    = 0x13 // salto incondicional
	OP_JMP_IF = 0x14 // salto condicional (se falso)
	OP_CALL   = 0x15 // chamada de função
	OP_RET    = 0x16 // retorno de função

	// Operações de array
	OP_ACCESS = 0x17 // acessa item específico do array
	OP_LENGTH = 0x19 // retorna tamanho de um array

	// Valores especiais
	OP_NULL = 0x18 // valor nulo

	// Storage (runtime)
	OP_STORE  = 0x1A // armazena valor na storage
	OP_SLOAD  = 0x1B // carrega valor da storage
	OP_DELETE = 0x1E // remove valor da storage

	// Registry (persistente)
	OP_AGENT_DECLARE  = 0x1C // armazena valor na memória
	OP_AGENT_GET      = 0x1D // carrega valor da memória
	OP_AGENT_VALIDATE = 0x1F // validates agent against registry

	OP_POLICY_DECLARE = 0x4F // declara política
	OP_TYPE_DECLARE   = 0x5F // declara tipo

	// Blockchain/Smart Contract
	OP_EMIT    = 0x51 // emite evento
	OP_REQUIRE = 0x52 // verifica condição (reverte se falso)
	OP_ERR     = 0x53 // lança erro/exceção
	OP_TRY     = 0x54 // inicia bloco try
	OP_END_TRY = 0x55 // finaliza bloco try-catch

	// Objetos
	OP_PUSH_OBJECT  = 0x60 // cria objeto vazio na pilha
	OP_SET_PROPERTY = 0x61 // define propriedade de objeto
	OP_GET_PROPERTY = 0x62 // obtém propriedade de objeto
)

// OpcodeNames mapeia opcodes para seus nomes (útil para debug)
var OpcodeNames = map[byte]string{
	OP_HALT:          "HALT",
	OP_CONST:         "CONST",
	OP_PUSH:          "PUSH",
	OP_POP:           "POP",
	OP_DUP:           "DUP",
	OP_SWAP:          "SWAP",
	OP_ADD:           "ADD",
	OP_SUB:           "SUB",
	OP_MUL:           "MUL",
	OP_DIV:           "DIV",
	OP_GT:            "GT",
	OP_GT_EQ:         "GT_EQ",
	OP_LT:            "LT",
	OP_LT_EQ:         "LT_EQ",
	OP_EQ:            "EQ",
	OP_DIFF:          "DIFF",
	OP_PLUS_EQ:       "PLUS_EQ",
	OP_AND:           "AND",
	OP_OR:            "OR",
	OP_NOT:           "NOT",
	OP_PRINT:         "PRINT",
	OP_NOP:           "NOP",
	OP_JMP:           "JMP",
	OP_JMP_IF:        "JMP_IF",
	OP_CALL:          "CALL",
	OP_RET:           "RET",
	OP_ACCESS:        "ACCESS",
	OP_LENGTH:        "LENGTH",
	OP_NULL:          "NULL",
	OP_STORE:         "STORE",
	OP_SLOAD:         "SLOAD",
	OP_DELETE:        "DELETE",
	OP_EMIT:          "EMIT",
	OP_GET_ENV:       "GET_ENV",
	OP_HASH:          "HASH",
	OP_NONCE:         "NONCE",
	OP_CONTAINS:      "CONTAINS",
	OP_REGEX:         "REGEX",
	OP_REQUIRE:       "REQUIRE",
	OP_AGENT_DECLARE: "REGISTRY_DECLARE",
	OP_AGENT_GET:     "REGISTRY_GET",
	OP_ERR:           "ERR",
	OP_TRY:           "TRY",
	OP_END_TRY:       "END_TRY",
	OP_PUSH_OBJECT:   "PUSH_OBJECT",
	OP_SET_PROPERTY:  "SET_PROPERTY",
	OP_GET_PROPERTY:  "GET_PROPERTY",
}

// HasOperand retorna true se o opcode requer um operando
func HasOperand(op byte) bool {
	switch op {
	case OP_PUSH, OP_CONST, OP_STORE, OP_SLOAD,
		OP_CALL, OP_JMP, OP_JMP_IF:
		return true
	}
	return false
}
