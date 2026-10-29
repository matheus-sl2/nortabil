package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matheus-sl2/nortabil/internal/config"
)

//go:embed migrations/001_initial_schema.sql
var migration001 string

//go:embed migrations/002_seed_accounts.sql
var migration002 string

// DB encapsula o pool de conexões pgxpool
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB inicializa o pool de conexões com PostgreSQL
func NewDB(ctx context.Context, cfg *config.Config) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao analisar string de conexão com postgres: %w", err)
	}

	poolConfig.MaxConns = cfg.DBMaxConns
	poolConfig.MinConns = cfg.DBMinConns
	poolConfig.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar pool de conexões pgx: %w", err)
	}

	// Testa a conectividade com o banco
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("não foi possível conectar ao PostgreSQL: %w", err)
	}

	db := &DB{Pool: pool}

	// Executa migrações se configurado
	if cfg.AutoMigrate {
		if err := db.RunMigrations(ctx, cfg.AutoSeed); err != nil {
			return nil, fmt.Errorf("falha ao executar migrações de banco de dados: %w", err)
		}
	}

	return db, nil
}

// Close fecha o pool de conexões
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// RunMigrations aplica os scripts SQL de esquema e seeds iniciais
func (db *DB) RunMigrations(ctx context.Context, autoSeed bool) error {
	log.Println("[MIGRAÇÃO] Aplicando esquema inicial (001_initial_schema.sql)...")
	if _, err := db.Pool.Exec(ctx, migration001); err != nil {
		return fmt.Errorf("erro na migração 001: %w", err)
	}

	if autoSeed {
		log.Println("[MIGRAÇÃO] Aplicando plano de contas referencial padrão (002_seed_accounts.sql)...")
		if _, err := db.Pool.Exec(ctx, migration002); err != nil {
			return fmt.Errorf("erro na migração 002 (seed): %w", err)
		}
	}

	log.Println("[MIGRAÇÃO] Esquema e tabelas verificados com sucesso.")
	return nil
}

