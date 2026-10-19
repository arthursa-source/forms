package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/arthursa-source/forms/internal/store"
	"github.com/arthursa-source/forms/internal/submission"
	"github.com/danielgtaylor/huma/v2"
)

// ---- formatos ---------------------------------------------------------

// publicFormBody é a visão pública: não expõe id interno, dono nem datas
// de edição. Os ids dos campos aparecem porque a submissão os usa como
// chave.
type publicFormBody struct {
	Slug        string            `json:"slug"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Fields      []publicFieldBody `json:"fields"`
}

type publicFieldBody struct {
	ID       int64    `json:"id"`
	Label    string   `json:"label"`
	Type     string   `json:"type" enum:"text,number,select,checkbox"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}

type publicFormInput struct {
	Slug string `path:"slug" maxLength:"120"`
}

type publicFormOutput struct {
	Body publicFormBody
}

type submitInput struct {
	Slug string `path:"slug" maxLength:"120"`
	Body struct {
		// Chave = id do campo (como texto, que é o que o JSON permite).
		// O valor aceita texto, número ou booleano, conforme o tipo.
		Answers map[string]any `json:"answers" doc:"mapa de id do campo para o valor respondido"`
	}
}

type submitOutput struct {
	Body struct {
		ID        int64     `json:"id"`
		CreatedAt time.Time `json:"createdAt"`
		Message   string    `json:"message"`
	}
}

// ---- registro ---------------------------------------------------------

func (s *Server) registerPublic(api huma.API) {
	// Sem Security: o middleware de auth deixa passar.
	huma.Register(api, huma.Operation{
		OperationID: "getPublicForm",
		Method:      http.MethodGet,
		Path:        "/public/forms/{slug}",
		Summary:     "Abre um formulário publicado (sem autenticação)",
		Tags:        []string{"public"},
	}, s.handlePublicForm)

	huma.Register(api, huma.Operation{
		OperationID:   "submitPublicForm",
		Method:        http.MethodPost,
		Path:          "/public/forms/{slug}/responses",
		Summary:       "Envia uma resposta (sem autenticação)",
		Description:   "O backend valida os valores contra a definição do formulário antes de persistir.",
		Tags:          []string{"public"},
		DefaultStatus: http.StatusCreated,
	}, s.handleSubmit)
}

// ---- handlers ---------------------------------------------------------

func (s *Server) handlePublicForm(ctx context.Context, in *publicFormInput) (*publicFormOutput, error) {
	form, err := s.Store.FormBySlug(ctx, in.Slug)
	if err != nil {
		return nil, publicFormError(err)
	}

	out := &publicFormOutput{Body: publicFormBody{
		Slug:        form.Slug,
		Title:       form.Title,
		Description: form.Description,
		Fields:      make([]publicFieldBody, 0, len(form.Fields)),
	}}
	for _, f := range form.Fields {
		out.Body.Fields = append(out.Body.Fields, publicFieldBody{
			ID: f.ID, Label: f.Label, Type: f.Type, Required: f.Required, Options: f.Options,
		})
	}
	return out, nil
}

func (s *Server) handleSubmit(ctx context.Context, in *submitInput) (*submitOutput, error) {
	// Busca por slug só encontra formulários publicados: um draft nunca
	// aceita resposta.
	form, err := s.Store.FormBySlug(ctx, in.Slug)
	if err != nil {
		return nil, publicFormError(err)
	}

	answers, fieldErrs := submission.Validate(form.Fields, in.Body.Answers)
	if len(fieldErrs) > 0 {
		// Devolve todos os problemas de uma vez, com o caminho do campo,
		// para o frontend marcar os inputs errados.
		details := make([]error, 0, len(fieldErrs))
		for _, fe := range fieldErrs {
			details = append(details, &huma.ErrorDetail{
				Message:  fe.Message,
				Location: "body.answers." + fieldKey(fe.FieldID),
			})
		}
		return nil, huma.Error422UnprocessableEntity("resposta inválida", details...)
	}

	saved, err := s.Store.CreateResponse(ctx, form.ID, answers)
	if err != nil {
		return nil, err
	}

	out := &submitOutput{}
	out.Body.ID = saved.ID
	out.Body.CreatedAt = saved.CreatedAt
	out.Body.Message = "resposta registrada"
	return out, nil
}

// fieldKey monta o caminho do campo no erro. Id 0 significa que a chave
// enviada não era de nenhum campo deste formulário.
func fieldKey(id int64) string {
	if id == 0 {
		return "desconhecido"
	}
	return strconv.FormatInt(id, 10)
}

func publicFormError(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		// Mesmo 404 para "não existe" e "existe mas é draft": a página
		// pública não revela a diferença.
		return huma.Error404NotFound("formulário não encontrado ou não publicado")
	}
	return err
}
