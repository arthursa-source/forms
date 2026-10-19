package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Tipos de campo suportados.
const (
	FieldText     = "text"
	FieldNumber   = "number"
	FieldSelect   = "select"
	FieldCheckbox = "checkbox"
)

// Estados do formulário.
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

type Form struct {
	ID          int64
	UserID      int64
	Title       string
	Description string
	Status      string
	Slug        string // vazio enquanto draft
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Fields      []Field
}

type Field struct {
	ID       int64
	Position int
	Label    string
	Type     string
	Required bool
	Options  []string
}

// ---- leitura ---------------------------------------------------------

// ListForms devolve os formulários do usuário, sem os campos (a lista da
// tela de admin não precisa deles).
func (s *Store) ListForms(ctx context.Context, userID int64) ([]Form, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, user_id, title, description, status, COALESCE(slug, ''),
		        published_at, created_at, updated_at
		 FROM forms WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	forms := []Form{}
	for rows.Next() {
		var f Form
		if err := rows.Scan(&f.ID, &f.UserID, &f.Title, &f.Description, &f.Status,
			&f.Slug, &f.PublishedAt, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		f.Fields = []Field{}
		forms = append(forms, f)
	}
	return forms, rows.Err()
}

// FormByID busca um formulário do usuário, com os campos.
func (s *Store) FormByID(ctx context.Context, userID, formID int64) (Form, error) {
	var f Form
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, user_id, title, description, status, COALESCE(slug, ''),
		        published_at, created_at, updated_at
		 FROM forms WHERE id = $1 AND user_id = $2`, formID, userID,
	).Scan(&f.ID, &f.UserID, &f.Title, &f.Description, &f.Status, &f.Slug,
		&f.PublishedAt, &f.CreatedAt, &f.UpdatedAt)

	if errors.Is(err, errNoRows) {
		return Form{}, ErrNotFound
	}
	if err != nil {
		return Form{}, err
	}

	f.Fields, err = s.fieldsOf(ctx, s.DB, f.ID)
	return f, err
}

// FormBySlug é a leitura pública: só encontra formulários publicados e não
// exige usuário.
func (s *Store) FormBySlug(ctx context.Context, slug string) (Form, error) {
	var f Form
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, user_id, title, description, status, COALESCE(slug, ''),
		        published_at, created_at, updated_at
		 FROM forms WHERE slug = $1 AND status = 'published'`, slug,
	).Scan(&f.ID, &f.UserID, &f.Title, &f.Description, &f.Status, &f.Slug,
		&f.PublishedAt, &f.CreatedAt, &f.UpdatedAt)

	if errors.Is(err, errNoRows) {
		return Form{}, ErrNotFound
	}
	if err != nil {
		return Form{}, err
	}

	f.Fields, err = s.fieldsOf(ctx, s.DB, f.ID)
	return f, err
}

// queryer aceita tanto *sql.DB quanto *sql.Tx.
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (s *Store) fieldsOf(ctx context.Context, q queryer, formID int64) ([]Field, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, position, label, type, required, options
		 FROM form_fields WHERE form_id = $1 ORDER BY position`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fields := []Field{}
	for rows.Next() {
		var f Field
		var options []byte
		if err := rows.Scan(&f.ID, &f.Position, &f.Label, &f.Type, &f.Required, &options); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(options, &f.Options); err != nil {
			return nil, fmt.Errorf("lendo options do campo %d: %w", f.ID, err)
		}
		if f.Options == nil {
			f.Options = []string{}
		}
		fields = append(fields, f)
	}
	return fields, rows.Err()
}

// ---- escrita ---------------------------------------------------------

// CreateForm cria o formulário como draft e grava os campos.
func (s *Store) CreateForm(ctx context.Context, userID int64, title, description string, fields []Field) (Form, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Form{}, err
	}
	defer tx.Rollback() // no-op depois do Commit

	var formID int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO forms (user_id, title, description)
		 VALUES ($1, $2, $3) RETURNING id`,
		userID, title, description,
	).Scan(&formID); err != nil {
		return Form{}, err
	}

	if err := insertFields(ctx, tx, formID, fields); err != nil {
		return Form{}, err
	}
	if err := tx.Commit(); err != nil {
		return Form{}, err
	}
	return s.FormByID(ctx, userID, formID)
}

// UpdateForm altera título/descrição e substitui a lista de campos.
// Substituir (apagar e reinserir) é mais simples que fazer diff e é o que
// a tela de edição precisa: ela sempre manda a definição completa.
func (s *Store) UpdateForm(ctx context.Context, userID, formID int64, title, description string, fields []Field) (Form, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Form{}, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`UPDATE forms SET title = $1, description = $2, updated_at = now()
		 WHERE id = $3 AND user_id = $4`,
		title, description, formID, userID)
	if err != nil {
		return Form{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Form{}, ErrNotFound
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM form_fields WHERE form_id = $1`, formID); err != nil {
		return Form{}, err
	}
	if err := insertFields(ctx, tx, formID, fields); err != nil {
		return Form{}, err
	}
	if err := tx.Commit(); err != nil {
		return Form{}, err
	}
	return s.FormByID(ctx, userID, formID)
}

func insertFields(ctx context.Context, tx *sql.Tx, formID int64, fields []Field) error {
	for i, f := range fields {
		options := f.Options
		if options == nil {
			options = []string{}
		}
		raw, err := json.Marshal(options)
		if err != nil {
			return err
		}
		// A posição é derivada da ordem recebida, não do cliente.
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO form_fields (form_id, position, label, type, required, options)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			formID, i, f.Label, f.Type, f.Required, raw); err != nil {
			return err
		}
	}
	return nil
}

// SetStatus publica (gerando slug) ou volta para draft (limpando o slug).
// O slug é gerado aqui dentro e, em caso de colisão, tentamos de novo.
func (s *Store) SetStatus(ctx context.Context, userID, formID int64, status string) (Form, error) {
	if status == StatusDraft {
		res, err := s.DB.ExecContext(ctx,
			`UPDATE forms SET status = 'draft', slug = NULL, published_at = NULL,
			        updated_at = now()
			 WHERE id = $1 AND user_id = $2`, formID, userID)
		if err != nil {
			return Form{}, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return Form{}, ErrNotFound
		}
		return s.FormByID(ctx, userID, formID)
	}

	current, err := s.FormByID(ctx, userID, formID)
	if err != nil {
		return Form{}, err
	}
	if len(current.Fields) == 0 {
		return Form{}, fmt.Errorf("%w: formulário sem campos", ErrInvalid)
	}
	// Já publicado: mantém o slug para não quebrar links existentes.
	if current.Status == StatusPublished {
		return current, nil
	}

	for attempt := 0; attempt < 5; attempt++ {
		slug := NewSlug(current.Title)
		res, err := s.DB.ExecContext(ctx,
			`UPDATE forms SET status = 'published', slug = $1,
			        published_at = now(), updated_at = now()
			 WHERE id = $2 AND user_id = $3`, slug, formID, userID)
		if err != nil {
			if isUniqueViolation(err) {
				continue // sorteia outro sufixo
			}
			return Form{}, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return Form{}, ErrNotFound
		}
		return s.FormByID(ctx, userID, formID)
	}
	return Form{}, errors.New("não foi possível gerar um slug único")
}

// DeleteForm remove o formulário; campos e respostas caem por cascade.
func (s *Store) DeleteForm(ctx context.Context, userID, formID int64) error {
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM forms WHERE id = $1 AND user_id = $2`, formID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
