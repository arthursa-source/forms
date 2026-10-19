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

// configOutput diz ao frontend o que este servidor tem ligado. Hoje só o
// login com Google, que depende de credenciais: sem isso o frontend
// esconde o botão em vez de mostrar um caminho que daria 503.
type configOutput struct {
	Body struct {
		GoogleAuth bool `json:"googleAuth" doc:"true quando o login com Google está configurado"`
	}
}

func (s *Server) registerConfig(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "config",
		Method:      http.MethodGet,
		Path:        "/config",
		Summary:     "Configuração pública do servidor",
		Description: "Não expõe segredo nenhum: só diz quais formas de login estão disponíveis.",
		Tags:        []string{"system"},
	}, func(_ context.Context, _ *struct{}) (*configOutput, error) {
		out := &configOutput{}
		out.Body.GoogleAuth = s.googleEnabled()
		return out, nil
	})
}
