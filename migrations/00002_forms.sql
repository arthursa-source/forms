-- Migration 2: formulários e seus campos.
--
-- Modelagem relacional (uma linha por campo) em vez de um JSON único com a
-- definição: assim o banco garante tipos válidos e ordem dos campos, e a
-- validação da submissão lê a definição com um SELECT simples.

-- +goose Up
CREATE TABLE forms (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title        TEXT        NOT NULL,
    description  TEXT        NOT NULL DEFAULT '',
    status       TEXT        NOT NULL DEFAULT 'draft'
                             CHECK (status IN ('draft', 'published')),
    -- slug só existe depois de publicar; UNIQUE permite vários NULLs.
    slug         TEXT        UNIQUE,
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Um formulário publicado precisa de slug; um draft nunca tem.
    CONSTRAINT slug_quando_publicado CHECK (
        (status = 'published' AND slug IS NOT NULL)
        OR (status = 'draft' AND slug IS NULL)
    )
);

CREATE INDEX forms_user_id_idx ON forms (user_id);

CREATE TABLE form_fields (
    id       BIGSERIAL PRIMARY KEY,
    form_id  BIGINT  NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    label    TEXT    NOT NULL,
    type     TEXT    NOT NULL CHECK (type IN ('text', 'number', 'select', 'checkbox')),
    required BOOLEAN NOT NULL DEFAULT FALSE,
    -- Opções do select, como array JSON. Vazio para os outros tipos.
    options  JSONB   NOT NULL DEFAULT '[]'::jsonb,
    UNIQUE (form_id, position)
);

CREATE INDEX form_fields_form_id_idx ON form_fields (form_id);

-- +goose Down
DROP TABLE form_fields;
DROP TABLE forms;
