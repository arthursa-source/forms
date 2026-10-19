package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/arthursa-source/forms/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

// ---- formatos ---------------------------------------------------------

// fieldBody é a definição de um campo, igual na entrada e na saída.
// A tag `enum` faz o huma rejeitar tipos desconhecidos e documentar a
// lista de valores na spec OpenAPI.
type fieldBody struct {
	ID       int64    `json:"id,omitempty" readOnly:"true"`
	Label    string   `json:"label" minLength:"1" maxLength:"200"`
	Type     string   `json:"type" enum:"text,number,select,checkbox"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty" nullable:"false" doc:"obrigatório (e só usado) quando type=select"`
}

type formBody struct {
	ID          int64       `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      string      `json:"status" enum:"draft,published"`
	Slug        string      `json:"slug,omitempty" doc:"preenchido quando publicado"`
	PublicPath  string      `json:"publicPath,omitempty" doc:"caminho público, ex: /f/meu-form-ab12cd34"`
	Fields      []fieldBody `json:"fields" nullable:"false"`
	PublishedAt *time.Time  `json:"publishedAt,omitempty"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

type formInput struct {
	Body struct {
		Title       string      `json:"title" minLength:"1" maxLength:"200"`
		Description string      `json:"description,omitempty" maxLength:"2000"`
		Fields      []fieldBody `json:"fields" nullable:"false" doc:"definição completa dos campos, na ordem de exibição"`
	}
}

type formIDInput struct {
	FormID int64 `path:"formId"`
}

type formUpdateInput struct {
	FormID int64 `path:"formId"`
	Body   struct {
		Title       string      `json:"title" minLength:"1" maxLength:"200"`
		Description string      `json:"description,omitempty" maxLength:"2000"`
		Fields      []fieldBody `json:"fields" nullable:"false"`
	}
}

type formOutput struct {
	Body formBody
}

type formListOutput struct {
	Body struct {
		Forms []formBody `json:"forms" nullable:"false"`
	}
}

// ---- conversões -------------------------------------------------------

func toFormBody(f store.Form) formBody {
	out := formBody{
		ID:          f.ID,
		Title:       f.Title,
		Description: f.Description,
		Status:      f.Status,
		Slug:        f.Slug,
		Fields:      make([]fieldBody, 0, len(f.Fields)),
		PublishedAt: f.PublishedAt,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
	if f.Slug != "" {
		out.PublicPath = "/f/" + f.Slug
	}
	for _, field := range f.Fields {
		out.Fields = append(out.Fields, fieldBody{
			ID:       field.ID,
			Label:    field.Label,
			Type:     field.Type,
			Required: field.Required,
			Options:  field.Options,
		})
	}
	return out
}

// toStoreFields valida a definição e converte para o formato do store.
// O huma já garantiu tipos e tamanhos; aqui ficam as regras que dependem
// da combinação dos valores.
func toStoreFields(in []fieldBody) ([]store.Field, error) {
	fields := make([]store.Field, 0, len(in))
	for i, f := range in {
		if f.Type == store.FieldSelect {
			if len(f.Options) == 0 {
				return nil, huma.Error422UnprocessableEntity(
					fmt.Sprintf("campo %d (%q): type=select exige pelo menos uma opção", i+1, f.Label))
			}
			seen := map[string]bool{}
			for _, opt := range f.Options {
				if opt == "" {
					return nil, huma.Error422UnprocessableEntity(
						fmt.Sprintf("campo %d (%q): opção vazia não é permitida", i+1, f.Label))
				}
				if seen[opt] {
					return nil, huma.Error422UnprocessableEntity(
						fmt.Sprintf("campo %d (%q): opção %q repetida", i+1, f.Label, opt))
				}
				seen[opt] = true
			}
		} else if len(f.Options) > 0 {
			return nil, huma.Error422UnprocessableEntity(
				fmt.Sprintf("campo %d (%q): options só vale para type=select", i+1, f.Label))
		}

		fields = append(fields, store.Field{
			Position: i,
			Label:    f.Label,
			Type:     f.Type,
			Required: f.Required,
			Options:  f.Options,
		})
	}
	return fields, nil
}

// ---- registro ---------------------------------------------------------

func (s *Server) registerForms(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "createForm",
		Method:        http.MethodPost,
		Path:          "/forms",
		Summary:       "Cria um formulário (nasce como draft)",
		Tags:          []string{"forms"},
		Security:      protected(),
		DefaultStatus: http.StatusCreated,
	}, s.handleCreateForm)

	huma.Register(api, huma.Operation{
		OperationID: "listForms",
		Method:      http.MethodGet,
		Path:        "/forms",
		Summary:     "Lista os formulários do usuário",
		Tags:        []string{"forms"},
		Security:    protected(),
	}, s.handleListForms)

	huma.Register(api, huma.Operation{
		OperationID: "getForm",
		Method:      http.MethodGet,
		Path:        "/forms/{formId}",
		Summary:     "Detalha um formulário com seus campos",
		Tags:        []string{"forms"},
		Security:    protected(),
	}, s.handleGetForm)

	huma.Register(api, huma.Operation{
		OperationID: "updateForm",
		Method:      http.MethodPut,
		Path:        "/forms/{formId}",
		Summary:     "Atualiza título, descrição e campos",
		Tags:        []string{"forms"},
		Security:    protected(),
	}, s.handleUpdateForm)

	huma.Register(api, huma.Operation{
		OperationID: "publishForm",
		Method:      http.MethodPost,
		Path:        "/forms/{formId}/publish",
		Summary:     "Publica o formulário e gera a URL pública",
		Tags:        []string{"forms"},
		Security:    protected(),
	}, s.handlePublishForm)

	huma.Register(api, huma.Operation{
		OperationID: "unpublishForm",
		Method:      http.MethodPost,
		Path:        "/forms/{formId}/unpublish",
		Summary:     "Volta o formulário para draft (desativa a URL pública)",
		Tags:        []string{"forms"},
		Security:    protected(),
	}, s.handleUnpublishForm)

	huma.Register(api, huma.Operation{
		OperationID:   "deleteForm",
		Method:        http.MethodDelete,
		Path:          "/forms/{formId}",
		Summary:       "Remove o formulário e suas respostas",
		Tags:          []string{"forms"},
		Security:      protected(),
		DefaultStatus: http.StatusNoContent,
	}, s.handleDeleteForm)
}

