package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config armazena todas as configurações do sistema Nortabil
type Config struct {
	Environment        string
	Port               string
	DatabaseURL        string
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime  time.Duration
	AutoMigrate        bool
	AutoSeed           bool
}

// Load carrega as variáveis de ambiente a partir do arquivo .env ou do ambiente operacional
func Load() (*Config, error) {
	// Carrega .env se existir, ignorando erro se não existir (ex: em produção/Docker)
	_ = godotenv.Load()

	cfg := &Config{
		Environment:       getEnv("ENVIRONMENT", "development"),
		Port:              getEnv("PORT", "8080"),
		DBMaxConns:        int32(getEnvAsInt("DB_MAX_CONNS", 25)),
		DBMinConns:        int32(getEnvAsInt("DB_MIN_CONNS", 5)),
		DBMaxConnLifetime: time.Duration(getEnvAsInt("DB_MAX_CONN_LIFETIME_MINUTES", 30)) * time.Minute,
		AutoMigrate:       getEnvAsBool("AUTO_MIGRATE", true),
		AutoSeed:          getEnvAsBool("AUTO_SEED", true),
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "postgres")
		pass := getEnv("DB_PASSWORD", "postgres")
		name := getEnv("DB_NAME", "nortabil_db")
		ssl := getEnv("DB_SSLMODE", "disable")
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, name, ssl)
	}
	cfg.DatabaseURL = dbURL

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

