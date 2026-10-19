package httpapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// No huma, a resposta é uma struct com um campo Body: é dele que sai o
// JSON e também o schema na spec OpenAPI.
type healthOutput struct {
	Body struct {
		Status   string `json:"status" example:"ok" doc:"ok quando a API está de pé"`
		Database string `json:"database" example:"ok" doc:"ok se o ping no Postgres respondeu"`
	}
}

func (s *Server) registerHealth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check",
		Tags:        []string{"system"},
	}, func(ctx context.Context, _ *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.Status = "ok"
		out.Body.Database = "ok"
		if s.Store == nil || s.Store.DB == nil {
			out.Body.Database = "não configurado"
		} else if err := s.Store.DB.PingContext(ctx); err != nil {
			out.Body.Database = "indisponível"
		}
		return out, nil
	})
}
