package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// Pequenos helpers usados também em public.go.
func formatInt(v int64) string        { return strconv.FormatInt(v, 10) }
func errorsIs(err, target error) bool { return errors.Is(err, target) }

type answerBody struct {
	Label string `json:"label"`
	Type  string `json:"type"`
	Value string `json:"value" doc:"valor normalizado como texto (number: \"42\", checkbox: \"true\")"`
}

type responseBody struct {
	ID        int64        `json:"id"`
	CreatedAt time.Time    `json:"createdAt"`
	Answers   []answerBody `json:"answers" nullable:"false"`
}

type responseListOutput struct {
	Body struct {
		FormID    int64          `json:"formId"`
		Total     int            `json:"total"`
		Responses []responseBody `json:"responses" nullable:"false"`
	}
}

func (s *Server) registerResponses(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "listResponses",
		Method:      http.MethodGet,
		Path:        "/forms/{formId}/responses",
		Summary:     "Lista as respostas recebidas por um formulário",
		Tags:        []string{"forms"},
		Security:    protected(),
	}, s.handleListResponses)
}

func (s *Server) handleListResponses(ctx context.Context, in *formIDInput) (*responseListOutput, error) {
	responses, err := s.Store.ListResponses(ctx, userIDFrom(ctx), in.FormID)
	if err != nil {
		return nil, formError(err)
	}

	out := &responseListOutput{}
	out.Body.FormID = in.FormID
	out.Body.Total = len(responses)
	out.Body.Responses = make([]responseBody, 0, len(responses))
	for _, r := range responses {
		rb := responseBody{ID: r.ID, CreatedAt: r.CreatedAt, Answers: make([]answerBody, 0, len(r.Answers))}
		for _, a := range r.Answers {
			rb.Answers = append(rb.Answers, answerBody{Label: a.Label, Type: a.Type, Value: a.Value})
		}
		out.Body.Responses = append(out.Body.Responses, rb)
	}
	return out, nil
}
