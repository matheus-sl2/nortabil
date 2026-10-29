package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/middleware"
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
func (h *ReportHandler) GetLedger(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
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

	report, err := h.service.GetLedger(r.Context(), companyID, startDate, endDate, accountID)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, report)
}

// GetDailyBook gera o Livro Diário Geral
func (h *ReportHandler) GetDailyBook(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", err.Error(), nil)
		return
	}

	report, err := h.service.GetDailyBook(r.Context(), companyID, startDate, endDate)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, report)
}

// GetTrialBalance gera o Balancete de Verificação
func (h *ReportHandler) GetTrialBalance(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", err.Error(), nil)
		return
	}

	report, err := h.service.GetTrialBalance(r.Context(), companyID, startDate, endDate)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, report)
}

// GetDRE gera a Demonstração do Resultado do Exercício
func (h *ReportHandler) GetDRE(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", err.Error(), nil)
		return
	}

	report, err := h.service.GetDRE(r.Context(), companyID, startDate, endDate)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, report)
}

// GetBalanceSheet gera o Balanço Patrimonial Estruturado
func (h *ReportHandler) GetBalanceSheet(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	refDate := time.Now().UTC()
	if dateParam := r.URL.Query().Get("date"); dateParam != "" {
		if t, err := time.Parse("2006-01-02", dateParam); err == nil {
			refDate = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
		}
	}

	report, err := h.service.GetBalanceSheet(r.Context(), companyID, refDate)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, report)
}

// GetIndicators gera os Indicadores Econômico-Financeiros (Liquidez, Endividamento, Margens, ROA e ROE)
func (h *ReportHandler) GetIndicators(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", err.Error(), nil)
		return
	}

	report, err := h.service.GetFinancialIndicators(r.Context(), companyID, startDate, endDate)
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
