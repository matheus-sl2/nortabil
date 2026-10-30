package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/middleware"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// AuthHandler gerencia os endpoints de autenticação e RBAC
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler instancia o handler de autenticação
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// LoginPayload corpo da requisição de login
type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login autentica o usuário e emite o token JWT
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var p LoginPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_BODY", "Corpo da requisição inválido", err.Error())
		return
	}

	res, err := h.authService.Login(r.Context(), p.Email, p.Password)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, res)
}

// Register cria um novo usuário no sistema
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input service.RegisterUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_BODY", "Corpo da requisição inválido", err.Error())
		return
	}

	user, err := h.authService.Register(r.Context(), input)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusCreated, user)
}

// Me retorna os dados do usuário autenticado a partir do token
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	uidVal := r.Context().Value(middleware.ContextUserID)
	mailVal := r.Context().Value(middleware.ContextUserMail)
	nameVal := r.Context().Value(middleware.ContextUserName)

	if uidVal == nil {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Nenhum usuário autenticado", nil)
		return
	}

	uid, _ := uidVal.(uuid.UUID)
	email, _ := mailVal.(string)
	name, _ := nameVal.(string)

	JSON(w, http.StatusOK, map[string]interface{}{
		"id":    uid,
		"name":  name,
		"email": email,
		"role":  role,
	})
}

// ListUsers lista todos os usuários do escritório
func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.authService.ListUsers(r.Context())
	if err != nil {
		HandleDomainError(w, err)
		return
	}
	JSON(w, http.StatusOK, users)
}

// ExportMyData exporta todos os dados pessoais do titular autenticado em formato estruturado (Art. 18, II e V da LGPD)
func (h *AuthHandler) ExportMyData(w http.ResponseWriter, r *http.Request) {
	uidVal := r.Context().Value(middleware.ContextUserID)
	if uidVal == nil {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Nenhum usuário autenticado", nil)
		return
	}
	uid, _ := uidVal.(uuid.UUID)

	data, err := h.authService.ExportPersonalData(r.Context(), uid)
	if err != nil {
		HandleDomainError(w, err)
		return
	}
	JSON(w, http.StatusOK, data)
}

// DeleteMyData processa a solicitação de exclusão/anonimização dos dados pessoais do titular (Art. 18, VI da LGPD)
func (h *AuthHandler) DeleteMyData(w http.ResponseWriter, r *http.Request) {
	uidVal := r.Context().Value(middleware.ContextUserID)
	if uidVal == nil {
		JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Nenhum usuário autenticado", nil)
		return
	}
	uid, _ := uidVal.(uuid.UUID)

	if err := h.authService.RequestAccountDeletion(r.Context(), uid); err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"message": "Seus dados pessoais identificáveis foram anonimizados e sua conta foi desativada em conformidade com o Art. 18, VI da Lei nº 13.709/2018 (LGPD).",
	})
}


