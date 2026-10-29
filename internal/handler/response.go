package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/matheus-sl2/nortabil/internal/domain"
)

// StandardResponse envelope padrão para respostas JSON da API
type StandardResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorItem  `json:"error,omitempty"`
}

// ErrorItem detalhes do erro retornado
type ErrorItem struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// JSON envia resposta de sucesso formatada
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(StandardResponse{
		Success: true,
		Data:    data,
	})
}

// JSONError envia resposta de erro padronizada
func JSONError(w http.ResponseWriter, statusCode int, errorCode, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(StandardResponse{
		Success: false,
		Error: &ErrorItem{
			Code:    errorCode,
			Message: message,
			Details: details,
		},
	})
}

// HandleDomainError mapeia erros de domínio contábil para o status HTTP e mensagem adequados
func HandleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnbalancedJournalEntry), errors.Is(err, domain.ErrMismatchedCompanyTenant):
		JSONError(w, http.StatusUnprocessableEntity, "UNBALANCED_ENTRY", err.Error(), nil)
	case errors.Is(err, domain.ErrSyntheticAccountPostNotAllowed):
		JSONError(w, http.StatusUnprocessableEntity, "SYNTHETIC_ACCOUNT_NOT_ALLOWED", err.Error(), nil)
	case errors.Is(err, domain.ErrAccountNotFound), errors.Is(err, domain.ErrParentAccountNotFound), errors.Is(err, domain.ErrCompanyNotFound), errors.Is(err, domain.ErrPeriodNotFound), errors.Is(err, domain.ErrResultAccountNotFound), errors.Is(err, domain.ErrJournalEntryNotFound):
		JSONError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, domain.ErrAccountAlreadyExists), errors.Is(err, domain.ErrCompanyAlreadyExists), errors.Is(err, domain.ErrJournalEntryAlreadyReversed):
		JSONError(w, http.StatusConflict, "ALREADY_EXISTS", err.Error(), nil)
	case errors.Is(err, domain.ErrPeriodClosed), errors.Is(err, domain.ErrPeriodLocked):
		JSONError(w, http.StatusForbidden, "PERIOD_CLOSED", err.Error(), nil)
	case errors.Is(err, domain.ErrPeriodAlreadyOpen):
		JSONError(w, http.StatusBadRequest, "PERIOD_ALREADY_OPEN", err.Error(), nil)
	case errors.Is(err, domain.ErrEmptyJournalLines),
		errors.Is(err, domain.ErrZeroOrNegativeAmount),
		errors.Is(err, domain.ErrInvalidAccountType),
		errors.Is(err, domain.ErrInvalidLineType),
		errors.Is(err, domain.ErrInvalidDate),
		errors.Is(err, domain.ErrInvalidAccountCode),
		errors.Is(err, domain.ErrInvalidDateRange),
		errors.Is(err, domain.ErrInactiveAccount),
		errors.Is(err, domain.ErrInvalidCNPJ),
		errors.Is(err, domain.ErrInvalidTaxRegime),
		errors.Is(err, domain.ErrParentAccountMustBeSynthetic),
		errors.Is(err, domain.ErrAccountHasEntriesCannotBeSynthetic):
		JSONError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
	default:
		JSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Ocorreu um erro interno no servidor contábil: "+err.Error(), nil)
	}
}

