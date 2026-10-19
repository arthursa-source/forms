package store

import (
	"context"
	"errors"
	"strings"
	"time"
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// CreateUser insere o usuário. Se o e-mail já existir, o UNIQUE do banco
// dispara erro e devolvemos ErrConflict.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	u := User{Email: normalizeEmail(email), PasswordHash: passwordHash}

	err := s.DB.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, created_at`,
		u.Email, u.PasswordHash,
	).Scan(&u.ID, &u.CreatedAt)

	if err != nil {
		// 23505 = unique_violation. Checamos pelo texto para não puxar
		// o pacote de erros do pgx só por isso.
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key") {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	return u, nil
}

// UserByEmail busca o usuário para o login.
func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at
		 FROM users WHERE email = $1`,
		normalizeEmail(email),
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	if errors.Is(err, errNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// UserByID é usado pelo endpoint /auth/me.
func (s *Store) UserByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	if errors.Is(err, errNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// E-mails são comparados sem diferenciar maiúsculas e espaços nas pontas.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
