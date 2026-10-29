package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// ReportHandler processa requisições de relatórios e livros contábeis
type ReportHandler struct {
	service *service.ReportService
}

// NewReportHandler cria uma nova instância do handler de relatórios
func NewReportHandler(service *service.ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

// GetLedger gera o Livro Razão
// @Summary Gerar Livro Razão
// @Description Emite o Livro Razão contábil detalhado por conta no período especificado
// @Tags Relatórios
// @Produce json
// @Param start_date query string false "Data inicial (YYYY-MM-DD, padrão: 30 dias atrás)"
// @Param end_date query string false "Data final (YYYY-MM-DD, padrão: hoje)"
// @Param account_id query string false "Filtrar por UUID de uma conta analítica específica"
// @Success 200 {object} domain.LedgerReport
// @Router /api/v1/reports/ledger [get]
func (h *ReportHandler) GetLedger(w http.ResponseWriter, r *http.Request) {
	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", err.Error(), nil)
		return
	}

	var accountID *uuid.UUID
	if accIDParam := r.URL.Query().Get("account_id"); accIDParam != "" {
		id, err := uuid.Parse(accIDParam)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "INVALID_UUID", "Parâmetro account_id inválido", nil)
			return
		}
		accountID = &id
	}

	report, err := h.service.GetLedger(r.Context(), startDate, endDate, accountID)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, report)
}

// GetDailyBook gera o Livro Diário Geral
// @Summary Gerar Livro Diário Geral
// @Description Emite o Livro Diário cronológico com todas as operações registradas no período
// @Tags Relatórios
// @Produce json
// @Param start_date query string false "Data inicial (YYYY-MM-DD)"
// @Param end_date query string false "Data final (YYYY-MM-DD)"
// @Success 200 {object} domain.DailyBookReport
// @Router /api/v1/reports/journal [get]
func (h *ReportHandler) GetDailyBook(w http.ResponseWriter, r *http.Request) {
	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", err.Error(), nil)
		return
	}

	report, err := h.service.GetDailyBook(r.Context(), startDate, endDate)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, report)
}

// GetTrialBalance gera o Balancete de Verificação
// @Summary Gerar Balancete de Verificação
// @Description Emite o Balancete de Verificação contendo saldos anteriores, movimentações e saldos finais de todas as contas
// @Tags Relatórios
// @Produce json
// @Param start_date query string false "Data inicial (YYYY-MM-DD)"
// @Param end_date query string false "Data final (YYYY-MM-DD)"
// @Success 200 {object} domain.TrialBalanceReport
// @Router /api/v1/reports/trial-balance [get]
func (h *ReportHandler) GetTrialBalance(w http.ResponseWriter, r *http.Request) {
	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", err.Error(), nil)
		return
	}

	report, err := h.service.GetTrialBalance(r.Context(), startDate, endDate)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, report)
}

func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)

	if s := r.URL.Query().Get("start_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return startDate, endDate, err
		}
		startDate = t
	}

	if e := r.URL.Query().Get("end_date"); e != "" {
		t, err := time.Parse("2006-01-02", e)
		if err != nil {
			return startDate, endDate, err
		}
		endDate = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
	}

	return startDate, endDate, nil
}

