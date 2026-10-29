package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

type tenantKey string

const CompanyIDContextKey tenantKey = "company_id"

// TenantMiddleware extrai e valida a empresa do contexto da requisição
func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var compID uuid.UUID

		// 1. URL Param (ex: /api/v1/companies/{companyId}/...)
		if paramID := chi.URLParam(r, "companyId"); paramID != "" {
			if id, err := uuid.Parse(paramID); err == nil {
				compID = id
			}
		}

		// 2. Header HTTP X-Company-ID
		if compID == uuid.Nil {
			if headerID := r.Header.Get("X-Company-ID"); headerID != "" {
				if id, err := uuid.Parse(headerID); err == nil {
					compID = id
				}
			}
		}

		// 3. Query Param ?company_id=
		if compID == uuid.Nil {
			if qID := r.URL.Query().Get("company_id"); qID != "" {
				if id, err := uuid.Parse(qID); err == nil {
					compID = id
				}
			}
		}

		// 4. Fallback para empresa matriz padrão
		if compID == uuid.Nil {
			compID = domain.DefaultCompanyID
		}

		ctx := context.WithValue(r.Context(), CompanyIDContextKey, compID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCompanyID recupera o UUID da empresa a partir do contexto da requisição
func GetCompanyID(r *http.Request) uuid.UUID {
	if val := r.Context().Value(CompanyIDContextKey); val != nil {
		if id, ok := val.(uuid.UUID); ok {
			return id
		}
	}
	return domain.DefaultCompanyID
}

