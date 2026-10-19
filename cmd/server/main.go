// Comando `server`: serviço HTTP independente do Form Builder.
//
// Subcomandos:
//
//	server run     [--address=... --database-url=... --jwt-secret=...]
//	server migrate [up|down|status]
//
// Todas as opções também podem vir de variáveis de ambiente
// (ADDRESS, DATABASE_URL, JWT_SECRET); a flag sobrescreve o ambiente.
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
  run                  sobe o servidor HTTP
  migrate [up|down|status]  aplica/desfaz migrations

flags (ou env ADDRESS / DATABASE_URL / JWT_SECRET):
  --address        endereço HTTP (padrão localhost:8080)
  --database-url   string de conexão do Postgres
  --jwt-secret     segredo para assinar os JWTs
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

	router, _ := httpapi.NewRouter(&httpapi.Server{DB: db, JWTSecret: cfg.JWTSecret})

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
