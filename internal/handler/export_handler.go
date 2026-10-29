package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/matheus-sl2/nortabil/internal/middleware"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// ExportHandler processa o download de relatórios em CSV e HTML para impressão PDF
type ExportHandler struct {
	exportService *service.ExportService
}

// NewExportHandler cria nova instância do handler de exportação
func NewExportHandler(exportService *service.ExportService) *ExportHandler {
	return &ExportHandler{exportService: exportService}
}

// ExportCSV gera e faz download de relatórios em formato CSV (Excel)
func (h *ExportHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", err.Error(), nil)
		return
	}

	reportType := strings.ToLower(r.URL.Query().Get("type"))
	var data []byte
	var filename string

	switch reportType {
	case "dre":
		data, filename, err = h.exportService.ExportDRECSV(r.Context(), companyID, startDate, endDate)
	default: // balancete
		data, filename, err = h.exportService.ExportTrialBalanceCSV(r.Context(), companyID, startDate, endDate)
	}

	if err != nil {
		HandleDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// ExportPrintable gera layout oficial com CSS pronto para visualização e impressão em PDF
func (h *ExportHandler) ExportPrintable(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", err.Error(), nil)
		return
	}

	reportType := r.URL.Query().Get("type")
	html, err := h.exportService.ExportPrintableHTML(r.Context(), companyID, reportType, startDate, endDate)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

