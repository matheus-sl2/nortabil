package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserRole define os níveis de permissão (RBAC) do sistema contábil
type UserRole string

const (
	RoleAdmin      UserRole = "ADMIN"      // Administrador do Escritório
	RoleContador   UserRole = "CONTADOR"   // Responsável Técnico Contábil (CRC)
	RoleAssistente UserRole = "ASSISTENTE" // Assistente / Digitador
)

var (
	ErrUserNotFound      = errors.New("usuário não encontrado")
	ErrInvalidPassword   = errors.New("senha incorreta")
	ErrEmailAlreadyInUse = errors.New("este endereço de e-mail já está em uso")
	ErrUnauthorized      = errors.New("acesso não autorizado: token ausente ou inválido")
	ErrForbidden         = errors.New("permissão insuficiente para executar esta operação")
)

// User representa o profissional ou colaborador do escritório de contabilidade
type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         UserRole  `json:"role"`
	CRCNumber    string    `json:"crc_number,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// HashPassword gera o hash Bcrypt seguro com salt para a senha fornecida
func HashPassword(plainPassword string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword compara a senha pura com o hash armazenado
func CheckPassword(plainPassword, passwordHash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(plainPassword))
	return err == nil
}

// ValidateRole verifica se o papel informado é válido
func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleContador, RoleAssistente:
		return true
	default:
		return false
	}
}

// HasPermission verifica se o usuário possui ao menos uma das roles requeridas
func (u *User) HasPermission(requiredRoles ...UserRole) bool {
	if !u.Active {
		return false
	}
	if u.Role == RoleAdmin {
		return true // Administrador tem acesso total
	}
	for _, req := range requiredRoles {
		if strings.EqualFold(string(u.Role), string(req)) {
			return true
		}
	}
	return false
}

