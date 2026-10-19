# Form Builder

Criação e publicação de formulários. Backend em Go (chi + huma, OpenAPI gerado
do código) e frontend React + TypeScript + Vite com client gerado da spec.

> Status: em construção por etapas. Ver "Limitações e decisões por falta de
> tempo" no fim deste arquivo.

## Pré-requisitos

- Go 1.26+
- Docker + Docker Compose (para o Postgres)
- Node 20+ (frontend)
- make (opcional, mas os comandos abaixo assumem que existe)

## Subindo o backend do zero

```bash
cp .env.example .env     # opcional: os padrões já apontam para o docker-compose
make db-up               # sobe o Postgres e espera o healthcheck
make migrate             # aplica as migrations
make run                 # servidor em http://localhost:8080
```

Verificando:

```bash
curl http://localhost:8080/health
# {"status":"ok","database":"ok"}
```

## Configuração

Três opções, nesta ordem de precedência (a última vence):

1. padrões embutidos no código
2. variáveis de ambiente
3. flags de linha de comando

| Flag             | Variável de ambiente | Padrão                                                            |
| ---------------- | -------------------- | ----------------------------------------------------------------- |
| `--address`      | `ADDRESS`            | `localhost:8080`                                                  |
| `--database-url` | `DATABASE_URL`       | `postgres://forms:forms@localhost:5432/forms?sslmode=disable`     |
| `--jwt-secret`   | `JWT_SECRET`         | `dev-secret-troque-isto`                                          |

Flags e env podem ser combinados:

```bash
DATABASE_URL=postgres://... go run ./cmd/server run --address=0.0.0.0:9000
```

## Banco e migrations

O `docker-compose.yml` sobe um Postgres 16 com usuário/senha/banco `forms`
na porta 5432, com volume persistente.

As migrations são SQL versionado em `migrations/`, aplicadas pelo
[goose](https://github.com/pressly/goose) usado como biblioteca. Os arquivos
são embutidos no binário com `go:embed`, então o comando funciona de qualquer
diretório:

```bash
make migrate          # ou: go run ./cmd/server migrate up
make migrate-status   # ou: go run ./cmd/server migrate status
go run ./cmd/server migrate down   # desfaz a última
```

Para criar uma nova migration, adicione `migrations/0000N_nome.sql` com os
marcadores `-- +goose Up` e `-- +goose Down`.

## Estrutura

```
cmd/server/          comando `server` (subcomandos run e migrate)
internal/config/     resolução de flags + env
internal/database/   conexão com Postgres e execução das migrations
internal/httpapi/    roteador chi + operações huma
migrations/          SQL versionado (embutido no binário)
api/                 spec OpenAPI gerada
web/                 frontend React + Vite
```

## Decisões de arquitetura

- **chi + huma**: o huma deriva a spec OpenAPI 3 das structs Go de
  request/response, o que atende ao requisito de spec gerada do código sem
  manter YAML à mão. O chi fica como roteador e dá acesso aos middlewares
  padrão de `net/http`.
- **database/sql + driver pgx (stdlib)**: um único `*sql.DB` serve tanto para
  as queries quanto para o goose. SQL escrito à mão, sem ORM — o modelo é
  pequeno e fica mais fácil de ler.
- **Migrations embutidas no binário**: evita depender do diretório de trabalho
  e dispensa instalar o CLI do goose.

## Limitações e decisões por falta de tempo

_A ser completado nas próximas etapas._
