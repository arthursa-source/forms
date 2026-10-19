// Package store guarda o acesso ao banco: SQL escrito à mão, sem ORM.
// Cada arquivo cuida de uma entidade (users.go, forms.go, ...).
package store

import (
	"database/sql"
	"errors"
	"strings"
)

type Store struct {
	DB *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{DB: db}
}

// Erros de domínio: os handlers HTTP traduzem estes para status codes,
// em vez de inspecionar erros do Postgres.
var (
	ErrNotFound = errors.New("não encontrado")
	ErrConflict = errors.New("já existe")
	ErrInvalid  = errors.New("dados inválidos")
)

// isUniqueViolation reconhece o erro 23505 do Postgres (unique_violation).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}
