package tests

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// TestBcryptHashing valida a segurança e geração de salt das senhas
func TestBcryptHashing(t *testing.T) {
	plain := "MinhaSenhaForte@2026"
	hash, err := domain.HashPassword(plain)
	if err != nil {
		t.Fatalf("Erro ao gerar hash: %v", err)
	}

	if hash == plain {
		t.Errorf("O hash não pode ser igual à senha pura")
	}

	if !domain.CheckPassword(plain, hash) {
		t.Errorf("A senha correta não foi validada pelo hash")
	}

	if domain.CheckPassword("SenhaIncorreta@123", hash) {
		t.Errorf("Senha incorreta foi aceita indevidamente!")
	}
}

// TestUserRolePermissions valida a hierarquia de perfis de acesso RBAC
func TestUserRolePermissions(t *testing.T) {
	admin := domain.User{Role: domain.RoleAdmin, Active: true}
	contador := domain.User{Role: domain.RoleContador, Active: true}
	assistente := domain.User{Role: domain.RoleAssistente, Active: true}
	inativo := domain.User{Role: domain.RoleAdmin, Active: false}

	// 1. Apenas Contador e Admin podem fechar períodos fiscais
	if !admin.HasPermission(domain.RoleContador) {
		t.Errorf("Admin deve ter acesso a recursos de Contador")
	}
	if !contador.HasPermission(domain.RoleContador) {
		t.Errorf("Contador deve ter acesso aos seus próprios recursos")
	}
	if assistente.HasPermission(domain.RoleContador) {
		t.Errorf("Assistente NÃO deve ter permissão para fechar período ou ARE!")
	}

	// 2. Usuário inativo não tem permissão
	if inativo.HasPermission(domain.RoleAdmin, domain.RoleContador) {
		t.Errorf("Usuário inativo não deve ter permissão alguma")
	}
}

// TestJWTTokenSignAndValidate valida a emissão e integridade de claims JWT
func TestJWTTokenSignAndValidate(t *testing.T) {
	secret := "chave_secreta_para_testes_tcc_januaria_123"
	authSvc := service.NewAuthService(nil, secret)

	userID := uuid.New()
	email := "contador@nortabil.com.br"
	name := "Sebastião Ferreira da Silva"
	role := domain.RoleContador

	claims := service.JWTClaims{
		UserID:    userID,
		Email:     email,
		Name:      name,
		Role:      role,
		CRCNumber: "MG-085421/O",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "nortabil-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Erro ao assinar token: %v", err)
	}

	// Valida o token
	parsed, err := authSvc.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("Erro ao validar token JWT: %v", err)
	}

	if parsed.UserID != userID {
		t.Errorf("UserID divergente: esperado %s, obteve %s", userID, parsed.UserID)
	}
	if parsed.Role != role {
		t.Errorf("Role divergente: esperado %s, obteve %s", role, parsed.Role)
	}
	if parsed.Email != email {
		t.Errorf("Email divergente: esperado %s, obteve %s", email, parsed.Email)
	}

	// Token adulterado deve ser rejeitado
	fakeToken := tokenStr + "adulterado"
	_, err = authSvc.ValidateToken(fakeToken)
	if err == nil {
		t.Errorf("Token com assinatura inválida deveria ser rejeitado!")
	}
}

