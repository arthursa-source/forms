// Package httpapi monta o roteador HTTP e registra as operações.
//
// Usamos chi (roteador) + huma v2 (camada de operações). O huma valida
// request/response a partir das structs Go e, com isso, GERA a spec
// OpenAPI 3 automaticamente — nada de escrever YAML à mão.
package httpapi

import (
	"net/http"

	"github.com/arthursa-source/forms/internal/store"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server carrega as dependências compartilhadas pelos handlers.
type Server struct {
	Store     *store.Store
	JWTSecret string
}

// NewRouter devolve o roteador pronto para servir e a instância huma
// (usada também para imprimir a spec OpenAPI).
func NewRouter(s *Server) (chi.Router, huma.API) {
	r := chi.NewMux()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	cfg := huma.DefaultConfig("Form Builder API", "1.0.0")
	// Declarar o esquema aqui faz a spec OpenAPI já sair documentando o
	// header Authorization: Bearer <token>.
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearerAuth": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}

	api := humachi.New(r, cfg)
	api.UseMiddleware(s.authMiddleware(api))

	s.registerHealth(api)
	s.registerAuth(api)
	s.registerForms(api)
	s.registerResponses(api)
	s.registerPublic(api)

	return r, api
}

// corsMiddleware libera o frontend do Vite (porta 5173) a chamar a API.
// Em produção isso viraria uma lista de origens configurável.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
