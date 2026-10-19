// Package config resolve a configuração da aplicação.
//
// Ordem de precedência (do menor para o maior):
//  1. valores padrão (bons para desenvolvimento local)
//  2. variáveis de ambiente (ADDRESS, DATABASE_URL, JWT_SECRET, ...)
//  3. flags de linha de comando (--address, --database-url, --jwt-secret, ...)
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
	FrontendURL string

	// Credenciais do OAuth2 do Google. Ficam vazias por padrão: sem elas
	// o login com Google apenas responde 503 e o resto da API segue
	// funcionando normalmente.
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

// Valores padrão: apontam para o Postgres do docker-compose.yml e para o
// frontend do Vite.
const (
	defaultAddress     = "localhost:8080"
	defaultDatabaseURL = "postgres://forms:forms@localhost:5432/forms?sslmode=disable"
	defaultJWTSecret   = "dev-secret-troque-isto"
	defaultFrontendURL = "http://localhost:5173"
)

// GoogleEnabled diz se dá para iniciar o fluxo do Google. Sem as três
// informações não há como montar a URL de consentimento nem trocar o
// código pelo token.
func (c Config) GoogleEnabled() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != "" && c.GoogleRedirectURL != ""
}

// Load monta a configuração a partir do ambiente e dos args recebidos
// (normalmente os argumentos que vêm depois do subcomando).
func Load(args []string) (Config, error) {
	// 1 e 2: padrão, sobrescrito pelo ambiente quando a variável existe.
	cfg := Config{
		Address:            envOr("ADDRESS", defaultAddress),
		DatabaseURL:        envOr("DATABASE_URL", defaultDatabaseURL),
		JWTSecret:          envOr("JWT_SECRET", defaultJWTSecret),
		FrontendURL:        envOr("FRONTEND_URL", defaultFrontendURL),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	}

	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	// Os defaults das flags são os valores já resolvidos acima, então mesmo
	// que o usuário não passe a flag o valor correto é mantido.
	address := fs.String("address", cfg.Address, "endereço HTTP (host:porta)")
	databaseURL := fs.String("database-url", cfg.DatabaseURL, "string de conexão do Postgres")
	jwtSecret := fs.String("jwt-secret", cfg.JWTSecret, "segredo para assinar os JWTs")
	frontendURL := fs.String("frontend-url", cfg.FrontendURL, "origem do frontend, para onde a callback do Google volta")
	googleClientID := fs.String("google-client-id", cfg.GoogleClientID, "client ID do OAuth2 do Google")
	googleClientSecret := fs.String("google-client-secret", cfg.GoogleClientSecret, "client secret do OAuth2 do Google")
	googleRedirectURL := fs.String("google-redirect-url", cfg.GoogleRedirectURL, "redirect URI registrado no Google Cloud Console")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	// 3: a flag vence. Como o default da flag já era o valor do ambiente,
	// basta ler o ponteiro.
	cfg.Address = *address
	cfg.DatabaseURL = *databaseURL
	cfg.JWTSecret = *jwtSecret
	cfg.FrontendURL = *frontendURL
	cfg.GoogleClientID = *googleClientID
	cfg.GoogleClientSecret = *googleClientSecret
	cfg.GoogleRedirectURL = *googleRedirectURL

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
