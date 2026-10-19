// Comando `server`: serviço HTTP independente do Form Builder.
//
// Subcomandos:
//
//	server run     [--address=... --database-url=... --jwt-secret=...]
//	server migrate [up|down|status]
//
// Todas as opções também podem vir de variáveis de ambiente
// (ADDRESS, DATABASE_URL, JWT_SECRET, FRONTEND_URL, GOOGLE_CLIENT_ID,
// GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URL); a flag sobrescreve o ambiente.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arthursa-source/forms/internal/config"
	"github.com/arthursa-source/forms/internal/database"
	"github.com/arthursa-source/forms/internal/httpapi"
	"github.com/arthursa-source/forms/internal/store"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("subcomando obrigatório")
	}

	switch args[0] {
	case "run":
		return cmdRun(args[1:])
	case "migrate":
		return cmdMigrate(args[1:])
	case "openapi":
		return cmdOpenAPI()
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("subcomando desconhecido: %q", args[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `uso: server <subcomando> [flags]

subcomandos:
  run                       sobe o servidor HTTP
  migrate [up|down|status]  aplica/desfaz migrations
  openapi                   imprime a spec OpenAPI 3 (YAML) no stdout

flags (ou as env vars equivalentes em maiúsculas, ex. DATABASE_URL):
  --address                endereço HTTP (padrão localhost:8080)
  --database-url           string de conexão do Postgres
  --jwt-secret             segredo para assinar os JWTs
  --frontend-url           origem do frontend (padrão http://localhost:5173)
  --google-client-id       client ID do OAuth2 do Google (opcional)
  --google-client-secret   client secret do OAuth2 do Google (opcional)
  --google-redirect-url    redirect URI registrado no Google (opcional)
`)
}

func cmdRun(args []string) error {
	cfg, err := config.Load(args)
	if err != nil {
		return err
	}

	// Contexto cancelado em Ctrl+C / SIGTERM: base do shutdown graceful.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	router, _ := httpapi.NewRouter(&httpapi.Server{
		Store:              store.New(db),
		JWTSecret:          cfg.JWTSecret,
		FrontendURL:        cfg.FrontendURL,
		GoogleClientID:     cfg.GoogleClientID,
		GoogleClientSecret: cfg.GoogleClientSecret,
		GoogleRedirectURL:  cfg.GoogleRedirectURL,
	})

	if !cfg.GoogleEnabled() {
		fmt.Println("aviso: login com Google desligado (defina GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET e GOOGLE_REDIRECT_URL)")
	}

	srv := &http.Server{
		Addr:              cfg.Address,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// O servidor roda numa goroutine para o main poder esperar o sinal.
	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("servidor ouvindo em http://%s (health: /health, docs: /docs)\n", cfg.Address)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		fmt.Println("\nencerrando...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// cmdOpenAPI imprime a spec gerada pelo huma a partir das structs Go.
// Não precisa de banco: só montamos o roteador para registrar as rotas.
func cmdOpenAPI() error {
	_, api := httpapi.NewRouter(&httpapi.Server{})

	spec, err := api.OpenAPI().YAML()
	if err != nil {
		return fmt.Errorf("gerando OpenAPI: %w", err)
	}
	_, err = os.Stdout.Write(spec)
	return err
}

func cmdMigrate(args []string) error {
	direction := "up"
	// O primeiro argumento posicional, se não for flag, é a direção.
	if len(args) > 0 && args[0][0] != '-' {
		direction = args[0]
		args = args[1:]
	}

	cfg, err := config.Load(args)
	if err != nil {
		return err
	}

	db, err := database.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	return database.Migrate(db, direction)
}
