# Form Builder

Criação e publicação de formulários, com backend em Go e frontend React.

- **Backend**: Go, chi + [huma](https://huma.rocks) (a spec OpenAPI 3 é gerada
  do código), Postgres, migrations com goose, auth por e-mail/senha com
  bcrypt, login com Google (OAuth2 + PKCE) e sessão JWT.
- **Frontend**: React + TypeScript + Vite, shadcn/ui, TanStack Router e
  TanStack Query. O código que fala com a API é **gerado** da spec OpenAPI
  (openapi-typescript + openapi-fetch), não escrito à mão.
- Toda comunicação do frontend passa pelo backend Go.

## Índice

- [Pré-requisitos](#pré-requisitos)
- [Subindo tudo do zero](#subindo-tudo-do-zero)
  - [Como executar o backend](#como-executar-o-backend)
  - [Como executar o frontend](#como-executar-o-frontend)
- [Configuração](#configuração)
  - [Login com Google](#login-com-google)
- [Banco de dados e migrations](#banco-de-dados-e-migrations)
- [Gerando a spec OpenAPI](#gerando-a-spec-openapi)
- [Gerando o client TypeScript](#gerando-o-client-typescript)
- [Comandos do Makefile](#comandos-do-makefile)
- [Testes](#testes)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Decisões de arquitetura](#decisões-de-arquitetura)
- [Limitações e decisões por falta de tempo](#limitações-e-decisões-por-falta-de-tempo)

## Pré-requisitos

| Ferramenta             | Versão testada | Para quê               |
| ---------------------- | -------------- | ---------------------- |
| Go                     | 1.26           | backend                |
| Docker + Docker Compose| 29 / v5        | Postgres local         |
| Node + npm             | 26 / 11        | frontend               |
| make                   | qualquer       | atalhos dos comandos   |

Multiplataforma: no Windows, rode os comandos em Git Bash ou WSL (o
Makefile só chama `docker`, `go` e `npm`).

Sem `make`? Cada alvo é um comando só — veja
[Comandos do Makefile](#comandos-do-makefile).

## Subindo tudo do zero

Quatro passos. Os dois primeiros num terminal, o terceiro e o quarto em
terminais separados (backend e frontend ficam rodando).

```bash
# 1. banco de dados (sobe o Postgres e espera o healthcheck passar)
make db-up

# 2. migrations
make migrate

# 3. backend — deixe rodando
make run
# → servidor ouvindo em http://localhost:8080

# 4. frontend — em outro terminal, deixe rodando
cd web && npm install && npm run dev
# → http://localhost:5173
```

### Como executar o backend

```bash
make db-up      # Postgres via docker-compose
make migrate    # aplica as migrations
make run        # go run ./cmd/server run --address=localhost:8080
```

Conferindo:

```bash
curl http://localhost:8080/health
# {"status":"ok","database":"ok"}
```

A documentação interativa da API fica em <http://localhost:8080/docs>.

### Como executar o frontend

```bash
cd web
npm install
npm run dev     # → http://localhost:5173
```

`npm run dev` roda `generate:client` antes de subir (script `predev`), então
o client TypeScript é gerado automaticamente a partir de `api/openapi.yaml`.
O backend precisa estar no ar: o Vite faz proxy de `/api` para
`http://localhost:8080`.

### Usando a aplicação

1. Abra <http://localhost:5173>, clique em **Não tenho conta — registrar** e
   crie um usuário (senha de no mínimo 8 caracteres). Se o login com Google
   estiver configurado, o botão **Entrar com Google** também aparece —
   veja [Login com Google](#login-com-google).
2. **Novo formulário** → dê um título e adicione campos (texto, número,
   seleção, caixa de marcação). Campos de seleção pedem as opções separadas
   por vírgula.
3. **Salvar** e depois **Publicar**. A URL pública aparece no card.
4. Abra a URL pública numa aba anônima (ela não exige login), responda e
   envie.
5. Volte ao admin e clique em **Respostas**.

## Configuração

Três fontes, nesta ordem de precedência (a de baixo vence):

1. padrões embutidos no código
2. variáveis de ambiente
3. flags de linha de comando

| Flag                      | Variável de ambiente     | Padrão                                                        |
| ------------------------- | ------------------------ | ------------------------------------------------------------- |
| `--address`               | `ADDRESS`                | `localhost:8080`                                              |
| `--database-url`          | `DATABASE_URL`           | `postgres://forms:forms@localhost:5432/forms?sslmode=disable` |
| `--jwt-secret`            | `JWT_SECRET`             | `dev-secret-troque-isto`                                      |
| `--frontend-url`          | `FRONTEND_URL`           | `http://localhost:5173`                                       |
| `--google-client-id`      | `GOOGLE_CLIENT_ID`       | vazio (login com Google desligado)                            |
| `--google-client-secret`  | `GOOGLE_CLIENT_SECRET`   | vazio (login com Google desligado)                            |
| `--google-redirect-url`   | `GOOGLE_REDIRECT_URL`    | vazio (login com Google desligado)                            |

As três últimas são **opcionais**: sem elas o projeto sobe normalmente, o
botão "Entrar com Google" não aparece e as rotas `/auth/google*` respondem
`503`.

Flags e env podem ser combinados — aqui o banco vem do ambiente e o
endereço da flag:

```bash
DATABASE_URL=postgres://user:pass@host:5432/db go run ./cmd/server run --address=0.0.0.0:9000
```

Há um `.env.example` na raiz. O `.env` está no `.gitignore` e **não** é
commitado. O backend não lê o `.env` sozinho: carregue-o no shell antes de
rodar, se quiser usá-lo.

```bash
cp .env.example .env
set -a && . ./.env && set +a    # bash/zsh
make run
```

O frontend tem seu próprio `web/.env.example` (`VITE_API_URL`), opcional:
sem ele, o Vite faz proxy de `/api` para `http://localhost:8080`.

### Login com Google

O login por e-mail e senha funciona sem configuração nenhuma. Para ligar
também o login com Google:

**1. Criar as credenciais no Google Cloud Console**

1. Abra o [Google Cloud Console](https://console.cloud.google.com/) e
   selecione (ou crie) um projeto.
2. **APIs e serviços → Tela de permissão OAuth**: escolha o tipo
   *Externo*, preencha nome do app e e-mail de suporte. Enquanto o app
   estiver em modo *Teste*, só as contas listadas em **Usuários de teste**
   conseguem entrar — adicione a sua ali.
3. **APIs e serviços → Credenciais → Criar credenciais → ID do cliente
   OAuth**, tipo **Aplicativo da Web**.
4. Em **URIs de redirecionamento autorizados**, registre exatamente:

   ```
   http://localhost:8080/auth/google/callback
   ```

   Tem que ser idêntico ao `GOOGLE_REDIRECT_URL` (o Google compara string
   com string). Em produção, a mesma URL com o domínio real e `https`.
5. Copie o **client ID** e o **client secret**.

**2. Configurar o backend**

```bash
export GOOGLE_CLIENT_ID=...apps.googleusercontent.com
export GOOGLE_CLIENT_SECRET=...
export GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback
export FRONTEND_URL=http://localhost:5173   # já é o padrão
make run
```

O `.env.example` já traz essas variáveis. O `.env` está no `.gitignore` —
**o client secret nunca é commitado**.

**3. Usar**

Com as credenciais no ar, `GET /config` passa a responder
`{"googleAuth": true}` e o botão **Entrar com Google** aparece na tela de
login. O fluxo é:

```
frontend  →  GET /auth/google              → 302 para a tela do Google
Google    →  GET /auth/google/callback     → 302 para o frontend
frontend  →  /auth/callback#token=...      → guarda o JWT e vai para /forms
```

Se o e-mail da conta do Google já existir em `users`, as contas são
vinculadas (o `google_id` é preenchido e a senha continua valendo); senão,
um usuário novo é criado sem senha.

### Rodando o servidor diretamente

```bash
go run ./cmd/server run --address=localhost:8080
go run ./cmd/server migrate up
go run ./cmd/server openapi
go run ./cmd/server help
```

Ou compile um binário: `go build -o server ./cmd/server && ./server run`.

## Banco de dados e migrations

`docker-compose.yml` sobe um Postgres 16 (usuário/senha/banco `forms`, porta
5432) com volume persistente e healthcheck.

```bash
make db-up      # docker compose up -d --wait
make db-down    # para o container, preserva os dados
```

Para zerar o banco de verdade (apaga o volume):

```bash
docker compose down -v
```

As migrations são SQL versionado em `migrations/`, aplicadas pelo
[goose](https://github.com/pressly/goose) usado como **biblioteca** e
embutidas no binário com `go:embed` — o comando funciona de qualquer
diretório e não exige instalar o CLI do goose.

```bash
make migrate           # go run ./cmd/server migrate up
make migrate-status    # go run ./cmd/server migrate status
go run ./cmd/server migrate down   # desfaz a última
```

Para criar uma nova migration, adicione `migrations/0000N_nome.sql` com os
marcadores `-- +goose Up` e `-- +goose Down`. As quatro existentes são:

| Arquivo                  | Conteúdo                                   |
| ------------------------ | ------------------------------------------ |
| `00001_users.sql`        | `users`                                    |
| `00002_forms.sql`        | `forms`, `form_fields`                     |
| `00003_responses.sql`    | `responses`, `response_answers`            |
| `00004_google_auth.sql`  | `users.google_id`, `password_hash` anulável |

## Gerando a spec OpenAPI

A spec **não é escrita à mão**: o huma a deriva das structs Go de
request/response (tags `json`, `enum`, `minLength`, `format`, `doc`...) e das
`huma.Operation` registradas.

```bash
make openapi      # go run ./cmd/server openapi > api/openapi.yaml
```

O resultado fica em [`api/openapi.yaml`](api/openapi.yaml) e é commitado,
para que o frontend possa gerar o client sem subir o backend.

## Gerando o client TypeScript

```bash
make client       # regenera a spec e depois o client
# ou, dentro de web/:
npm run generate:client
```

Isso roda `openapi-typescript api/openapi.yaml -o web/src/api/schema.d.ts`.
O arquivo gerado **não é commitado** (está no `.gitignore`) — justamente para
garantir que ninguém o edite à mão. Os scripts `predev` e `prebuild` do
`package.json` o regeneram antes de `npm run dev` e `npm run build`.

`web/src/lib/api.ts` instancia o `openapi-fetch` com esses tipos; errar um
caminho, um body ou ler um campo inexistente vira erro de compilação.

## Comandos do Makefile

| Comando             | O que faz                                             |
| ------------------- | ----------------------------------------------------- |
| `make db-up`        | sobe o Postgres e espera ficar pronto                 |
| `make db-down`      | derruba o Postgres (mantém os dados)                  |
| `make migrate`      | aplica as migrations pendentes                        |
| `make migrate-status`| mostra o estado das migrations                       |
| `make run`          | sobe o backend em `localhost:8080`                    |
| `make openapi`      | gera `api/openapi.yaml` a partir do código Go         |
| `make client`       | gera a spec e o client TypeScript                     |
| `make test`         | testes unitários do Go                                |
| `make smoke`        | testa a API ponta a ponta (backend precisa estar no ar)|
| `make dev`          | sobe banco + migrations e lembra os dois comandos     |
| `make help`         | lista os comandos                                     |

## Testes

```bash
make test     # go test ./... — validação da submissão
make smoke    # python3 scripts/smoke.py — 24 asserções contra a API no ar
```

O `smoke.py` usa só a biblioteca padrão do Python 3: cria usuário, monta um
formulário com os quatro tipos de campo, publica, envia respostas válidas e
inválidas, confere os status e verifica o isolamento entre usuários.

## Estrutura do projeto

```
cmd/server/          comando `server` (run, migrate, openapi)
internal/config/     resolução de flags + variáveis de ambiente
internal/database/   conexão com Postgres e execução das migrations
internal/auth/       bcrypt e emissão/validação de JWT
internal/store/      acesso ao banco (SQL escrito à mão)
internal/submission/ validação da resposta contra a definição do formulário
internal/httpapi/    roteador chi + operações huma
migrations/          SQL versionado, embutido no binário
api/openapi.yaml     spec gerada do código
scripts/smoke.py     teste ponta a ponta da API
web/                 frontend React + Vite
```

### Rotas

| Método | Rota                                | Auth | O que faz                       |
| ------ | ----------------------------------- | ---- | ------------------------------- |
| GET    | `/health`                           | não  | status da API e do banco        |
| GET    | `/config`                           | não  | diz se o login com Google existe |
| POST   | `/auth/register`                    | não  | cria conta de admin             |
| POST   | `/auth/login`                       | não  | devolve o JWT                   |
| GET    | `/auth/google`                      | não  | redireciona para o Google (503 se não configurado) |
| GET    | `/auth/google/callback`             | não  | volta do Google e emite o JWT   |
| GET    | `/auth/me`                          | sim  | usuário autenticado             |
| GET    | `/forms`                            | sim  | lista os formulários do usuário |
| POST   | `/forms`                            | sim  | cria formulário (draft)         |
| GET    | `/forms/{formId}`                   | sim  | detalhe com os campos           |
| PUT    | `/forms/{formId}`                   | sim  | atualiza dados e campos         |
| DELETE | `/forms/{formId}`                   | sim  | remove formulário e respostas   |
| POST   | `/forms/{formId}/publish`           | sim  | publica e gera o slug           |
| POST   | `/forms/{formId}/unpublish`         | sim  | volta para draft                |
| GET    | `/forms/{formId}/responses`         | sim  | respostas recebidas             |
| GET    | `/public/forms/{slug}`              | não  | abre formulário publicado       |
| POST   | `/public/forms/{slug}/responses`    | não  | envia uma resposta              |

A página pública que o usuário final abre é `/f/{slug}` **no frontend**; ela
consome `/public/forms/{slug}` no backend. Os caminhos são diferentes de
propósito, para que a rota da SPA e a rota da API não disputem o mesmo
prefixo.

## Decisões de arquitetura

### Modelagem dos formulários

Um formulário é uma linha em `forms` e cada campo é uma linha em
`form_fields` (`position`, `label`, `type`, `required`, `options` em JSONB).
A alternativa seria guardar a definição inteira como um JSON só.

Escolhi o modelo relacional porque:

- o banco garante as invariantes com `CHECK` (os quatro tipos de campo) e
  `UNIQUE (form_id, position)` (ordem sem duplicatas);
- validar uma submissão vira um `SELECT` simples da definição;
- a listagem do admin não precisa carregar e parsear JSON para mostrar o
  formulário.

`options` ficou em JSONB porque é uma lista curta de strings que só tem
sentido dentro do campo — uma terceira tabela só para isso não pagaria o
custo.

**Estados**: `status` é `draft` ou `published`, com um `CHECK` garantindo que
publicado tem slug e draft não tem. O slug é gerado ao publicar: título
"achatado" mais um sufixo aleatório de `crypto/rand`, com retry em caso de
colisão. O sufixo evita que alguém descubra formulários alheios adivinhando
o título.

**Edição substitui os campos**: salvar um formulário apaga e reinsere a lista
de campos numa transação. É o que a tela de edição precisa (ela sempre manda
a definição completa) e evita a complexidade de calcular um diff.

**Respostas guardam uma cópia do campo**: `response_answers` tem `label` e
`type` copiados do campo no momento do envio, sem chave estrangeira para
`form_fields`. Como editar substitui os campos, um `ON DELETE CASCADE`
apagaria respostas antigas; com a cópia, o histórico continua legível mesmo
depois de o formulário mudar. O valor é sempre texto, com o `type` dizendo
como interpretá-lo.

### Validação da submissão

Fica em `internal/submission`, sem dependência de HTTP ou de banco: recebe a
definição e o mapa de respostas, devolve os valores normalizados ou a lista
de erros. Valida obrigatórios, tipos, opções válidas do select e recusa
chaves que não correspondem a nenhum campo do formulário. Devolve **todos**
os erros de uma vez (HTTP 422 no formato RFC 7807, com
`location: body.answers.<fieldId>`), e a página pública usa isso para marcar
cada input errado.

Um `<input type="number">` entrega string, então números são aceitos tanto
como número JSON quanto como texto numérico.

### Escolha das bibliotecas

| Biblioteca | Por quê |
| ---------- | ------- |
| **chi** | roteador fino sobre `net/http`, sem framework próprio; os middlewares padrão (RequestID, Logger, Recoverer) vêm prontos |
| **huma v2** | gera a spec OpenAPI 3 a partir das structs Go e valida a entrada com ela. Era o requisito central: a spec nunca sai do ar em relação ao código |
| **database/sql + pgx (stdlib)** | um único `*sql.DB` serve para as queries e para o goose. SQL à mão, sem ORM — o modelo é pequeno e fica mais fácil de ler e revisar |
| **goose como biblioteca** | migrations embutidas com `go:embed`; dispensa instalar um CLI e funciona de qualquer diretório |
| **golang-jwt/jwt v5** | padrão de fato; `ParseWithClaims` com `WithValidMethods` bloqueia a troca de algoritmo para `none` |
| **bcrypt** | pedido no enunciado; o salt vai dentro do próprio hash |
| **openapi-typescript + openapi-fetch** | gera os tipos da spec e dá um cliente fino e tipado, sem camada de cache própria (quem cuida disso é o TanStack Query) |
| **TanStack Query** | estado remoto, cache e invalidação após mutações |
| **TanStack Router** | em *code-based routing*: o projeto já tem um passo de geração (o client), e não quis um segundo |

### Autenticação

Registro e login devolvem um JWT HS256 com o id do usuário no `subject` e
validade de 24h, guardado no `localStorage` e enviado em
`Authorization: Bearer`. O middleware exige token apenas nas operações
marcadas com `Security` no huma — ou seja, a lista de rotas protegidas é
exatamente a que a spec OpenAPI publica, sem duas listas para manter em
sincronia.

**Login com Google (OAuth2, Authorization Code + PKCE).** `GET /auth/google`
sorteia um `state` (anti-CSRF) e um `code_verifier` (PKCE), guarda os dois
num cookie `HttpOnly; SameSite=Lax` que vale 10 minutos e redireciona para o
consentimento. `GET /auth/google/callback` confere o `state` contra o
cookie, troca o código pelo token revelando o `code_verifier`, lê o perfil
em `/oauth2/v3/userinfo` e **exige `email_verified`** — vincular contas pelo
e-mail só é seguro com essa garantia. No fim emite exatamente o mesmo JWT do
login por senha: nada muda no middleware nem nas rotas de formulários.

PKCE não é obrigatório para um cliente com secret, mas custa poucas linhas
(o próprio `golang.org/x/oauth2` fornece `GenerateVerifier` e
`S256ChallengeOption`) e tira valor de um código de autorização interceptado.

**Token devolvido no fragmento da URL.** A callback redireciona para
`FRONTEND_URL/auth/callback#token=...`. Escolhi o fragmento em vez de query
string porque o fragmento **não é enviado ao servidor**: o token não cai em
log de acesso nem em header `Referer`. A página de callback lê o fragmento,
guarda o token e limpa a URL com `history.replaceState`, para o token não
ficar no histórico. A alternativa — devolver o token num cookie `HttpOnly` —
seria mais segura (ver limitações), mas exigiria mudar também o login por
senha, que hoje usa `localStorage`.

**Credenciais ausentes não quebram o projeto.** As rotas do Google são
sempre registradas (a spec OpenAPI é estável), mas respondem `503` quando
faltam credenciais, e `GET /config` diz ao frontend para esconder o botão —
assim ninguém esbarra num caminho sem saída.

**Uma conta, duas formas de entrar.** `users.google_id` é `UNIQUE` e
`password_hash` virou anulável, com um `CHECK` garantindo que toda conta tem
pelo menos uma forma de login. A busca é por `google_id` primeiro (o `sub` do
Google é estável, o e-mail pode mudar) e só então por e-mail, caso em que as
contas são vinculadas.

Acessar um formulário de outro usuário devolve **404**, não 403: assim a API
não confirma a existência de ids alheios.

## Limitações e decisões por falta de tempo

Este projeto foi feito sob um limite de tempo. O que ficou de fora, e como
eu faria:

### O que ficou de fora do login com Google

Implementado, mas com cortes conscientes:

- **Sem desvincular a conta do Google** pela interface: o `google_id` entra
  no primeiro login e não há tela para removê-lo.
- **O token do Google é descartado** depois de ler o perfil. Não guardamos
  access token nem refresh token porque não chamamos nenhuma outra API do
  Google — se um dia precisássemos (Drive, Calendar), aí sim valeria
  persistir, criptografado.
- **Sem `id_token` verificado localmente.** Lemos o perfil chamando o
  `userinfo` do Google por HTTPS, o que é suficiente aqui. Validar o
  `id_token` pela JWKS do Google pouparia essa chamada de rede.
- **Erros da callback viram texto no fragmento** (`#error=...`) e aparecem
  numa página genérica. Um mapeamento de códigos para mensagens por caso
  seria mais amigável.

### Outros itens fora do escopo

- **Sem refresh token.** Continua fora do escopo — vale tanto para o login
  por senha quanto para o do Google. O JWT vale 24h e, quando expira, é preciso logar de
  novo. Um refresh token em cookie `HttpOnly` seria o próximo passo, junto
  com tirar o token de acesso do `localStorage`.
- **Token no `localStorage`.** Simples e suficiente para o teste, mas
  vulnerável a XSS. O caminho seguro é cookie `HttpOnly` + `SameSite`.
- **CORS liberado para qualquer origem.** Em produção seria uma lista de
  origens configurável por env var.
- **Despublicar e republicar gera um slug novo**, invalidando o link
  anterior. Preservar o slug antigo é uma linha de SQL, mas a decisão de
  produto (link permanente vs. link revogável) não era óbvia, então mantive
  o comportamento mais previsível: draft nunca tem slug.
- **Sem paginação** na lista de formulários nem na de respostas. Com volume
  real, ambas precisariam de `LIMIT/OFFSET` ou cursor.
- **Sem exportação CSV** das respostas — seria o pedido mais provável de um
  usuário real logo depois desta tela.
- **Testes**: há testes unitários da validação (a regra de negócio central) e
  um smoke test da API inteira, mas não há testes dos handlers HTTP nem do
  `store` com banco de testes, nem nada no frontend.
- **Sem Dockerfile do backend nem docker-compose da stack completa.** O
  compose sobe só o Postgres; backend e frontend rodam na máquina. Empacotar
  os três exigiria build multi-stage e um tempo que não havia.
- **Reordenar campos** só é possível removendo e recriando: a tela não tem
  drag-and-drop. A ordem no banco já é explícita (`position`), então é só
  trabalho de frontend.
- **Sem rate limiting** nas rotas públicas de submissão, que são o alvo
  óbvio de abuso.
