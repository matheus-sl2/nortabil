package domain

import "errors"

// Erros de domínio contábil e validações de regras de negócio
var (
	// ErrUnbalancedJournalEntry ocorre quando a soma dos débitos não é igual à soma dos créditos (Lei de Luca Pacioli).
	ErrUnbalancedJournalEntry = errors.New("lançamento contábil desbalanceado: a soma dos débitos deve ser estritamente igual à soma dos créditos")

	// ErrEmptyJournalLines ocorre quando um lançamento contábil não possui partidas (linhas).
	ErrEmptyJournalLines = errors.New("o lançamento contábil deve conter ao menos duas partidas (débito e crédito)")

	// ErrZeroOrNegativeAmount ocorre quando uma partida possui valor menor ou igual a zero.
	ErrZeroOrNegativeAmount = errors.New("o valor de cada partida deve ser estritamente positivo maior que zero")

	// ErrAccountNotFound ocorre quando a conta contábil especificada não existe.
	ErrAccountNotFound = errors.New("conta contábil não encontrada")

	// ErrAccountAlreadyExists ocorre quando tenta-se criar uma conta com um código já existente.
	ErrAccountAlreadyExists = errors.New("já existe uma conta contábil cadastrada com este código")

	// ErrSyntheticAccountPostNotAllowed ocorre quando se tenta efetuar lançamento em conta sintética (agrupadora).
	// Conforme a Resolução CFC ITG 2000 e Lei 6.404/76, lançamentos só são permitidos em contas analíticas.
	ErrSyntheticAccountPostNotAllowed = errors.New("lançamento não permitido em conta sintética: lançamentos contábeis devem ser realizados exclusivamente em contas analíticas")

	// ErrInactiveAccount ocorre quando uma conta está inativa para novos lançamentos.
	ErrInactiveAccount = errors.New("a conta contábil informada está inativa")

	// ErrInvalidAccountType ocorre quando o tipo da conta informado é inválido.
	ErrInvalidAccountType = errors.New("tipo de conta contábil inválido (permitidos: ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE)")

	// ErrInvalidLineType ocorre quando o tipo da partida não é débito nem crédito.
	ErrInvalidLineType = errors.New("tipo de partida inválido (permitidos: DEBIT, CREDIT)")

	// ErrInvalidDate ocorre quando a data de competência é inválida ou vazia.
	ErrInvalidDate = errors.New("data de competência contábil inválida ou não informada")

	// ErrInvalidAccountCode ocorre quando o formato do código contábil é inválido.
	ErrInvalidAccountCode = errors.New("código contábil inválido: deve seguir o padrão hierárquico (ex: 1, 1.1, 1.1.01, 1.1.01.001)")

	// ErrParentAccountNotFound ocorre quando a conta pai informada na hierarquia não foi encontrada.
	ErrParentAccountNotFound = errors.New("conta contábil pai não encontrada na hierarquia do plano de contas")

	// ErrParentAccountMustBeSynthetic ocorre quando tenta-se criar uma conta filha sob uma conta que não pode ser sintética.
	ErrParentAccountMustBeSynthetic = errors.New("uma conta filha só pode ser criada sob uma conta sintética")

	// ErrAccountHasEntriesCannotBeSynthetic ocorre quando tenta-se tornar sintética uma conta que já possui lançamentos diretos.
	ErrAccountHasEntriesCannotBeSynthetic = errors.New("a conta já possui lançamentos contábeis registrados e não pode se tornar sintética")

	// ErrInvalidDateRange ocorre quando o intervalo de datas de um relatório contábil é inconsistente.
	ErrInvalidDateRange = errors.New("intervalo de datas inválido: a data inicial não pode ser superior à data final")

	// ErrCompanyNotFound ocorre quando a empresa solicitada não foi localizada.
	ErrCompanyNotFound = errors.New("empresa cliente não encontrada")

	// ErrCompanyAlreadyExists ocorre quando já existe uma empresa com o CNPJ informado.
	ErrCompanyAlreadyExists = errors.New("já existe uma empresa cadastrada com este CNPJ")

	// ErrInvalidCNPJ ocorre quando o CNPJ informado possui formato ou dígitos verificadores inválidos.
	ErrInvalidCNPJ = errors.New("CNPJ inválido: falha na validação de dígitos verificadores da Receita Federal")

	// ErrInvalidTaxRegime ocorre quando o regime tributário informado não é suportado.
	ErrInvalidTaxRegime = errors.New("regime tributário inválido (permitidos: SIMPLES_NACIONAL, LUCRO_PRESUMIDO, LUCRO_REAL, MEI)")

	// ErrMismatchedCompanyTenant ocorre quando há tentativa de misturar contas ou lançamentos de empresas diferentes.
	ErrMismatchedCompanyTenant = errors.New("violação de isolamento multi-tenant: contas pertencem a empresas distintas")

	// ErrPeriodClosed ocorre quando tenta-se lançar em período mensal já encerrado.
	ErrPeriodClosed = errors.New("período contábil fechado: não são permitidos lançamentos retroativos em períodos já encerrados")

	// ErrPeriodLocked ocorre quando tenta-se reabrir um período já auditado e travado.
	ErrPeriodLocked = errors.New("período contábil travado (LOCKED): períodos auditados não podem ser alterados")

	// ErrPeriodNotFound ocorre quando o período solicitado não existe.
	ErrPeriodNotFound = errors.New("período contábil não localizado")

	// ErrPeriodAlreadyOpen ocorre quando tenta-se abrir um período que já está aberto.
	ErrPeriodAlreadyOpen = errors.New("o período contábil informado já está aberto")

	// ErrResultAccountNotFound ocorre quando a conta de Lucros ou Prejuízos Acumulados não foi localizada no ARE.
	ErrResultAccountNotFound = errors.New("conta de destinação do resultado (Lucros/Prejuízos Acumulados) não encontrada")

	// ErrJournalEntryNotFound ocorre quando o lançamento contábil solicitado não existe.
	ErrJournalEntryNotFound = errors.New("lançamento contábil não encontrado")

	// ErrJournalEntryAlreadyReversed ocorre quando tenta-se estornar um lançamento que já foi estornado.
	ErrJournalEntryAlreadyReversed = errors.New("este lançamento contábil já foi estornado anteriormente")
)

