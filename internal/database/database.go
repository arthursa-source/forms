// Package database abre e verifica a conexão com o Postgres.
//
// Usamos database/sql com o driver pgx no modo "stdlib": assim o mesmo
// *sql.DB serve para as queries da aplicação e para o goose (migrations).
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registra o driver "pgx"
)

// Open abre o pool de conexões e faz um ping para falhar rápido se o
// banco não estiver de pé.
func Open(ctx context.Context, databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("abrindo conexão: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("conectando no Postgres: %w", err)
	}
	return db, nil
}
