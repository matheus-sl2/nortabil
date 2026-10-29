package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/middleware"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// SPEDHandler gerencia a exportação da Escrituração Contábil Digital (SPED ECD)
type SPEDHandler struct {
	spedService *service.SPEDService
}

// NewSPEDHandler cria uma nova instância do handler de SPED
func NewSPEDHandler(spedService *service.SPEDService) *SPEDHandler {
	return &SPEDHandler{spedService: spedService}
}

// ExportECD gera e faz o download do arquivo de escrituração contábil digital oficial
func (h *SPEDHandler) ExportECD(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	q := r.URL.Query()

	// Ano base de referência (padrão ano corrente)
	year := time.Now().Year()
	if yearStr := q.Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 1900 {
			year = y
		}
	}

	startDate := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, time.December, 31, 23, 59, 59, 999999999, time.UTC)

	if s := q.Get("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = t
		}
	}
	if e := q.Get("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			endDate = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
		}
	}

	cfg := domain.SPEDExportConfig{
		CompanyID:     companyID,
		StartDate:     startDate,
		EndDate:       endDate,
		BookType:      "G",
		AuditorCRC:    q.Get("auditor_crc"),
		AuditorName:   q.Get("auditor_name"),
		AuditorCPF:    q.Get("auditor_cpf"),
		SignatoryName: q.Get("signatory_name"),
		SignatoryCPF:  q.Get("signatory_cpf"),
	}

	content, err := h.spedService.GenerateECD(r.Context(), cfg)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	filename := fmt.Sprintf("SPED_ECD_%d_%s.txt", year, companyID.String()[:8])
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