// ---- handlers ---------------------------------------------------------

func (s *Server) handleCreateForm(ctx context.Context, in *formInput) (*formOutput, error) {
	fields, err := toStoreFields(in.Body.Fields)
	if err != nil {
		return nil, err
	}

	form, err := s.Store.CreateForm(ctx, userIDFrom(ctx), in.Body.Title, in.Body.Description, fields)
	if err != nil {
		return nil, err
	}
	return &formOutput{Body: toFormBody(form)}, nil
}

func (s *Server) handleListForms(ctx context.Context, _ *struct{}) (*formListOutput, error) {
	forms, err := s.Store.ListForms(ctx, userIDFrom(ctx))
	if err != nil {
		return nil, err
	}

	out := &formListOutput{}
	out.Body.Forms = make([]formBody, 0, len(forms))
	for _, f := range forms {
		out.Body.Forms = append(out.Body.Forms, toFormBody(f))
	}
	return out, nil
}

func (s *Server) handleGetForm(ctx context.Context, in *formIDInput) (*formOutput, error) {
	form, err := s.Store.FormByID(ctx, userIDFrom(ctx), in.FormID)
	if err != nil {
		return nil, formError(err)
	}
	return &formOutput{Body: toFormBody(form)}, nil
}

func (s *Server) handleUpdateForm(ctx context.Context, in *formUpdateInput) (*formOutput, error) {
	fields, err := toStoreFields(in.Body.Fields)
	if err != nil {
		return nil, err
	}

	form, err := s.Store.UpdateForm(ctx, userIDFrom(ctx), in.FormID,
		in.Body.Title, in.Body.Description, fields)
	if err != nil {
		return nil, formError(err)
	}
	return &formOutput{Body: toFormBody(form)}, nil
}

func (s *Server) handlePublishForm(ctx context.Context, in *formIDInput) (*formOutput, error) {
	form, err := s.Store.SetStatus(ctx, userIDFrom(ctx), in.FormID, store.StatusPublished)
	if err != nil {
		return nil, formError(err)
	}
	return &formOutput{Body: toFormBody(form)}, nil
}

func (s *Server) handleUnpublishForm(ctx context.Context, in *formIDInput) (*formOutput, error) {
	form, err := s.Store.SetStatus(ctx, userIDFrom(ctx), in.FormID, store.StatusDraft)
	if err != nil {
		return nil, formError(err)
	}
	return &formOutput{Body: toFormBody(form)}, nil
}

func (s *Server) handleDeleteForm(ctx context.Context, in *formIDInput) (*struct{}, error) {
	if err := s.Store.DeleteForm(ctx, userIDFrom(ctx), in.FormID); err != nil {
		return nil, formError(err)
	}
	return nil, nil
}

// formError traduz os erros de domínio do store em respostas HTTP.
// Um formulário de outro usuário devolve 404, não 403: assim a API não
// confirma a existência de ids alheios.
func formError(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return huma.Error404NotFound("formulário não encontrado")
	case errors.Is(err, store.ErrInvalid):
		return huma.Error422UnprocessableEntity(err.Error())
	default:
		return err
	}
}
