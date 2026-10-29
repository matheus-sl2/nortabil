package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/middleware"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// NFeHandler processa requisições de importação e contabilização automática de XML de NF-e (SEFAZ Modelo 55)
type NFeHandler struct {
	service *service.NFeService
}

func NewNFeHandler(service *service.NFeService) *NFeHandler {
	return &NFeHandler{service: service}
}

type importNFeRequest struct {
	XMLContent      string `json:"xml_content"`
	DebitAccountID  string `json:"debit_account_id,omitempty"`
	CreditAccountID string `json:"credit_account_id,omitempty"`
}

// ImportXML recebe um arquivo XML de NF-e (via JSON ou multipart/form-data) e registra o lançamento em partidas dobradas
func (h *NFeHandler) ImportXML(w http.ResponseWriter, r *http.Request) {
	var rawXML string
	var debitStr, creditStr string

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			JSONError(w, http.StatusBadRequest, "INVALID_FORM", "Falha ao processar formulário multipart de NF-e", err.Error())
			return
		}
		debitStr = r.FormValue("debit_account_id")
		creditStr = r.FormValue("credit_account_id")

		file, _, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			data, readErr := io.ReadAll(file)
			if readErr != nil {
				JSONError(w, http.StatusBadRequest, "READ_ERROR", "Falha ao ler arquivo XML enviado", readErr.Error())
				return
			}
			rawXML = string(data)
		} else {
			rawXML = r.FormValue("xml_content")
		}
	} else {
		var req importNFeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Corpo da requisição JSON inválido", err.Error())
			return
		}
		rawXML = req.XMLContent
		debitStr = req.DebitAccountID
		creditStr = req.CreditAccountID
	}

	var debitID, creditID uuid.UUID
	if strings.TrimSpace(debitStr) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(debitStr))
		if err != nil {
			JSONError(w, http.StatusBadRequest, "INVALID_UUID", "UUID da conta de Débito inválido", nil)
			return
		}
		debitID = parsed
	}
	if strings.TrimSpace(creditStr) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(creditStr))
		if err != nil {
			JSONError(w, http.StatusBadRequest, "INVALID_UUID", "UUID da conta de Crédito inválido", nil)
			return
		}
		creditID = parsed
	}

	companyID := middleware.GetCompanyID(r)
	createdBy := middleware.GetUserIdentity(r)
	if createdBy == "" {
		createdBy = "Importador Fiscal NF-e"
	}

	result, err := h.service.ImportNFeXML(r.Context(), companyID, rawXML, debitID, creditID, createdBy)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusCreated, result)
}
