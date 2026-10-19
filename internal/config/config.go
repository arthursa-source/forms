// Package config resolve a configuração da aplicação.
//
// Ordem de precedência (do menor para o maior):
//  1. valores padrão (bons para desenvolvimento local)
//  2. variáveis de ambiente (ADDRESS, DATABASE_URL, JWT_SECRET)
//  3. flags de linha de comando (--address, --database-url, --jwt-secret)
//
// Flags e env podem ser combinados: uma flag só sobrescreve o ambiente
// quando é informada explicitamente.
package config

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	Address     string
	DatabaseURL string
	JWTSecret   string
}

// Valores padrão: apontam para o Postgres do docker-compose.yml.
const (
	defaultAddress     = "localhost:8080"
	defaultDatabaseURL = "postgres://forms:forms@localhost:5432/forms?sslmode=disable"
	defaultJWTSecret   = "dev-secret-troque-isto"
)

// Load monta a configuração a partir do ambiente e dos args recebidos
// (normalmente os argumentos que vêm depois do subcomando).
func Load(args []string) (Config, error) {
	// 1 e 2: padrão, sobrescrito pelo ambiente quando a variável existe.
	cfg := Config{
		Address:     envOr("ADDRESS", defaultAddress),
		DatabaseURL: envOr("DATABASE_URL", defaultDatabaseURL),
		JWTSecret:   envOr("JWT_SECRET", defaultJWTSecret),
	}

	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	// Os defaults das flags são os valores já resolvidos acima, então mesmo
	// que o usuário não passe a flag o valor correto é mantido.
	address := fs.String("address", cfg.Address, "endereço HTTP (host:porta)")
	databaseURL := fs.String("database-url", cfg.DatabaseURL, "string de conexão do Postgres")
	jwtSecret := fs.String("jwt-secret", cfg.JWTSecret, "segredo para assinar os JWTs")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	// 3: a flag vence. Como o default da flag já era o valor do ambiente,
	// basta ler o ponteiro.
	cfg.Address = *address
	cfg.DatabaseURL = *databaseURL
	cfg.JWTSecret = *jwtSecret

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL (ou --database-url) é obrigatório")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
