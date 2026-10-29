package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
)

// JWTClaims estrutura as reivindicações embutidas no token de autenticação
type JWTClaims struct {
	UserID    uuid.UUID       `json:"user_id"`
	Email     string          `json:"email"`
	Name      string          `json:"name"`
	Role      domain.UserRole `json:"role"`
	CRCNumber string          `json:"crc_number,omitempty"`
	jwt.RegisteredClaims
}

// AuthService gerencia a autenticação e geração de tokens
type AuthService struct {
	userRepo  *postgres.UserRepository
	jwtSecret []byte
}

// NewAuthService cria nova instância do serviço de autenticação
func NewAuthService(userRepo *postgres.UserRepository, jwtSecret string) *AuthService {
	if jwtSecret == "" {
		jwtSecret = "nortabil_secret_key_tcc_januaria_2026_super_segura"
	}
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

// LoginOutput dados retornados após sucesso na autenticação
type LoginOutput struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      *domain.User `json:"user"`
}

// Login valida o e-mail e a senha e emite o token JWT
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginOutput, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidPassword // Mensagem genérica por segurança
		}
		return nil, err
	}

	if !user.Active {
		return nil, errors.New("usuário desativado pelo administrador")
	}

	if !domain.CheckPassword(password, user.PasswordHash) {
		return nil, domain.ErrInvalidPassword
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	claims := JWTClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		CRCNumber: user.CRCNumber,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "nortabil-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("falha ao assinar token jwt: %w", err)
	}

	return &LoginOutput{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

// ValidateToken decodifica e valida a assinatura de um token JWT
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, domain.ErrUnauthorized
}

// RegisterUserInput parâmetros para criação de usuário
type RegisterUserInput struct {
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	Password  string          `json:"password"`
	Role      domain.UserRole `json:"role"`
	CRCNumber string          `json:"crc_number,omitempty"`
}

// Register cria um novo usuário com senha criptografada
func (s *AuthService) Register(ctx context.Context, input RegisterUserInput) (*domain.User, error) {
	if !input.Role.IsValid() {
		return nil, errors.New("função (role) inválida: informe ADMIN, CONTADOR ou ASSISTENTE")
	}

	if len(input.Password) < 6 {
		return nil, errors.New("a senha deve conter no mínimo 6 caracteres")
	}

	// Verifica se e-mail já existe
	existing, _ := s.userRepo.GetByEmail(ctx, input.Email)
	if existing != nil {
		return nil, domain.ErrEmailAlreadyInUse
	}

	hash, err := domain.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar hash da senha: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: hash,
		Role:         input.Role,
		CRCNumber:    input.CRCNumber,
		Active:       true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// ListUsers lista todos os usuários do escritório
func (s *AuthService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.userRepo.List(ctx)
}

