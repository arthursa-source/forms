package store

import (
	"context"
	"time"
)

// Answer é uma resposta já validada e normalizada para texto.
type Answer struct {
	Position int
	Label    string
	Type     string
	Value    string
}

type Response struct {
	ID        int64
	FormID    int64
	CreatedAt time.Time
	Answers   []Answer
}

// CreateResponse grava a resposta e seus valores numa transação.
func (s *Store) CreateResponse(ctx context.Context, formID int64, answers []Answer) (Response, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Response{}, err
	}
	defer tx.Rollback()

	r := Response{FormID: formID, Answers: answers}
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO responses (form_id) VALUES ($1) RETURNING id, created_at`,
		formID).Scan(&r.ID, &r.CreatedAt); err != nil {
		return Response{}, err
	}

	for _, a := range answers {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO response_answers (response_id, position, label, type, value)
			 VALUES ($1, $2, $3, $4, $5)`,
			r.ID, a.Position, a.Label, a.Type, a.Value); err != nil {
			return Response{}, err
		}
	}

	return r, tx.Commit()
}

// ListResponses devolve as respostas de um formulário do usuário, mais
// recentes primeiro. Confere a posse antes de ler.
func (s *Store) ListResponses(ctx context.Context, userID, formID int64) ([]Response, error) {
	// Garante que o formulário é do usuário (senão, ErrNotFound).
	if _, err := s.FormByID(ctx, userID, formID); err != nil {
		return nil, err
	}

	// Um único SELECT com JOIN: as respostas vêm ordenadas e agrupamos em Go.
	rows, err := s.DB.QueryContext(ctx,
		`SELECT r.id, r.created_at,
		        COALESCE(a.position, 0), COALESCE(a.label, ''),
		        COALESCE(a.type, ''), COALESCE(a.value, '')
		 FROM responses r
		 LEFT JOIN response_answers a ON a.response_id = r.id
		 WHERE r.form_id = $1
		 ORDER BY r.created_at DESC, r.id DESC, a.position`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	responses := []Response{}
	for rows.Next() {
		var (
			id        int64
			createdAt time.Time
			a         Answer
		)
		if err := rows.Scan(&id, &createdAt, &a.Position, &a.Label, &a.Type, &a.Value); err != nil {
			return nil, err
		}
		// As linhas da mesma resposta vêm juntas, então basta comparar
		// com a última adicionada.
		if len(responses) == 0 || responses[len(responses)-1].ID != id {
			responses = append(responses, Response{
				ID: id, FormID: formID, CreatedAt: createdAt, Answers: []Answer{},
			})
		}
		if a.Label != "" {
			last := &responses[len(responses)-1]
			last.Answers = append(last.Answers, a)
		}
	}
	return responses, rows.Err()
}
