package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/matheus-sl2/nortabil/internal/middleware"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// PeriodHandler processa requisições de controle de períodos contábeis
type PeriodHandler struct {
	periodService *service.PeriodService
}

// NewPeriodHandler cria uma nova instância do handler de períodos
func NewPeriodHandler(periodService *service.PeriodService) *PeriodHandler {
	return &PeriodHandler{periodService: periodService}
}

// List lista os 12 períodos contábeis do ano
func (h *PeriodHandler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	year := time.Now().Year()
	if yParam := r.URL.Query().Get("year"); yParam != "" {
		if yInt, err := strconv.Atoi(yParam); err == nil && yInt > 0 {
			year = yInt
		}
	}

	periods, err := h.periodService.ListPeriods(r.Context(), companyID, year)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, periods)
}

type periodActionInput struct {
	Year     int    `json:"year"`
	Month    int    `json:"month"`
	ClosedBy string `json:"closed_by,omitempty"`
}

// Close fecha um período mensal
func (h *PeriodHandler) Close(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	var input periodActionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Corpo da requisição JSON inválido", err.Error())
		return
	}

	if input.ClosedBy == "" {
		input.ClosedBy = middleware.GetUserIdentity(r)
	}

	err := h.periodService.ClosePeriod(r.Context(), companyID, input.Year, input.Month, input.ClosedBy)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{
		"message": "Período contábil fechado com sucesso",
	})
}

// Reopen reabre um período mensal fechado
func (h *PeriodHandler) Reopen(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	var input periodActionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Corpo da requisição JSON inválido", err.Error())
		return
	}

	err := h.periodService.ReopenPeriod(r.Context(), companyID, input.Year, input.Month)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{
		"message": "Período contábil reaberto com sucesso",
	})
}

// CloseYear executa a Apuração do Resultado do Exercício (ARE)
func (h *PeriodHandler) CloseYear(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	var input service.YearClosingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Corpo da requisição JSON inválido", err.Error())
		return
	}

	summary, err := h.periodService.ExecuteYearClosing(r.Context(), companyID, input)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, summary)
}

