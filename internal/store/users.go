package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type User struct {
	ID    int64
	Email string
	// Vazio quando a conta só entra pelo Google (password_hash NULL).
	PasswordHash string
	// "sub" da conta do Google; vazio quando a conta é só de e-mail/senha.
	GoogleID  string
	CreatedAt time.Time
}

// Colunas lidas em todas as buscas de usuário, para as queries e o Scan
// não saírem de sincronia.
const userColumns = `id, email, password_hash, google_id, created_at`

// CreateUser insere o usuário com senha. Se o e-mail já existir, o UNIQUE
// do banco dispara erro e devolvemos ErrConflict.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	u := User{Email: normalizeEmail(email), PasswordHash: passwordHash}

	err := s.DB.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, created_at`,
		u.Email, u.PasswordHash,
	).Scan(&u.ID, &u.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	return u, nil
}

// CreateGoogleUser cria a conta de quem entrou pelo Google: sem senha,
// com o google_id preenchido.
func (s *Store) CreateGoogleUser(ctx context.Context, email, googleID string) (User, error) {
	u := User{Email: normalizeEmail(email), GoogleID: googleID}

	err := s.DB.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash, google_id)
		 VALUES ($1, NULL, $2)
		 RETURNING id, created_at`,
		u.Email, u.GoogleID,
	).Scan(&u.ID, &u.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	return u, nil
}

// LinkGoogleID vincula uma conta de e-mail/senha que já existe à conta do
// Google do mesmo e-mail. A senha continua valendo: depois disso o usuário
// pode entrar das duas formas.
func (s *Store) LinkGoogleID(ctx context.Context, userID int64, googleID string) (User, error) {
	var u User
	err := s.DB.QueryRowContext(ctx,
		`UPDATE users SET google_id = $2
		 WHERE id = $1
		 RETURNING `+userColumns, userID, googleID,
	).Scan(&u.ID, &u.Email, &nullable{&u.PasswordHash}, &nullable{&u.GoogleID}, &u.CreatedAt)

	if errors.Is(err, errNoRows) {
		return User{}, ErrNotFound
	}
	if isUniqueViolation(err) {
		// O google_id já está vinculado a outro e-mail.
		return User{}, ErrConflict
	}
	return u, err
}

// UserByEmail busca o usuário para o login.
func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	return s.userBy(ctx, `email = $1`, normalizeEmail(email))
}

// UserByGoogleID encontra quem já entrou pelo Google antes.
func (s *Store) UserByGoogleID(ctx context.Context, googleID string) (User, error) {
	return s.userBy(ctx, `google_id = $1`, googleID)
}

// UserByID é usado pelo endpoint /auth/me.
func (s *Store) UserByID(ctx context.Context, id int64) (User, error) {
	return s.userBy(ctx, `id = $1`, id)
}

// userBy evita repetir a mesma lista de colunas e o mesmo Scan em cada
// busca. `where` é SQL fixo escrito aqui no pacote, nunca entrada do
// usuário — o valor é que vai como parâmetro ($1).
func (s *Store) userBy(ctx context.Context, where string, arg any) (User, error) {
	var u User
	err := s.DB.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE `+where, arg,
	).Scan(&u.ID, &u.Email, &nullable{&u.PasswordHash}, &nullable{&u.GoogleID}, &u.CreatedAt)

	if errors.Is(err, errNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// nullable lê uma coluna TEXT que aceita NULL para dentro de uma string
// comum: NULL vira "".
type nullable struct{ dest *string }

func (n *nullable) Scan(value any) error {
	var s sql.NullString
	if err := s.Scan(value); err != nil {
		return err
	}
	*n.dest = s.String
	return nil
}

// E-mails são comparados sem diferenciar maiúsculas e espaços nas pontas.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
