package httpapi

import (
	"context"
	"strings"

	"github.com/arthursa-source/forms/internal/auth"
	"github.com/danielgtaylor/huma/v2"
)

// Chave privada para guardar o id do usuário no contexto. Um tipo próprio
// evita colisão com chaves de outros pacotes.
type ctxKey string

const userIDKey ctxKey = "userID"

// authMiddleware roda antes de cada handler. Ele só exige token nas
// operações marcadas com Security — assim as rotas públicas (/f/{slug},
// login, health) seguem abertas, e a lista de rotas protegidas é a mesma
// coisa que a spec OpenAPI publica.
func (s *Server) authMiddleware(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if len(ctx.Operation().Security) == 0 {
			next(ctx)
			return
		}

		header := ctx.Header("Authorization")
		raw, ok := bearerToken(header)
		if !ok {
			_ = huma.WriteErr(api, ctx, 401, "informe o header Authorization: Bearer <token>")
			return
		}

		userID, err := auth.ParseToken(s.JWTSecret, raw)
		if err != nil {
			_ = huma.WriteErr(api, ctx, 401, "token inválido ou expirado")
			return
		}

		// Passa o id adiante no contexto para os handlers lerem.
		next(huma.WithValue(ctx, userIDKey, userID))
	}
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}

// userIDFrom lê o id injetado pelo middleware. Só é chamado em handlers
// de operações protegidas, então a presença é garantida.
func userIDFrom(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}

// protected é um atalho para marcar a operação como autenticada.
func protected() []map[string][]string {
	return []map[string][]string{{"bearerAuth": {}}}
}
