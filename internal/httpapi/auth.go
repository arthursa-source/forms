package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/arthursa-source/forms/internal/auth"
	"github.com/arthursa-source/forms/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

// ---- formatos de entrada e saída -------------------------------------
//
// As tags `minLength`, `format`, `required` etc. são lidas pelo huma: ele
// valida a requisição E descreve isso na spec OpenAPI.

type credentialsInput struct {
	Body struct {
		Email    string `json:"email" format:"email" doc:"e-mail do admin"`
		Password string `json:"password" minLength:"8" maxLength:"72" doc:"senha (mín. 8 caracteres)"`
	}
}

type sessionOutput struct {
	Body struct {
		Token     string    `json:"token" doc:"JWT a enviar em Authorization: Bearer <token>"`
		ExpiresAt time.Time `json:"expiresAt"`
		User      userBody  `json:"user"`
	}
}

type userOutput struct {
	Body userBody
}

type userBody struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

func toUserBody(u store.User) userBody {
	return userBody{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt}
}

// ---- registro das operações ------------------------------------------

func (s *Server) registerAuth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "register",
		Method:        http.MethodPost,
		Path:          "/auth/register",
		Summary:       "Cria uma conta de admin",
		Tags:          []string{"auth"},
		DefaultStatus: http.StatusCreated,
	}, s.handleRegister)

	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/auth/login",
		Summary:     "Autentica e devolve um JWT",
		Tags:        []string{"auth"},
	}, s.handleLogin)

	huma.Register(api, huma.Operation{
		OperationID: "me",
		Method:      http.MethodGet,
		Path:        "/auth/me",
		Summary:     "Dados do usuário autenticado",
		Tags:        []string{"auth"},
		Security:    protected(), // exige o Bearer token
	}, s.handleMe)
}

func (s *Server) handleRegister(ctx context.Context, in *credentialsInput) (*sessionOutput, error) {
	hash, err := auth.HashPassword(in.Body.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.Store.CreateUser(ctx, in.Body.Email, hash)
	if errors.Is(err, store.ErrConflict) {
		return nil, huma.Error409Conflict("este e-mail já está cadastrado")
	}
	if err != nil {
		return nil, err
	}

	return s.newSession(user)
}

func (s *Server) handleLogin(ctx context.Context, in *credentialsInput) (*sessionOutput, error) {
	// Mensagem genérica de propósito: não revela se o e-mail existe.
	invalid := huma.Error401Unauthorized("e-mail ou senha inválidos")

	user, err := s.Store.UserByEmail(ctx, in.Body.Email)
	if errors.Is(err, store.ErrNotFound) {
		return nil, invalid
	}
	if err != nil {
		return nil, err
	}
	if !auth.CheckPassword(user.PasswordHash, in.Body.Password) {
		return nil, invalid
	}

	return s.newSession(user)
}

func (s *Server) handleMe(ctx context.Context, _ *struct{}) (*userOutput, error) {
	user, err := s.Store.UserByID(ctx, userIDFrom(ctx))
	if errors.Is(err, store.ErrNotFound) {
		return nil, huma.Error401Unauthorized("usuário não existe mais")
	}
	if err != nil {
		return nil, err
	}
	return &userOutput{Body: toUserBody(user)}, nil
}

func (s *Server) newSession(user store.User) (*sessionOutput, error) {
	token, expiresAt, err := auth.IssueToken(s.JWTSecret, user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	out := &sessionOutput{}
	out.Body.Token = token
	out.Body.ExpiresAt = expiresAt
	out.Body.User = toUserBody(user)
	return out, nil
}
