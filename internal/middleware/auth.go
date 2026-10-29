package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

type userContextKey string

const (
	ContextUserID   userContextKey = "user_id"
	ContextUserRole userContextKey = "user_role"
	ContextUserMail userContextKey = "user_email"
	ContextUserName userContextKey = "user_name"
)

// TokenValidator interface mínima para validação de token
type TokenValidator interface {
	ValidateToken(tokenString string) (interface{ GetClaims() (uuid.UUID, string, string, domain.UserRole) }, error)
}

// AuthMiddleware extrai e valida o token JWT do cabeçalho Authorization
func AuthMiddleware(validateFunc func(string) (uuid.UUID, string, string, domain.UserRole, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Permite continuar sem autenticação para endpoints públicos, mas não injeta claims
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, `{"success":false,"error":"TOKEN_MALFORMED","message":"Formato de cabeçalho Authorization inválido. Use 'Bearer <token>'"}`, http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]
			uid, email, name, role, err := validateFunc(tokenString)
			if err != nil {
				http.Error(w, `{"success":false,"error":"UNAUTHORIZED","message":"Token JWT inválido ou expirado"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ContextUserID, uid)
			ctx = context.WithValue(ctx, ContextUserRole, role)
			ctx = context.WithValue(ctx, ContextUserMail, email)
			ctx = context.WithValue(ctx, ContextUserName, name)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole restringe o endpoint a uma ou mais roles específicas
func RequireRole(allowedRoles ...domain.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleVal := r.Context().Value(ContextUserRole)
			if roleVal == nil {
				http.Error(w, `{"success":false,"error":"UNAUTHORIZED","message":"Autenticação requerida para este recurso"}`, http.StatusUnauthorized)
				return
			}

			userRole, ok := roleVal.(domain.UserRole)
			if !ok {
				http.Error(w, `{"success":false,"error":"FORBIDDEN","message":"Permissão não identificada"}`, http.StatusForbidden)
				return
			}

			// Admin possui acesso irrestrito
			if userRole == domain.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			for _, allowed := range allowedRoles {
				if userRole == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, `{"success":false,"error":"FORBIDDEN","message":"Seu perfil de usuário não tem permissão para executar esta operação"}`, http.StatusForbidden)
		})
	}
}

// GetUserRole recupera a Role do contexto da requisição
func GetUserRole(r *http.Request) domain.UserRole {
	if val := r.Context().Value(ContextUserRole); val != nil {
		if role, ok := val.(domain.UserRole); ok {
			return role
		}
	}
	return ""
}

