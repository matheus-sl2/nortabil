package handler

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/middleware"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// ReconciliationHandler processa requisições de conciliação bancária e extratos OFX
type ReconciliationHandler struct {
	ofxService *service.OFXService
}

// NewReconciliationHandler cria uma nova instância do handler
func NewReconciliationHandler(ofxService *service.OFXService) *ReconciliationHandler {
	return &ReconciliationHandler{ofxService: ofxService}
}

// ImportOFX processa o arquivo de extrato bancário OFX e gera os lançamentos contábeis
func (h *ReconciliationHandler) ImportOFX(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)

	var bankAccountID, counterpartID uuid.UUID
	var fileContent io.Reader

	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Upload de arquivo via formulário multipart
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
			JSONError(w, http.StatusBadRequest, "INVALID_FORM", "Falha ao processar formulário multipart", err.Error())
			return
		}

		bankIDStr := r.FormValue("bank_account_id")
		counterpartIDStr := r.FormValue("counterpart_account_id")

		var err error
		bankAccountID, err = uuid.Parse(bankIDStr)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "INVALID_UUID", "bank_account_id inválido", nil)
			return
		}
		counterpartID, err = uuid.Parse(counterpartIDStr)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "INVALID_UUID", "counterpart_account_id inválido", nil)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			JSONError(w, http.StatusBadRequest, "MISSING_FILE", "Arquivo 'file' não fornecido na requisição", nil)
			return
		}
		defer file.Close()

		buf := new(bytes.Buffer)
		_, _ = io.Copy(buf, file)
		fileContent = buf
	} else {
		// Envio direto do arquivo no corpo da requisição com parâmetros na Query String
		q := r.URL.Query()
		var err error
		bankAccountID, err = uuid.Parse(q.Get("bank_account_id"))
		if err != nil {
			JSONError(w, http.StatusBadRequest, "INVALID_UUID", "Parâmetro query bank_account_id inválido", nil)
			return
		}
		counterpartID, err = uuid.Parse(q.Get("counterpart_account_id"))
		if err != nil {
			JSONError(w, http.StatusBadRequest, "INVALID_UUID", "Parâmetro query counterpart_account_id inválido", nil)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil || len(bodyBytes) == 0 {
			JSONError(w, http.StatusBadRequest, "EMPTY_BODY", "Corpo da requisição OFX vazio", nil)
			return
		}
		fileContent = bytes.NewReader(bodyBytes)
	}

	input := service.ImportOFXInput{
		BankAccountID:        bankAccountID,
		DefaultCounterpartID: counterpartID,
	}

	summary, err := h.ofxService.ImportOFX(r.Context(), companyID, input, fileContent)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, summary)
}

// ListItems lista as transações bancárias conciliadas e pendentes
func (h *ReconciliationHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	bankAccountIDStr := r.URL.Query().Get("bank_account_id")
	bankAccountID, err := uuid.Parse(bankAccountIDStr)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_UUID", "Parâmetro bank_account_id é obrigatório e deve ser um UUID", nil)
		return
	}

	onlyUnreconciled := r.URL.Query().Get("unreconciled_only") == "true"
	items, err := h.ofxService.ListItems(r.Context(), companyID, bankAccountID, onlyUnreconciled)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, items)
}

