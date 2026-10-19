-- Migration 1: tabela de usuários do admin.
-- As linhas de anotação do goose delimitam o que sobe e o que desce.

-- +goose Up
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Login busca por e-mail; o UNIQUE acima já cria o índice necessário.

-- +goose Down
DROP TABLE users;
