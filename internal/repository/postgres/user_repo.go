package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// UserRepository gerencia a persistência de usuários e credenciais
type UserRepository struct {
	db *DB
}

// NewUserRepository cria nova instância do repositório de usuários
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create insere um novo usuário no banco
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	now := time.Now()
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	u.CreatedAt = now
	u.UpdatedAt = now

	query := `
		INSERT INTO users (id, name, email, password_hash, role, crc_number, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		u.ID,
		u.Name,
		u.Email,
		u.PasswordHash,
		u.Role,
		u.CRCNumber,
		u.Active,
		u.CreatedAt,
		u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("erro ao inserir usuário: %w", err)
	}
	return nil
}

// GetByEmail busca um usuário pelo seu e-mail único
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, role, COALESCE(crc_number, ''), active, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`
	var u domain.User
	var roleStr string
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.PasswordHash,
		&roleStr,
		&u.CRCNumber,
		&u.Active,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("erro ao buscar usuário por email: %w", err)
	}
	u.Role = domain.UserRole(roleStr)
	return &u, nil
}

// GetByID busca um usuário por ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, role, COALESCE(crc_number, ''), active, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u domain.User
	var roleStr string
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.PasswordHash,
		&roleStr,
		&u.CRCNumber,
		&u.Active,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("erro ao buscar usuário por id: %w", err)
	}
	u.Role = domain.UserRole(roleStr)
	return &u, nil
}

// List retorna todos os usuários cadastrados
func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	query := `
		SELECT id, name, email, role, COALESCE(crc_number, ''), active, created_at, updated_at
		FROM users
		ORDER BY name ASC
	`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar usuários: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		var roleStr string
		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&roleStr,
			&u.CRCNumber,
			&u.Active,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler usuário: %w", err)
		}
		u.Role = domain.UserRole(roleStr)
		users = append(users, u)
	}
	return users, nil
}

