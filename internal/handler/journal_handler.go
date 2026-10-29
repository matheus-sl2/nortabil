package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// JournalHandler processa requisições de lançamentos contábeis
type JournalHandler struct {
	service *service.JournalService
}

// NewJournalHandler cria uma nova instância do handler de lançamentos
func NewJournalHandler(service *service.JournalService) *JournalHandler {
	return &JournalHandler{service: service}
}

// Create cria um lançamento contábil individual aplicando a Lei de Luca Pacioli
// @Summary Criar lançamento contábil
// @Description Registra um novo fato contábil validando a igualdade estrita entre Débitos e Créditos
// @Tags Lançamentos
// @Accept json
// @Produce json
// @Success 201 {object} domain.JournalEntry
// @Router /api/v1/journal-entries [post]
func (h *JournalHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateJournalEntryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Corpo da requisição JSON inválido", err.Error())
		return
	}

	entry, err := h.service.CreateEntry(r.Context(), input)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusCreated, entry)
}

// CreateBatch processa uma lista de lançamentos contábeis de alta vazão
// @Summary Criar lote de lançamentos contábeis
// @Description Recebe centenas ou milhares de lançamentos em lote com processamento concorrente
// @Tags Lançamentos
// @Accept json
// @Produce json
// @Success 200 {object} service.BatchProcessReport
// @Router /api/v1/journal-entries/batch [post]
func (h *JournalHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var input []service.CreateJournalEntryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Corpo da requisição JSON inválido (esperado array de lançamentos)", err.Error())
		return
	}

	report := h.service.CreateBatchEntries(r.Context(), input)
	JSON(w, http.StatusOK, report)
}

// GetByID busca um lançamento pelo ID UUID
// @Summary Obter lançamento por ID
// @Tags Lançamentos
// @Produce json
// @Param id path string true "UUID do lançamento"
// @Success 200 {object} domain.JournalEntry
// @Router /api/v1/journal-entries/{id} [get]
func (h *JournalHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_UUID", "Identificador UUID do lançamento inválido", nil)
		return
	}

	entry, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, entry)
}

// List lista lançamentos com filtros e paginação
// @Summary Listar lançamentos contábeis
// @Tags Lançamentos
// @Produce json
// @Param start_date query string false "Data inicial (YYYY-MM-DD)"
// @Param end_date query string false "Data final (YYYY-MM-DD)"
// @Param status query string false "Status (POSTED, DRAFT, REVERSED)"
// @Param limit query int false "Limite por página (padrão: 50)"
// @Param offset query int false "Deslocamento de página"
// @Success 200 {array} domain.JournalEntry
// @Router /api/v1/journal-entries [get]
func (h *JournalHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := postgres.ListFilter{
		Status: q.Get("status"),
		Limit:  50,
		Offset: 0,
	}

	if l := q.Get("limit"); l != "" {
		if limitInt, err := strconv.Atoi(l); err == nil && limitInt > 0 {
			filter.Limit = limitInt
		}
	}
	if o := q.Get("offset"); o != "" {
		if offsetInt, err := strconv.Atoi(o); err == nil && offsetInt >= 0 {
			filter.Offset = offsetInt
		}
	}
	if s := q.Get("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.StartDate = &t
		}
	}
	if e := q.Get("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			filter.EndDate = &t
		}
	}

	entries, total, err := h.service.ListEntries(r.Context(), filter)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	response := map[string]interface{}{
		"total":   total,
		"limit":   filter.Limit,
		"offset":  filter.Offset,
		"entries": entries,
	}

	JSON(w, http.StatusOK, response)
}

